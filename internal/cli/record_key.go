package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/recordsecret"
	"github.com/hapyco/dygo/internal/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

func newRecordKeyCommand(ctx context.Context, stdout io.Writer) *cobra.Command {
	parent := &cobra.Command{Use: "record-key", Short: "Manage Record encryption keys", Args: cobra.NoArgs}
	for _, operation := range []string{"init", "rotate"} {
		envName := "development"
		yes := false
		offline := false
		dryRun := false
		cmd := &cobra.Command{Use: operation, Short: operation + " Record encryption key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if operation == "rotate" && !offline {
				return errors.New("stop servers and workers, then pass --offline to confirm")
			}
			env, root, url, err := databaseInputs(envName)
			if err != nil {
				return err
			}
			if _, err = fmt.Fprintf(stdout, "Record key %s (%s)\nproject: %s\n", operation, env, root); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if !yes {
				return errors.New("review the target, then pass --yes")
			}
			conn, err := pgx.Connect(ctx, url)
			if err != nil {
				return db.SanitizeDatabaseError("connect Record key database", url, err)
			}
			defer conn.Close(context.Background())
			var locked bool
			if err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", db.RecordKeyLock).Scan(&locked); err != nil {
				return errors.New("acquire Record key lock failed")
			}
			if !locked {
				return errors.New("another Record key command is running")
			}
			store := secrets.NewStore(root)
			if operation == "init" {
				if err = recordsecret.Init(store, env); err != nil {
					return err
				}
			} else {
				ring, err := recordsecret.BeginRotation(store, env)
				if err != nil {
					return err
				}
				count, err := db.RotateRecordSecrets(ctx, conn, ring)
				if err != nil {
					return err
				}
				// A full second pass verifies every live ciphertext and key ID before finalization.
				remaining, err := db.RotateRecordSecrets(ctx, conn, ring)
				if err != nil {
					return err
				}
				if remaining != 0 {
					return errors.New("Record secrets changed during offline verification; stop writers and resume")
				}
				if err = recordsecret.FinishRotation(store, env, ring); err != nil {
					return err
				}
				if _, err = fmt.Fprintf(stdout, "re-encrypted values: %d\nold keys retained for backup recovery\n", count); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(stdout, "Record key "+operation+" complete")
			return err
		}}
		cmd.Flags().StringVar(&envName, "env", envName, "Environment: development, staging, or production")
		cmd.Flags().BoolVar(&yes, "yes", false, "Confirm the displayed target")
		cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the target without changes")
		if operation == "rotate" {
			cmd.Flags().BoolVar(&offline, "offline", false, "Confirm servers and workers are stopped")
		}
		parent.AddCommand(cmd)
	}
	return parent
}
