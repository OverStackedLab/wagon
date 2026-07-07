package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jqn/wagon/internal/browser"
	"github.com/jqn/wagon/internal/rclone"
	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func Execute(ctx context.Context) error {
	cmd := NewRootCommand()
	return cmd.ExecuteContext(ctx)
}

func NewRootCommand() *cobra.Command {
	var localPath string
	var remotePath string
	var rightPath string

	root := &cobra.Command{
		Use:   "wagon",
		Short: "A terminal file manager for rclone",
		Long:  "Wagon is a terminal file manager for browsing local files and rclone remotes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(cmd.Context(), localPath, rightPath, remotePath)
		},
	}

	root.Flags().StringVar(&localPath, "local", "", "local path to open in the left pane")
	root.Flags().StringVar(&rightPath, "right", "", "local or remote path to open in the right pane, for example /Volumes/Backup or gdrive:")
	root.Flags().StringVar(&remotePath, "remote", "", "remote path to open in the right pane, for example gdrive:")

	root.AddCommand(newVersionCommand())
	root.AddCommand(newDoctorCommand())
	root.AddCommand(newRemotesCommand())
	root.AddCommand(newBrowseCommand())
	root.AddCommand(newCopyCommand())
	root.AddCommand(newSyncCommand())

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Wagon version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check local Wagon and rclone setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			client := rclone.NewClient()
			out := cmd.OutOrStdout()

			path, err := client.Path()
			if err != nil {
				return fmt.Errorf("rclone was not found on PATH; install it with: brew install rclone")
			}

			fmt.Fprintf(out, "wagon:  ok (%s)\n", version)
			fmt.Fprintf(out, "rclone: ok (%s)\n", path)

			versionText, err := client.Version(ctx)
			if err != nil {
				return err
			}
			if first := firstLine(versionText); first != "" {
				fmt.Fprintf(out, "version: %s\n", first)
			}

			remotes, err := client.ListRemotes(ctx)
			if err != nil {
				fmt.Fprintf(out, "remotes: unable to list (%v)\n", err)
				return nil
			}
			fmt.Fprintf(out, "remotes: %d configured\n", len(remotes))
			return nil
		},
	}
}

func newRemotesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remotes",
		Short: "List configured rclone remotes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			remotes, err := rclone.NewClient().ListRemotes(ctx)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(remotes) == 0 {
				fmt.Fprintln(out, "No rclone remotes configured. Run: rclone config")
				return nil
			}
			for _, remote := range remotes {
				fmt.Fprintln(out, remote)
			}
			return nil
		},
	}
}

func newBrowseCommand() *cobra.Command {
	var localPath string
	var remotePath string
	var rightPath string

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Open the two-pane terminal browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(cmd.Context(), localPath, rightPath, remotePath)
		},
	}

	cmd.Flags().StringVar(&localPath, "local", "", "local path to open in the left pane")
	cmd.Flags().StringVar(&rightPath, "right", "", "local or remote path to open in the right pane, for example /Volumes/Backup or gdrive:")
	cmd.Flags().StringVar(&remotePath, "remote", "", "remote path to open in the right pane, for example gdrive:")
	return cmd
}

func newCopyCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "copy <source> <destination>",
		Short: "Copy files with rclone",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := rclone.NewClient()
			runArgs := []string{"copy", args[0], args[1]}
			if dryRun {
				runArgs = append(runArgs, "--dry-run")
			} else {
				runArgs = append(runArgs, "--progress")
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Running: %s\n", rclone.CommandString("rclone", runArgs...))
			return client.Run(cmd.Context(), runArgs, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be copied without changing anything")
	return cmd
}

func newSyncCommand() *cobra.Command {
	var apply bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "sync <source> <destination>",
		Short: "Sync files with rclone; dry-run by default",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := rclone.NewClient()
			runArgs := []string{"sync", args[0], args[1]}
			if !apply {
				runArgs = append(runArgs, "--dry-run")
				fmt.Fprintln(cmd.ErrOrStderr(), "Dry run only. Re-run with --apply to perform the sync.")
			} else {
				runArgs = append(runArgs, "--progress")
				if !yes && !confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Sync can update or delete destination files. Continue? [y/N] ") {
					return nil
				}
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Running: %s\n", rclone.CommandString("rclone", runArgs...))
			return client.Run(cmd.Context(), runArgs, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "perform the sync instead of a dry run")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation when used with --apply")
	return cmd
}

func runBrowse(ctx context.Context, localPath string, rightPath string, remotePath string) error {
	if strings.TrimSpace(rightPath) != "" && strings.TrimSpace(remotePath) != "" {
		return fmt.Errorf("use either --right or --remote, not both")
	}

	options := browser.Options{
		LocalPath: localPath,
		RightKind: browser.LocalPane,
	}
	if strings.TrimSpace(remotePath) != "" {
		options.RightPath = remotePath
		options.RightKind = browser.RemotePane
		options.AutoPickRemote = false
	} else if strings.TrimSpace(rightPath) != "" {
		options.RightPath = rightPath
		options.RightKind = browser.DetectPaneKind(rightPath)
		options.AutoPickRemote = false
	}

	client := rclone.NewClient()
	return browser.Run(ctx, client, options)
}

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}
