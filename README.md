<img src="docs/icon.svg" alt="Wagon icon" width="96" height="96" />

# Wagon

Wagon is a terminal file manager for `rclone`. It starts with a two-pane browser for local files, external drives, and cloud remotes, plus safe command wrappers for common transfer operations.

![Wagon terminal UI screenshot](docs/screenshot.svg)

## Requirements

- macOS or another Unix-like shell
- Go 1.26 or newer
- `rclone` installed
- An `rclone` remote configured only if you want cloud/remote browsing

```bash
brew install rclone go
rclone config # optional, for cloud remotes
```

## Install With Homebrew

Wagon will install from the OverStackedLab tap:

```bash
brew tap OverStackedLab/tap
brew install wagon
```

## Run Locally

```bash
go run ./cmd/wagon doctor
go run ./cmd/wagon remotes
go run ./cmd/wagon browse
go run ./cmd/wagon browse --local ~/Documents --right /Volumes/Backup
```

By default, `wagon browse` opens two local panes: the left pane starts in the current folder and the right pane starts in your home folder.

You can also build a local binary:

```bash
go build -o bin/wagon ./cmd/wagon
./bin/wagon doctor
./bin/wagon browse
```

To make `wagon` work as a shell command, install it into a directory on your `PATH`:

```bash
make install
wagon doctor
wagon browse
```

## Commands

```bash
wagon
wagon browse
wagon browse --local ~/Documents --right /Volumes/Backup
wagon browse --local ~/Documents --right gdrive:
wagon doctor
wagon remotes
wagon copy <source> <destination>
wagon copy <source> <destination> --dry-run
wagon sync <source> <destination>
wagon sync <source> <destination> --apply
wagon sync <source> <destination> --apply --yes
wagon jobs add <name> <source> <destination> [--mode copy|sync] [--apply] [--flag <rclone-flag>]
wagon jobs list
wagon jobs run <name>
wagon jobs schedule <name> --every 1h
wagon jobs unschedule <name>
wagon jobs remove <name>
```

`wagon sync` is dry-run by default. Use `--apply` to perform the sync, and `--yes` to skip the confirmation prompt.

## Saved Jobs and Scheduling

Save a copy or sync as a named job, then run it by name or put it on a schedule:

```bash
wagon jobs add "photos to b2" ~/Pictures b2:photos --mode sync --apply
wagon jobs run "photos to b2"
wagon jobs schedule "photos to b2" --every 6h
```

Jobs live in `~/.config/wagon/jobs.yaml`. Sync jobs stay dry-run unless they were saved with `--apply`, so a scheduled job can never delete destination files you did not explicitly opt into. Running an `--apply` sync job interactively still asks for confirmation; pass `--yes` to skip it.

Scheduling uses macOS launchd: `wagon jobs schedule` writes a LaunchAgent that runs the job in the background on the chosen interval, even with no terminal open. Scheduled runs write to a per-job log in `~/Library/Logs/wagon/`, record their outcome for `wagon jobs list`, take a per-job lock so runs never overlap, and show a desktop notification when a run fails. `wagon jobs unschedule` removes the agent. Scheduling requires macOS for now; jobs themselves work anywhere.

## Browser Keys

- `Tab`: switch pane
- `Up/Down` or `k/j`: move cursor
- `Enter`: open folder
- `Backspace`: go up
- `Space`: select or unselect item
- `/`: search/filter the active pane as you type
- `c`: copy selected/current item into the opposite pane
- `p`: pause or resume the active transfer queue after the current item
- `z`: calculate the size of the selected/current folder or unknown-size item
- `Z`: analyze sizes for all visible unknown-size items in the active pane
- `v`: choose a local drive or location for the active pane
- `Esc`: clear search, cancel size analysis, or close the drive picker
- `a`: select all
- `A`: clear selection
- `r`: refresh active pane
- `?`: show help
- `q`: quit

The browser supports copying between any two loaded locations: local-to-local, local-to-remote, remote-to-local, and remote-to-remote. For an external drive, press `v` inside the UI and choose it from `/Volumes`, or open it directly with a path like `/Volumes/DriveName`.

Search is incremental: press `/`, type part of a file or folder name, and the active pane filters immediately. `Enter` opens the highlighted match, arrow keys move through matches, and `Esc` clears the search.

Browser copy shows an item-level progress strip while copying, including a spinner, current item count, current filename, current item size, known batch size, destination, and elapsed time. Press `p` during a copy to pause the queue after the current item finishes, then press `p` again to resume. Byte-level `rclone` progress is still available in CLI copy output and is planned for the TUI transfer queue.

Folder sizes are calculated on demand because recursive local walks and remote `rclone size` calls can be slow. Folders show `?` until you select one or move the cursor to it and press `z`. Press `Z` to analyze every visible unknown-size item in the active pane without selecting items first. Size analysis shows a spinner strip with the current item and elapsed time. If an analysis is slow, press `Esc` to cancel it, or press `z`/`Z` again to restart with a new target.

Sync is still available as a CLI command while the transfer queue and in-browser sync actions are built out.

## Disclaimer

- **Wagon moves real files.** Always review dry-run output before running `wagon sync` with `--apply`, and keep backups of anything irreplaceable. The authors are not responsible for lost or corrupted data.
- **Sync deletes files.** `wagon sync --apply` makes the destination mirror the source, which means files at the destination that do not exist in the source are deleted.
- **Early-stage software.** Wagon is pre-1.0. Commands, keybindings, and behavior may change between releases, and you should expect rough edges.
- **Not affiliated with rclone.** Wagon is an independent project and is not affiliated with or endorsed by the rclone project.
- **Cloud costs are yours.** Transfers to or from cloud remotes can incur storage, bandwidth, or API charges from your provider.
- **No warranty.** Wagon is provided as-is under the [MIT License](LICENSE), without warranty of any kind.

## Project Docs

- [Architecture guide](docs/architecture.md)
- [Homebrew release guide](docs/homebrew-release.md)
