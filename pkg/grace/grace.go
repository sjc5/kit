package grace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sjc5/kit/pkg/colorlog"
)

type OrchestrateOptions struct {
	Timeout         time.Duration // Default: 30 seconds
	Signals         []os.Signal   // Default: SIGHUP, SIGINT, SIGTERM, SIGQUIT
	Logger          *slog.Logger  // Default: os.Stdout
	StartupCallback func() error
	CleanupCallback func(context.Context) error
}

// Orchestrate manages the core lifecycle of an application, including startup, shutdown, and os signal handling.
func Orchestrate(options OrchestrateOptions) {
	// Set defaults
	if options.Logger == nil {
		options.Logger = newDefaultLogger()
	}
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if len(options.Signals) == 0 {
		options.Signals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
	}

	// Context for orchestrating shutdown
	ctx, stopCtx := context.WithCancel(context.Background())
	defer stopCtx()

	// Signal handling
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, options.Signals...)
	defer signal.Stop(sig)

	// Create a channel to coordinate cleanup
	cleanup := make(chan struct{})

	// Handle cleanup in a separate goroutine
	go func() {
		select {
		case receivedSignal := <-sig:
			options.Logger.Info(fmt.Sprintf("[shutdown] Signal received: %v. Initiating graceful shutdown.", receivedSignal))
		case <-ctx.Done():
			options.Logger.Info("[shutdown] Initiating graceful shutdown due to startup failure")
		}

		shutdownCtx, cancelCtx := context.WithTimeout(context.Background(), options.Timeout)
		defer cancelCtx()

		// Execute cleanup logic
		if options.CleanupCallback != nil {
			if err := options.CleanupCallback(shutdownCtx); err != nil {
				options.Logger.Error(fmt.Sprintf("[shutdown] Cleanup error: %v", err))
			}
		}

		if shutdownCtx.Err() == context.DeadlineExceeded {
			options.Logger.Warn("[shutdown] Graceful shutdown timed out. Forcing exit.")
		}

		close(cleanup)
	}()

	// Execute startup logic
	if options.StartupCallback != nil {
		if err := options.StartupCallback(); err != nil {
			options.Logger.Error(fmt.Sprintf("[startup] Error: %v", err))
			stopCtx() // This will now trigger cleanup via ctx.Done()
		}
	}

	// Wait for cleanup to complete
	<-cleanup
}

// TerminateProcess attempts to gracefully terminate a process, falling back to force kill after timeout.
// If logger is nil, defaults to stdout.
func TerminateProcess(process *os.Process, timeToWait time.Duration, logger *slog.Logger) error {
	if logger == nil {
		logger = newDefaultLogger()
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	done := make(chan error)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process exited with error: %w", err)
		}
		return nil
	case <-time.After(timeToWait):
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process after timeout: %w", err)
		}
		pid := process.Pid
		logger.Warn(fmt.Sprintf("process %d killed after timeout of %v", pid, timeToWait))
		return nil
	}
}

func newDefaultLogger() *slog.Logger {
	return colorlog.New("pkg/grace")
}
