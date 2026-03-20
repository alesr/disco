package cli

import (
	"context"
	"fmt"

	"github.com/alesr/disco/internal/app"
	"github.com/alesr/disco/internal/review"
	"github.com/alesr/disco/internal/transport/local"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

type DaemonManager interface {
	Start() error
	Stop() error
	Install() error
	Uninstall() error
	Status() (service.Status, error)
}

type Dependencies struct {
	StyleGuideDir    string
	DaemonManager    DaemonManager
	NewDaemonManager func() (DaemonManager, error)

	RunServe        func(context.Context, app.ServeOptions) error
	RunStatus       func(context.Context, app.StatusOptions) (local.HealthResponse, error)
	RunReviewStream func(context.Context, app.ReviewOptions, func(review.ReviewEvent) error) (string, error)
}

type Application struct {
	deps          Dependencies
	root          *cobra.Command
	daemonManager DaemonManager
}

func NewApplication(deps Dependencies) *Application {
	app := &Application{deps: deps}
	app.root = app.newRootCommand()
	return app
}

func (a *Application) Execute(args []string) error {
	a.root.SetArgs(args)
	if err := a.root.Execute(); err != nil {
		return err
	}
	return nil
}

func (a *Application) newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "disco",
		Short: "Local-first style-guide review daemon",
	}

	rootCmd.AddCommand(a.newReviewCommand())
	rootCmd.AddCommand(a.newServeCommand())
	rootCmd.AddCommand(a.newStatusCommand())
	rootCmd.AddCommand(a.newDaemonCommand())
	rootCmd.AddCommand(newCompletionCommand(rootCmd))
	return rootCmd
}

func (a *Application) getDaemonManager() (DaemonManager, error) {
	if a.daemonManager != nil {
		return a.daemonManager, nil
	}

	if a.deps.DaemonManager != nil {
		a.daemonManager = a.deps.DaemonManager
		return a.daemonManager, nil
	}

	if a.deps.NewDaemonManager == nil {
		return nil, fmt.Errorf("could not initialize daemon manager: factory is nil")
	}

	manager, err := a.deps.NewDaemonManager()
	if err != nil {
		return nil, err
	}

	a.daemonManager = manager
	return a.daemonManager, nil
}
