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
```

`wagon sync` is dry-run by default. Use `--apply` to perform the sync, and `--yes` to skip the confirmation prompt.

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

## Project Docs

- [Architecture guide](docs/architecture.md)
- [Homebrew release guide](docs/homebrew-release.md)
