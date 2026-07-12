package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OverStackedLab/wagon/internal/config"
	"github.com/OverStackedLab/wagon/internal/jobs"
	"github.com/OverStackedLab/wagon/internal/notify"
	"github.com/OverStackedLab/wagon/internal/rclone"
	"github.com/OverStackedLab/wagon/internal/schedule"
	"github.com/spf13/cobra"
)

func newJobsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage saved copy and sync jobs",
	}
	cmd.AddCommand(newJobsAddCommand())
	cmd.AddCommand(newJobsListCommand())
	cmd.AddCommand(newJobsRemoveCommand())
	cmd.AddCommand(newJobsRunCommand())
	cmd.AddCommand(newJobsScheduleCommand())
	cmd.AddCommand(newJobsUnscheduleCommand())
	return cmd
}

func newJobsAddCommand() *cobra.Command {
	var mode string
	var apply bool
	var flags []string

	cmd := &cobra.Command{
		Use:   "add <name> <source> <destination>",
		Short: "Save a named copy or sync job",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.JobsPath()
			if err != nil {
				return err
			}
			file, err := jobs.Load(path)
			if err != nil {
				return err
			}

			job := jobs.Job{
				Name:        args[0],
				Source:      args[1],
				Destination: args[2],
				Mode:        mode,
				Apply:       apply,
				Flags:       flags,
			}
			if err := file.Add(job); err != nil {
				return err
			}
			if err := jobs.Save(path, file); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Saved job %q: %s %s -> %s\n", job.Name, job.Mode, job.Source, job.Destination)
			if job.Mode == jobs.ModeSync && !job.Apply {
				fmt.Fprintln(out, "This sync job is dry-run only. Re-create it with --apply to let it change the destination.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", jobs.ModeCopy, "job mode: copy or sync")
	cmd.Flags().BoolVar(&apply, "apply", false, "let a sync job change the destination instead of dry-running")
	cmd.Flags().StringArrayVar(&flags, "flag", nil, "extra rclone flag, repeatable, for example --flag=--exclude=.DS_Store")
	return cmd
}

func newJobsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved jobs and their last run",
		RunE: func(cmd *cobra.Command, args []string) error {
			jobsPath, err := config.JobsPath()
			if err != nil {
				return err
			}
			file, err := jobs.Load(jobsPath)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(file.Jobs) == 0 {
				fmt.Fprintln(out, "No saved jobs. Create one with: wagon jobs add <name> <source> <destination>")
				return nil
			}

			runsPath, err := config.RunsPath()
			if err != nil {
				return err
			}
			runs, err := jobs.LoadRuns(runsPath)
			if err != nil {
				return err
			}

			for _, job := range file.Sorted() {
				mode := job.Mode
				if job.Mode == jobs.ModeSync && !job.Apply {
					mode = "sync (dry-run)"
				}
				scheduled := ""
				if schedule.IsScheduled(job.Name) {
					scheduled = "  [scheduled]"
				}
				fmt.Fprintf(out, "%s  %s  %s -> %s%s\n", job.Name, mode, job.Source, job.Destination, scheduled)

				if record, ok := runs[job.Name]; ok {
					result := "ok"
					if !record.Success {
						result = "failed: " + record.Error
					}
					fmt.Fprintf(out, "    last run: %s  %s\n", record.StartedAt.Format("2006-01-02 15:04"), result)
				} else {
					fmt.Fprintln(out, "    last run: never")
				}
			}
			return nil
		},
	}
}

func newJobsRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a saved job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.JobsPath()
			if err != nil {
				return err
			}
			file, err := jobs.Load(path)
			if err != nil {
				return err
			}
			if !file.Remove(args[0]) {
				return fmt.Errorf("no job named %q", args[0])
			}
			if err := jobs.Save(path, file); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed job %q.\n", args[0])
			if schedule.IsScheduled(args[0]) {
				fmt.Fprintf(cmd.OutOrStdout(), "The job is still scheduled. Remove the schedule with: wagon jobs unschedule %q\n", args[0])
			}
			return nil
		},
	}
}

