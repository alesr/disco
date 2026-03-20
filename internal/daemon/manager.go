package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kardianos/service"
)

const (
	ServiceName        = "disco"
	ServiceDisplayName = "Disco"
	ServiceDescription = "Disco local review daemon"
)

type Manager struct {
	service service.Service
}

type Options struct {
	WorkingDirectory   string
	StyleGuideDir      string
	EmbeddingProvider  string
	GenerationProvider string
	MistralAPIKey      string
	MistralModel       string
	MistralEmbedModel  string
	MistralBaseURL     string
}

func NewManager() (*Manager, error) {
	return NewManagerWithOptions(Options{})
}

func NewManagerWithOptions(options Options) (*Manager, error) {
	svcOptions := service.KeyValue{}
	if runtime.GOOS == "darwin" {
		svcOptions["UserService"] = true
	}

	workingDir := strings.TrimSpace(options.WorkingDirectory)
	if workingDir == "" {
		var err error
		workingDir, err = serviceWorkingDirectory()
		if err != nil {
			return nil, fmt.Errorf("could not resolve service working directory: %w", err)
		}
	}

	envVars := map[string]string{}
	if styleGuideDir := strings.TrimSpace(options.StyleGuideDir); styleGuideDir != "" {
		envVars["STYLE_GUIDE_DIR"] = styleGuideDir
	}

	if embeddingProvider := strings.TrimSpace(options.EmbeddingProvider); embeddingProvider != "" {
		envVars["EMBEDDING_PROVIDER"] = embeddingProvider
	}

	if generationProvider := strings.TrimSpace(options.GenerationProvider); generationProvider != "" {
		envVars["GENERATION_PROVIDER"] = generationProvider
	}

	if mistralAPIKey := strings.TrimSpace(options.MistralAPIKey); mistralAPIKey != "" {
		envVars["MISTRAL_API_KEY"] = mistralAPIKey
	}

	if mistralModel := strings.TrimSpace(options.MistralModel); mistralModel != "" {
		envVars["MISTRAL_MODEL"] = mistralModel
	}

	if mistralEmbedModel := strings.TrimSpace(options.MistralEmbedModel); mistralEmbedModel != "" {
		envVars["MISTRAL_EMBED_MODEL"] = mistralEmbedModel
	}

	if mistralBaseURL := strings.TrimSpace(options.MistralBaseURL); mistralBaseURL != "" {
		envVars["MISTRAL_BASE_URL"] = mistralBaseURL
	}

	svc, err := service.New(noopProgram{}, &service.Config{
		Name:             ServiceName,
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		Arguments:        []string{"serve"},
		WorkingDirectory: workingDir,
		Option:           svcOptions,
		EnvVars:          envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create service manager: %w", err)
	}
	return &Manager{service: svc}, nil
}

func (m *Manager) Start() error {
	// starting w/o reinstall first avoids rewriting launchd state on healthy setups
	if err := m.startWithoutReinstall(4 * time.Second); err == nil {
		return nil
	}

	if err := m.reinstallServiceDefinition(); err != nil {
		return fmt.Errorf("could not refresh daemon service definition: %w", err)
	}

	if err := m.startWithoutReinstall(12 * time.Second); err != nil {
		return fmt.Errorf("could not start daemon service after refresh: %w", err)
	}
	return nil
}

func (m *Manager) startWithoutReinstall(timeout time.Duration) error {
	status, statusErr := m.service.Status()
	if statusErr == nil && status == service.StatusRunning {
		return nil
	}

	if err := m.service.Start(); err != nil && !isAlreadyRunningError(err) {
		return fmt.Errorf("could not start daemon service: %w", err)
	}

	if err := m.waitForRunning(timeout); err != nil {
		return err
	}
	return nil
}

func (m *Manager) waitForRunning(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := m.service.Status()
		if err == nil && status == service.StatusRunning {
			return nil
		}
		// small polling interval keeps status checks responsive without hammering launchd
		time.Sleep(400 * time.Millisecond)
	}

	status, err := m.service.Status()
	if err != nil {
		return fmt.Errorf("could not query daemon status after start: %w", err)
	}
	return fmt.Errorf("current status is %s; check ~/disco.err.log for startup errors", StatusLabel(status))
}

func (m *Manager) Stop() error {
	status, statusErr := m.service.Status()
	if statusErr == nil && status != service.StatusRunning {
		return nil
	}

	if err := m.service.Stop(); err != nil {
		if isNotInstalledError(err) {
			// stop is treated as idempotent to keep scripted teardown simple
			return nil
		}

		if isLaunchctlUnloadIOError(err) {
			status, statusErr := m.service.Status()
			if statusErr == nil && status == service.StatusStopped {
				return nil
			}
		}
		return fmt.Errorf("could not stop daemon service: %w", err)
	}
	return nil
}

func (m *Manager) Install() error {
	if err := m.service.Install(); err != nil && !isAlreadyInstalledError(err) {
		return fmt.Errorf("could not install daemon service: %w", err)
	}
	return nil
}

func (m *Manager) Uninstall() error {
	if err := m.service.Uninstall(); err != nil {
		if isNotInstalledError(err) {
			return nil
		}
		return fmt.Errorf("could not uninstall daemon service: %w", err)
	}
	return nil
}

func (m *Manager) Status() (service.Status, error) {
	status, err := m.service.Status()
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("could not query daemon service status: %w", err)
	}
	return status, nil
}

func StatusLabel(status service.Status) string {
	switch status {
	case service.StatusRunning:
		return "running"
	case service.StatusStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

func isNotInstalledError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "not installed") || strings.Contains(lower, "does not exist")
}

func isAlreadyInstalledError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already exists") || strings.Contains(lower, "already installed")
}

func isLaunchctlUnloadIOError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unload failed: 5") || strings.Contains(lower, "input/output error")
}

func isAlreadyRunningError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already started") || strings.Contains(lower, "already running")
}

func (m *Manager) reinstallServiceDefinition() error {
	if err := m.service.Uninstall(); err != nil && !isNotInstalledError(err) {
		return fmt.Errorf("could not uninstall daemon service: %w", err)
	}
	if err := m.service.Install(); err != nil && !isAlreadyInstalledError(err) {
		return fmt.Errorf("could not install daemon service: %w", err)
	}
	return nil
}

func serviceWorkingDirectory() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not get executable path: %w", err)
	}

	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		resolvedPath = executablePath
	}

	workingDir := filepath.Dir(resolvedPath)
	if filepath.Base(workingDir) == ".bin" {
		workingDir = filepath.Dir(workingDir)
	}
	return workingDir, nil
}

type noopProgram struct{}

func (noopProgram) Start(service.Service) error { return nil }

func (noopProgram) Stop(service.Service) error { return nil }
