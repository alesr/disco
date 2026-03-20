package cli

import (
	"context"
	"fmt"

	"github.com/alesr/disco/internal/app"
	"github.com/spf13/cobra"
)

func (a *Application) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check local daemon health via socket",
		RunE: func(_ *cobra.Command, _ []string) error {
			health, err := a.deps.RunStatus(context.Background(), app.StatusOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("daemon status: %s\n", health.Status)
			return nil
		},
	}
}
