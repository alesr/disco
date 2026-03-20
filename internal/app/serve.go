package app

import (
	"context"
	"fmt"

	"github.com/alesr/disco/internal/policy/rag"
	"github.com/alesr/disco/internal/runtime"
	"github.com/alesr/disco/internal/transport/local"
	"github.com/philippgille/chromem-go"
)

const vectorDBPath = "./data/vector_db"

type ServeOptions struct {
	StyleGuideDir string
}

func RunServe(ctx context.Context, options ServeOptions) error {
	db, err := chromem.NewPersistentDB(vectorDBPath, false)
	if err != nil {
		return fmt.Errorf("could not create persistent DB: %w", err)
	}

	if err := ingestStyleGuideOnStartup(ctx, db, options.StyleGuideDir); err != nil {
		return fmt.Errorf("could not ingest style guide on daemon startup: %w", err)
	}

	manager, err := runtime.NewManager(ctx)
	if err != nil {
		return fmt.Errorf("could not create runtime manager: %w", err)
	}
	defer manager.Close(context.Background())

	server, err := local.NewServer(local.DefaultSocketPath, manager)
	if err != nil {
		return fmt.Errorf("could not create local server: %w", err)
	}
	return runServerLoop(ctx, server)
}

func ingestStyleGuideOnStartup(ctx context.Context, db *chromem.DB, styleGuideDir string) error {
	return rag.NewIngestService(db).Ingest(ctx, styleGuideDir)
}

type daemonServer interface {
	Serve() error
	Shutdown(context.Context) error
}

func runServerLoop(ctx context.Context, server daemonServer) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	select {
	case <-ctx.Done():
		if err := server.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("could not shutdown local daemon: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			_ = server.Shutdown(context.Background())
			return fmt.Errorf("local daemon stopped with error: %w", err)
		}
		return nil
	}
}
