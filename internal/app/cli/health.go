package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

func newHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check the health of the FUSE server",
		RunE: func(_ *cobra.Command, _ []string) error {
			url := fmt.Sprintf("http://127.0.0.1:%s/health", port)
			client := &http.Client{Timeout: 3 * time.Second}

			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("health check returned status %d", resp.StatusCode)
			}

			fmt.Println("OK")
			return nil
		},
	}
}
