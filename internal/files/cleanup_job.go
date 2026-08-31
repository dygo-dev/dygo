package files

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hapyco/dygo/pkg/dygo"
)

// CleanupJobRegistrar registers the framework-owned post-commit blob cleanup.
// The payload contains a path produced by the configured local driver.
func CleanupJobRegistrar() dygo.JobRegistrar {
	return func(registry dygo.JobRegistry) error {
		return registry.RegisterJob(coreApp, cleanupJob, func(ctx context.Context, execution dygo.JobExecution) error {
			var payload cleanupPayload
			if err := json.Unmarshal(execution.Payload, &payload); err != nil || payload.Path == "" {
				return fmt.Errorf("file cleanup payload is invalid")
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := os.Remove(payload.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove file blob: %w", err)
			}
			return nil
		})
	}
}
