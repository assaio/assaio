package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/server"
)

// memberHexLen is the sync-member pseudonym's hex length ("member-a1b2"): a short sender
// label, distinct from internal/report.Pseudonym's 10-hex report id.
const memberHexLen = 4

func newSyncCmd() *cobra.Command {
	var serverURL, token, member, since string
	c := &cobra.Command{
		Use:   "sync",
		Short: "Push local usage records to a team server",
		Long: `Export local usage records and push them to a team server started with
'assaio-agent serve'. By default the sender is identified by a pseudonym derived from
this machine's hostname and OS user, not a real name -- pass --member to opt in to
self-identifying.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSync(cmd, &serverURL, &token, &member, &since)
		},
	}
	c.Flags().StringVar(&serverURL, "server", "", "team server base URL, e.g. http://localhost:8787 (required; also ASSAIO_SYNC_SERVER)")
	c.Flags().StringVar(&token, "token", "", "shared bearer token (required; also ASSAIO_SYNC_TOKEN)")
	c.Flags().StringVar(&member, "member", "", "self-identify with this name instead of an auto-derived pseudonym (opt-in)")
	c.Flags().StringVar(&since, "since", "30d", "how far back to export local records, e.g. 30d")
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
	memberID := resolveMember(*member)
	if err := server.ValidateMember(memberID); err != nil {
		return fmt.Errorf("--member: %w", err)
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

	result, err := pushUsage(ctx, *serverURL, *token, memberID, recs)
	if err != nil {
		return err
	}
	cmd.Printf("synced as %s: sent %d, server inserted %d (of %d received)\n",
		memberID, len(recs), result.Inserted, result.Received)
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

// resolveMember returns explicit if the caller opted in to self-identifying, else a
// stable pseudonym derived from this machine's hostname and OS user -- pseudonymized is
// assaio's default privacy mode (AGENTS.md), so an unconfigured sync stays anonymous.
func resolveMember(explicit string) string {
	if explicit != "" {
		return explicit
	}
	host, _ := os.Hostname()
	who := ""
	if u, err := user.Current(); err == nil {
		who = u.Username
	}
	sum := sha256.Sum256([]byte(host + ":" + who))
	return "member-" + hex.EncodeToString(sum[:])[:memberHexLen]
}
