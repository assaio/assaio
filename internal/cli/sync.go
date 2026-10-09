package cli

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/server"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func newSyncCmd() *cobra.Command {
	var serverURL, token, member, since string
	var printMemberDigest bool
	c := &cobra.Command{
		Use:   "sync",
		Short: "Push local usage records to a team server",
		Long: `Export local usage records and push them to a team server started with
'assaio-agent serve'. Set ASSAIO_SYNC_IDENTITY_KEY to a private 32-byte hex key;
the member label is keyed locally and the branch never leaves this machine.
--member selects the stable local input to that digest, never a name sent as-is.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if printMemberDigest {
				if err := resolveSyncFlags(cmd, &serverURL, &token, &member); err != nil {
					return err
				}
				digest, err := syncMemberDigest(member)
				if err != nil {
					return err
				}
				cmd.Println(digest)
				return nil
			}
			return runSync(cmd, &serverURL, &token, &member, &since)
		},
	}
	c.Flags().StringVar(&serverURL, "server", "", "team server base URL, e.g. http://localhost:8787 (required; also ASSAIO_SYNC_SERVER)")
	c.Flags().StringVar(&token, "token", "", "per-member bearer token (required; also ASSAIO_SYNC_TOKEN)")
	c.Flags().StringVar(&member, "member", "", "stable local member input for the keyed digest; never sent as-is")
	c.Flags().StringVar(&since, "since", "30d", "how far back to export local records, e.g. 30d")
	c.Flags().BoolVar(&printMemberDigest, "print-member-digest", false, "print the v2 member digest locally for server setup or migration")
	return c
}

func runSync(cmd *cobra.Command, serverURL, token, member, since *string) error {
	if err := resolveSyncFlags(cmd, serverURL, token, member); err != nil {
		return err
	}
	if *serverURL == "" {
		return errors.New("--server is required")
	}
	if *token == "" {
		return errors.New("--token is required")
	}
	memberID, err := syncMemberDigest(*member)
	if err != nil {
		return err
	}
	if isCleartextRemote(*serverURL) {
		cmd.PrintErrln("warning: --server is plaintext http:// to a non-localhost host -- the token and usage data are sent in cleartext.")
	}

	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}

	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	// Ctrl-C (SIGINT) or a process manager's stop signal (SIGTERM) cancels ctx, aborting
	// an in-flight export or push instead of leaving sync unresponsive until it finishes
	// or the process is killed -- mirrors serve.go's own signal wiring.
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	recs, err := st.Export(ctx, start)
	if err != nil {
		return err
	}
	if err := warnPooledNames(cmd, st, recs); err != nil {
		return err
	}

	result, err := pushUsage(ctx, *serverURL, *token, memberID, recs)
	if err != nil {
		return err
	}
	cmd.Printf("synced as %s: sent %d, server inserted %d (of %d received)\n",
		memberID, len(recs), result.Inserted, result.Received)
	return nil
}

// warnPooledNames names the projects in recs that more than one local repository shares. The
// server receives a project's name and never the repository behind it, so it counts their
// usage as one project's.
func warnPooledNames(cmd *cobra.Command, st *store.Store, recs []usage.Record) error {
	split, err := st.SplitLabels(cmd.Context())
	if err != nil {
		return err
	}
	sent := make(map[string]bool, len(recs))
	for i := range recs {
		sent[recs[i].Project] = true
	}
	var pooled []string
	for _, name := range split {
		if sent[name] {
			pooled = append(pooled, name)
		}
	}
	if len(pooled) > 0 {
		cmd.PrintErrf("warning: more than one repository on this machine is named %s; "+
			"the team server counts each of these names as one project.\n", strings.Join(pooled, ", "))
	}
	return nil
}

// isCleartextRemote reports whether serverURL would send the bearer token and usage
// payload in the clear: plaintext http:// to a host other than "localhost" or a loopback
// IP (the whole 127.0.0.0/8 range, or ::1), where anything on the network path between
// here and there could read both. A malformed serverURL is not this function's concern
// -- pushUsage will fail on it with its own clear error -- so a parse error yields false
// here.
func isCleartextRemote(serverURL string) bool {
	u, err := url.Parse(serverURL)
	if err != nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return false
	}
	return true
}

// resolveSyncFlags fills server/token/member from config when the caller did not
// override them on the command line; an unset config value never blanks out a flag's
// own default.
func resolveSyncFlags(cmd *cobra.Command, serverURL, token, member *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("server") && cfg.Sync.Server != "" {
		*serverURL = cfg.Sync.Server
	}
	if !cmd.Flags().Changed("token") && cfg.Sync.Token != "" {
		*token = cfg.Sync.Token
	}
	if !cmd.Flags().Changed("member") && cfg.Sync.Member != "" {
		*member = cfg.Sync.Member
	}
	return nil
}

// syncMemberDigest binds the local member input to a client-held key. The server
// sees only the digest; losing or rotating the key requires an explicit rekey.
func syncMemberDigest(explicit string) (string, error) {
	key, err := hex.DecodeString(os.Getenv("ASSAIO_SYNC_IDENTITY_KEY"))
	if err != nil || len(key) != 32 {
		return "", errors.New("ASSAIO_SYNC_IDENTITY_KEY must be a private 32-byte hex key; keep it for future syncs and backups")
	}
	identity := explicit
	if identity == "" {
		host, _ := os.Hostname()
		who := ""
		if u, err := user.Current(); err == nil {
			who = u.Username
		}
		identity = host + ":" + who
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("assaio-sync-member-v2\x00" + identity))
	digest := "member-v2-" + hex.EncodeToString(mac.Sum(nil)[:16])
	if err := server.ValidateSyncMemberDigest(digest); err != nil {
		return "", fmt.Errorf("member digest: %w", err)
	}
	return digest, nil
}
