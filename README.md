# Wagon

Wagon is a terminal file manager for `rclone`. It starts with a two-pane browser for local files, external drives, and cloud remotes, plus safe command wrappers for common transfer operations.

## Requirements

- macOS or another Unix-like shell
- Go 1.26 or newer
- `rclone` installed
- An `rclone` remote configured only if you want cloud/remote browsing

```bash
brew install rclone go
rclone config # optional, for cloud remotes
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
bin/wagon doctor
bin/wagon browse
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
- `c`: copy selected/current item into the opposite pane
- `v`: choose a local drive or location for the active pane
- `Esc`: close the drive picker
- `a`: select all
- `A`: clear selection
- `r`: refresh active pane
- `?`: show help
- `q`: quit

The browser supports copying between any two loaded locations: local-to-local, local-to-remote, remote-to-local, and remote-to-remote. For an external drive, press `v` inside the UI and choose it from `/Volumes`, or open it directly with a path like `/Volumes/DriveName`.

Sync is still available as a CLI command while the transfer queue and in-browser sync actions are built out.
