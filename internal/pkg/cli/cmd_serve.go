package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alesr/disco/internal/app"
	"github.com/alesr/disco/internal/transport/local"
	"github.com/spf13/cobra"
)

func (a *Application) newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run local warm daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			fmt.Fprintf(os.Stderr, "starting local daemon on %s\n", local.DefaultSocketPath)
			return a.deps.RunServe(ctx, app.ServeOptions{StyleGuideDir: a.deps.StyleGuideDir})
		},
	}
}
