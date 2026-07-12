package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/OverStackedLab/wagon/internal/rclone"
)

// RunOptions controls how a job run behaves and where its output goes.
type RunOptions struct {
	// Scheduled runs headless: output goes to LogPath and failures notify.
	Scheduled bool
	LogPath   string
	LockPath  string
	RunsPath  string
	Stdout    io.Writer
	Stderr    io.Writer
	// Notify is called with a title and message when a scheduled run fails.
	Notify func(title, message string)
}

// Args builds the rclone arguments for a job. Sync jobs stay dry-run unless
// the job was saved with apply, so a scheduled run can never delete data the
// user did not explicitly opt into.
func Args(job Job, scheduled bool) []string {
	var args []string
	switch job.Mode {
	case ModeSync:
		args = []string{"sync", job.Source, job.Destination}
		if !job.Apply {
			args = append(args, "--dry-run")
		}
	default:
		args = []string{"copy", job.Source, job.Destination}
	}
	if scheduled {
		args = append(args, "--stats-one-line")
	} else {
		args = append(args, "--progress")
	}
	return append(args, job.Flags...)
}

// Run executes a job with rclone, holding the job lock for the duration and
// recording the outcome in run history.
func Run(ctx context.Context, client rclone.Client, job Job, opts RunOptions) error {
	unlock, err := acquireLock(opts.LockPath)
	if err != nil {
		return err
	}
	defer unlock()

	stdout := opts.Stdout
	stderr := opts.Stderr
	if opts.Scheduled {
		if err := os.MkdirAll(filepath.Dir(opts.LogPath), 0o755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		logFile, err := os.OpenFile(opts.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open job log: %w", err)
		}
		defer logFile.Close()
		stdout = logFile
		stderr = logFile
	}

	args := Args(job, opts.Scheduled)
	started := time.Now()
	fmt.Fprintf(stderr, "=== %s: %s at %s\n", job.Name, rclone.CommandString("rclone", args...), started.Format(time.RFC3339))

	runErr := client.Run(ctx, args, nil, stdout, stderr)
	finished := time.Now()

	record := RunRecord{
		StartedAt:  started,
		FinishedAt: finished,
		Success:    runErr == nil,
		Scheduled:  opts.Scheduled,
	}
	if runErr != nil {
		record.Error = runErr.Error()
	}
	if opts.RunsPath != "" {
		if err := RecordRun(opts.RunsPath, job.Name, record); err != nil {
			fmt.Fprintf(stderr, "warning: could not record run history: %v\n", err)
		}
	}

	elapsed := finished.Sub(started).Round(time.Second)
	if runErr != nil {
		fmt.Fprintf(stderr, "=== %s: failed after %s: %v\n", job.Name, elapsed, runErr)
		if opts.Scheduled && opts.Notify != nil {
			opts.Notify("Wagon job failed", fmt.Sprintf("%s: %v", job.Name, runErr))
		}
		return runErr
	}
	fmt.Fprintf(stderr, "=== %s: completed in %s\n", job.Name, elapsed)
	return nil
}
