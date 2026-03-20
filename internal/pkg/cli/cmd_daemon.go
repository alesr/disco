package cli

import (
	"fmt"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

func (a *Application) newDaemonCommand() *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage daemon service lifecycle",
	}

	daemonCmd.AddCommand(a.newDaemonStartCommand())
	daemonCmd.AddCommand(a.newDaemonStopCommand())
	daemonCmd.AddCommand(a.newDaemonStatusCommand())
	daemonCmd.AddCommand(a.newDaemonInstallCommand())
	daemonCmd.AddCommand(a.newDaemonUninstallCommand())
	return daemonCmd
}

func (a *Application) newDaemonStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start daemon service (auto-installs when missing)",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := a.getDaemonManager()
			if err != nil {
				return err
			}

			if err := manager.Start(); err != nil {
				return err
			}

			fmt.Println("daemon service \"disco\" started")
			return nil
		},
	}
}

func (a *Application) newDaemonStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop daemon service",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := a.getDaemonManager()
			if err != nil {
				return err
			}

			if err := manager.Stop(); err != nil {
				return err
			}

			fmt.Println("daemon service \"disco\" stopped")
			return nil
		},
	}
}

func (a *Application) newDaemonStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon service status",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := a.getDaemonManager()
			if err != nil {
				return err
			}

			status, err := manager.Status()
			if err != nil {
				return err
			}

			label := serviceStatusLabel(status)
			fmt.Printf("daemon service \"disco\" status: %s\n", label)
			return nil
		},
	}
}

func (a *Application) newDaemonInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install daemon service",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := a.getDaemonManager()
			if err != nil {
				return err
			}

			if err := manager.Install(); err != nil {
				return err
			}

			fmt.Println("daemon service \"disco\" installed")
			return nil
		},
	}
}

func (a *Application) newDaemonUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall daemon service",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := a.getDaemonManager()
			if err != nil {
				return err
			}

			if err := manager.Uninstall(); err != nil {
				return err
			}

			fmt.Println("daemon service \"disco\" uninstalled")
			return nil
		},
	}
}

func serviceStatusLabel(status service.Status) string {
	switch status {
	case service.StatusRunning:
		return "running"
	case service.StatusStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
