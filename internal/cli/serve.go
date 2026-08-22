package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/server"
	"github.com/assaio/assaio/internal/store"
)

// serveDefaultDBName is the central store's default filename, kept distinct from the
// local agent's assaio.db (internal/paths.DBPath) so `serve` never opens a teammate's
// local usage store by accident.
const serveDefaultDBName = "assaio-server.db"

func newServeCmd() *cobra.Command {
	var addr, token, dbPath string
	c := &cobra.Command{
		Use:   "serve",
		Short: "Run the team server: collect pushed usage and serve the aggregated dashboard",
		Long: `Run assaio's team-server MVP: a self-hosted HTTP endpoint that collects usage
pushed by teammates' 'assaio-agent sync' runs and serves it back as one aggregated,
pseudonymized-by-default Assay dashboard.

This is a no-TLS MVP -- not production-hardened. Run it behind a reverse proxy on a network
you trust; see internal/server's package doc for the exact boundary.

SECURITY BOUNDARY (read before exposing beyond localhost): there is no TLS -- put a reverse
proxy in front of it. Every route but the /healthz probe requires the bearer token, the dashboard included. With a
single shared --token any holder can push usage under any member name; configure
server.members (one secret per member) and the server decides who a request is from the secret
instead. --addr defaults to loopback (127.0.0.1) so the server is reachable only from this
machine unless you deliberately choose a wider address.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, &addr, &token, &dbPath)
		},
	}
	c.Flags().StringVar(&addr, "addr", "127.0.0.1:8787", "listen address (loopback by default; widen deliberately)")
	c.Flags().StringVar(&token, "token", "", "shared bearer token clients must present (required; also ASSAIO_SERVER_TOKEN)")
	c.Flags().StringVar(&dbPath, "db", "", "central store path (default: "+serveDefaultDBName+" under the data dir)")
	return c
}

func runServe(cmd *cobra.Command, addr, token, dbPath *string) error {
	if err := resolveServeFlags(cmd, addr, token); err != nil {
		return err
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	members := server.Members(cfg.Server.Members)
	if err := members.Validate(); err != nil {
		return fmt.Errorf("server.members: %w", err)
	}
	// With per-member tokens configured the shared secret authenticates nothing, so requiring
	// one would be a secret an operator has to invent and then keep.
	if err := requireServeSecret(*token, members); err != nil {
		return err
	}

	resolvedDB, err := resolveServeDBPath(*dbPath)
	if err != nil {
		return err
	}
	if err := ensureParent(resolvedDB); err != nil {
		return err
	}
	st, err := store.Open(resolvedDB)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	srv := server.New(st, *token, server.BuildDashboard).
		WithMembers(members).
		WithRateLimit(cfg.Server.RateLimitPerMinute)

	cmd.Printf("assaio team server listening on %s (db: %s)\n", *addr, resolvedDB)
	cmd.Printf("identity: %s\n", srv.Identity())
	if srv.Identity() == server.ClientAsserted {
		cmd.Println("  warning: any holder of the shared token can push usage under any member name.")
		cmd.Println("  Configure server.members (one secret per member) to have the server decide who a request is.")
	}
	cmd.Println("security note: no TLS -- run behind a reverse proxy or on a trusted network.")
	cmd.Println("Every route but /healthz requires the bearer token, the dashboard included.")

	// Ctrl-C (SIGINT) or a process manager's stop signal (SIGTERM) cancels ctx, which
	// srv.Run treats as a graceful-shutdown request.
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return srv.Run(ctx, *addr)
}

// resolveServeFlags fills addr/token from config when the caller did not override them
// on the command line; an unset config value never blanks out a flag's own default.
func resolveServeFlags(cmd *cobra.Command, addr, token *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("addr") && cfg.Server.Addr != "" {
		*addr = cfg.Server.Addr
	}
	if !cmd.Flags().Changed("token") && cfg.Server.Token != "" {
		*token = cfg.Server.Token
	}
	return nil
}

// resolveServeDBPath returns dbPath if set, else the default central-store path under
// the data dir.
func resolveServeDBPath(dbPath string) (string, error) {
	if dbPath != "" {
		return dbPath, nil
	}
	dir, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, serveDefaultDBName), nil
}

// requireServeSecret refuses to start a server nobody has to authenticate to. Either mode
// satisfies it; what it will not accept is neither, or a shared secret short enough to guess.
func requireServeSecret(token string, members server.Members) error {
	if members.Mode() == server.ServerDerived {
		return nil
	}
	if token == "" {
		return errors.New("--token or server.members is required: refusing to run an open server (see AGENTS.md honesty rules)")
	}
	if len(token) < server.MinTokenBytes {
		return fmt.Errorf("--token must be at least %d characters: a secret short enough to guess is not one", server.MinTokenBytes)
	}
	return nil
}
