package ingest

import (
	"context"
	"fmt"
	"os"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/plugin"
	"github.com/assaio/assaio/internal/store"
)

// ingestPlugins runs every configured exec plugin and inserts the records it emits. A
// plugin that fails to resolve, times out, or fails its handshake counts as Failed and
// does not abort the run; the remaining plugins still get processed. The failure reason is
// written to stderr so a misconfigured plugin does not fail silently -- otherwise the run
// summary shows only "failed=1" with no way to tell a bad command from a timeout.
func ingestPlugins(ctx context.Context, st *store.Store, plugins []config.PluginConfig) ([]Result, error) {
	results := make([]Result, 0, len(plugins))
	for _, pc := range plugins {
		res := Result{Tool: "plugin:" + pc.Name, Files: 1}
		cfg, err := plugin.Resolve(pc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: plugin %q failed: %v\n", pc.Name, err)
			res.Failed = 1
			results = append(results, res)
			continue
		}
		recs, stats, err := plugin.Run(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: plugin %q failed: %v\n", pc.Name, err)
			res.Failed = 1
			results = append(results, res)
			continue
		}
		res.Records = stats.Records
		res.Skipped = stats.Skipped
		n, err := st.Insert(ctx, recs)
		if err != nil {
			return results, err
		}
		res.Inserted = n
		results = append(results, res)
	}
	return results, nil
}
