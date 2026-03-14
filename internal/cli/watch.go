package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rafamoreira/trove/internal/vault"
)

func newWatchCmd(opts Options) *cobra.Command {
	var interval int
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch the vault and sync on a polling interval",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("interval") {
				if interval <= 0 {
					return fmt.Errorf("--interval must be greater than zero")
				}
				rt.cfg.SyncDebounceSeconds = interval
			}
			return runWatchLoop(rt)
		},
	}
	cmd.Flags().IntVar(&interval, "interval", 0,
		"polling interval in seconds (overrides sync_debounce_seconds in config)")
	return cmd
}

func runWatchLoop(rt *runtime) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigs)

	d := time.Duration(rt.cfg.SyncDebounceSeconds) * time.Second
	if d <= 0 {
		d = 30 * time.Second
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	fmt.Fprintf(rt.stderr, "watch: started, interval=%s, vault=%s\n", d, rt.vault.Path)

	runWatchCycle(rt, time.Now())

	for {
		select {
		case <-sigs:
			fmt.Fprintf(rt.stderr, "watch: received signal, shutting down\n")
			return nil
		case t := <-ticker.C:
			runWatchCycle(rt, t)
		}
	}
}

func runWatchCycle(rt *runtime, t time.Time) {
	fmt.Fprintf(rt.stderr, "watch: cycle at %s\n", t.UTC().Format(time.RFC3339))

	if !rt.cfg.AutoSync {
		return
	}

	if vault.GitAvailable() && rt.vault.GitIsRepo() && rt.cfg.GitRemote != "" {
		if err := rt.vault.GitPull(rt.cfg.GitRemote, rt.cfg.GitBranch); err != nil {
			fmt.Fprintf(rt.stderr, "watch: pull warning: %v\n", err)
			return
		}
		fmt.Fprintf(rt.stderr, "watch: pull ok\n")
	}

	committed, pushed, warnings, err := rt.vault.SyncNow()
	if err != nil {
		fmt.Fprintf(rt.stderr, "watch: sync error: %v\n", err)
		return
	}
	for _, w := range warnings {
		fmt.Fprintf(rt.stderr, "watch: warning: %s\n", w.Message)
	}
	if committed {
		fmt.Fprintf(rt.stderr, "watch: committed local changes\n")
	}
	if pushed {
		fmt.Fprintf(rt.stderr, "watch: pushed to remote\n")
	}
}
