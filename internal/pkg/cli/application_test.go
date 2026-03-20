package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDaemonManager struct {
	status service.Status
}

func (f fakeDaemonManager) Start() error                    { return nil }
func (f fakeDaemonManager) Stop() error                     { return nil }
func (f fakeDaemonManager) Install() error                  { return nil }
func (f fakeDaemonManager) Uninstall() error                { return nil }
func (f fakeDaemonManager) Status() (service.Status, error) { return f.status, nil }

func TestApplication_DaemonStatus(t *testing.T) {
	t.Parallel()

	t.Run("running status succeeds", func(t *testing.T) {
		t.Parallel()

		app := NewApplication(Dependencies{DaemonManager: fakeDaemonManager{status: service.StatusRunning}})

		err := app.Execute([]string{"daemon", "status"})
		require.NoError(t, err)
	})

	t.Run("stopped status succeeds", func(t *testing.T) {
		t.Parallel()

		app := NewApplication(Dependencies{DaemonManager: fakeDaemonManager{status: service.StatusStopped}})

		err := app.Execute([]string{"daemon", "status"})
		require.NoError(t, err)
	})
}

func TestApplication_CompletionFish(t *testing.T) {
	// dont parellelize it (stdout)
	app := NewApplication(Dependencies{DaemonManager: fakeDaemonManager{status: service.StatusRunning}})
	output := captureStdout(t, func() {
		err := app.Execute([]string{"completion", "fish"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "complete -c disco")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer
	fn()

	require.NoError(t, writer.Close())
	os.Stdout = originalStdout

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	return strings.TrimSpace(string(data))
}