func newJobsRunCommand() *cobra.Command {
	var scheduled bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a saved job now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobsPath, err := config.JobsPath()
			if err != nil {
				return err
			}
			file, err := jobs.Load(jobsPath)
			if err != nil {
				return err
			}
			job, ok := file.Find(args[0])
			if !ok {
				return fmt.Errorf("no job named %q; list jobs with: wagon jobs list", args[0])
			}

			if job.Mode == jobs.ModeSync && job.Apply && !scheduled && !yes {
				if !confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Sync can update or delete destination files. Continue? [y/N] ") {
					return nil
				}
			}

			opts, err := runOptionsForJob(job, scheduled)
			if err != nil {
				return err
			}
			opts.Stdout = cmd.OutOrStdout()
			opts.Stderr = cmd.ErrOrStderr()

			return jobs.Run(cmd.Context(), rclone.NewClient(), job, opts)
		},
	}

	cmd.Flags().BoolVar(&scheduled, "scheduled", false, "run headless for a scheduler: log to file, notify on failure")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation for sync jobs saved with --apply")
	return cmd
}

func newJobsScheduleCommand() *cobra.Command {
	var every time.Duration

	cmd := &cobra.Command{
		Use:   "schedule <name>",
		Short: "Run a saved job on a recurring interval via launchd",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if every < time.Minute {
				return fmt.Errorf("--every must be at least 1m")
			}

			jobsPath, err := config.JobsPath()
			if err != nil {
				return err
			}
			file, err := jobs.Load(jobsPath)
			if err != nil {
				return err
			}
			job, ok := file.Find(args[0])
			if !ok {
				return fmt.Errorf("no job named %q; list jobs with: wagon jobs list", args[0])
			}

			wagonPath, err := executablePath()
			if err != nil {
				return err
			}
			logPath, err := jobLogPath(job)
			if err != nil {
				return err
			}

			plistPath, err := schedule.Install(cmd.Context(), job.Name, wagonPath, logPath, every)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Scheduled job %q every %s.\n", job.Name, every)
			fmt.Fprintf(out, "Agent: %s\nLog:   %s\n", plistPath, logPath)
			if job.Mode == jobs.ModeSync && !job.Apply {
				fmt.Fprintln(out, "Note: this sync job was saved without --apply, so scheduled runs only log a dry-run.")
			}
			return nil
		},
	}

	cmd.Flags().DurationVar(&every, "every", time.Hour, "run interval, for example 30m, 1h, 12h")
	return cmd
}

func newJobsUnscheduleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unschedule <name>",
		Short: "Stop running a job on a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plistPath, err := schedule.Uninstall(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unscheduled job %q and removed %s.\n", args[0], plistPath)
			return nil
		},
	}
}

func runOptionsForJob(job jobs.Job, scheduled bool) (jobs.RunOptions, error) {
	locksDir, err := config.LocksDir()
	if err != nil {
		return jobs.RunOptions{}, err
	}
	runsPath, err := config.RunsPath()
	if err != nil {
		return jobs.RunOptions{}, err
	}
	logPath, err := jobLogPath(job)
	if err != nil {
		return jobs.RunOptions{}, err
	}

	return jobs.RunOptions{
		Scheduled: scheduled,
		LogPath:   logPath,
		LockPath:  filepath.Join(locksDir, jobs.SanitizeName(job.Name)+".lock"),
		RunsPath:  runsPath,
		Notify:    notify.Failure,
	}, nil
}

func jobLogPath(job jobs.Job) (string, error) {
	logsDir, err := config.LogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logsDir, jobs.SanitizeName(job.Name)+".log"), nil
}

// executablePath resolves the running wagon binary to a stable absolute path
// for launchd, following the make-install symlink to the real binary.
func executablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve wagon executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve wagon executable symlink: %w", err)
	}
	return resolved, nil
}
