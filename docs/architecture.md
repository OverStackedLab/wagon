# Wagon Architecture

This guide explains how Wagon is structured and how the app runs.

## High-Level Layers

Wagon has four main layers:

1. Entrypoint
2. CLI commands
3. Browser/TUI
4. File listing and `rclone` integration

## Entrypoint

The app starts in:

```text
cmd/wagon/main.go
```

In Go, a compiled command starts at `package main` and `func main()`.

Wagon's `main()` function creates a cancellable context, runs the CLI, and prints any error:

```text
main.go
  -> cli.Execute
```

## CLI Commands

CLI command setup lives in:

```text
internal/cli/root.go
```

This package uses Cobra to define commands such as:

```bash
wagon browse
wagon doctor
wagon remotes
wagon copy
wagon sync
wagon version
```

The important browse path is:

```text
runBrowse
  -> builds browser options from --local, --right, and --remote
  -> creates an rclone client
  -> launches browser.Run
```

`wagon copy` and `wagon sync` are command-line wrappers around `rclone`.

`wagon sync` is dry-run by default. Use `--apply` to perform the sync and `--yes` to skip confirmation.

## Browser/TUI

The interactive terminal UI lives in:

```text
internal/browser/browser.go
```

It uses Bubble Tea. Bubble Tea apps follow a simple model:

```text
Model  = current app state
Update = handle keypresses/events and return new state
View   = render state as terminal text
```

Wagon's browser state includes:

- two panes
- active pane
- cursor position
- selected files
- search mode
- drive picker state
- copy transfer state
- folder size calculation state
- status text

Important functions:

```text
Run                 starts Bubble Tea
NewModel            creates initial pane state
Init                loads initial pane data
Update              handles keypresses and async messages
View                renders the full terminal UI
renderPane          renders one file pane
renderTransfer      renders copy progress
startSearch         starts incremental search
sizeCurrentSelection calculates selected/current unknown sizes
copyCurrentSelection starts browser copy
toggleTransferPause pauses/resumes copy queues between items
```

## File Listing

Shared file item types and path helpers live in:

```text
internal/filelist/filelist.go
```

The central type is `Item`, which represents both local files and remote files:

```text
Item
  Name
  Path
  IsDir
  IsParent
  Size
  ModTime
```

Local folders are read with Go filesystem APIs:

```text
ListLocal
  -> os.ReadDir
  -> converts entries into []Item
```

Remote folders are converted from `rclone lsjson` output:

```text
FromRemoteEntries
  -> converts rclone.Entry values into []Item
```

This lets the browser render local and remote panes with the same UI code.

## Rclone Integration

The `rclone` wrapper lives in:

```text
internal/rclone/client.go
```

It runs real `rclone` commands as subprocesses.

Important methods:

```text
Path        finds rclone on PATH
Version     runs rclone version
ListRemotes runs rclone listremotes
LSJSON      runs rclone lsjson <path>
CopyItem    runs rclone copyto for files or rclone copy for folders
Run         runs a generic rclone command for CLI commands
```

The app does not implement cloud storage itself. It delegates cloud/local transfer behavior to `rclone`.

## Browse Flow

When you run:

```bash
wagon browse
```

the flow is:

```text
cmd/wagon/main.go
  -> internal/cli.Execute
    -> Cobra parses command and flags
      -> runBrowse
        -> internal/browser.Run
          -> Bubble Tea event loop
            -> Init loads panes
            -> Update handles keys/messages
            -> View renders terminal UI
```

By default:

- left pane opens the current folder
- right pane opens the user's home folder

You can override the right pane:

```bash
wagon browse --right /Volumes/Backup
wagon browse --right gdrive:
wagon browse --remote gdrive:
```

## Pane Loading Flow

Pane loading happens in `browser.loadPane`.

For local panes:

```text
loadPane
  -> filelist.ListLocal
    -> os.ReadDir
```

For remote panes:

```text
loadPane
  -> rclone.LSJSON
    -> rclone lsjson <path>
    -> filelist.FromRemoteEntries
```

The result is always a list of `filelist.Item` values.

## Search Flow

When you press `/`:

```text
Update
  -> startSearch
```

While typing:

```text
updateSearch
  -> updates active pane search text
  -> filteredItemRefs filters visible items
  -> syncCursorWithFilter keeps cursor on a visible match
```

Search affects real actions:

- `Enter` opens the highlighted match
- `Space` selects the highlighted match
- `a` selects all visible matches
- `c` copies the selected/current visible item

## Copy Flow

When you press `c` in the browser:

```text
Update
  -> copyCurrentSelection
    -> chooses selected items, or current item if nothing is selected
    -> creates transfer state
    -> starts copyTransferItem(0)
```

Each item copy runs:

```text
copyTransferItem
  -> rclone.CopyItem
    -> rclone copyto source destination   for files
    -> rclone copy source destination     for folders
```

After each item completes:

```text
copyStepFinishedMsg
  -> advances to the next item
  -> pauses before the next item when pause was requested
  -> updates progress state
  -> refreshes destination pane when done
```

The progress strip is item-level:

```text
/ Copying item 2/5: tax-2025.pdf  size 2.4 MB / 6.9 MB+?  -> /Volumes/Backup  elapsed 3s
```

The first size is the current item. The second size is the known selected-item total; `+?` means at least one selected item still has an unknown size.

Pressing `p` during a copy requests a pause. The current `rclone` item is allowed to finish, then Wagon stops before launching the next queued item. Pressing `p` again resumes from that saved item index.

Byte-level `rclone` progress is currently available in CLI copy output and is planned for the TUI transfer queue.

## Size Flow

Folders show `?` until their size is calculated. This keeps browsing responsive because true folder size requires recursively walking local contents or asking `rclone` to scan a remote path.

When you press `z`:

```text
Update
  -> sizeCurrentSelection
    -> chooses selected unknown-size items, or the current unknown-size item
    -> starts sizeTransferItem(0)
```

For local paths:

```text
sizeTransferItem
  -> filelist.SizeLocal
    -> filepath.WalkDir
```

For remote paths:

```text
sizeTransferItem
  -> rclone.Size
    -> rclone size --json <path>
```

Each completed size updates the matching item in the active pane if the pane has not navigated away.

## Drive Picker Flow

When you press `v`:

```text
openDrivePicker
  -> localDriveChoices
    -> adds /
    -> adds home folder
    -> reads /Volumes
```

When you choose a location with `Enter`, the active pane becomes local and loads that path.

## Why `internal/`

Go treats directories named `internal` specially. Packages inside `internal/` can only be imported by code inside this module.

That fits Wagon well:

- `cmd/wagon` is the public executable
- `internal/*` packages are implementation details

## Build and Install

Build a local binary:

```bash
make build
./bin/wagon browse
```

Install `wagon` as a normal shell command:

```bash
make install
wagon browse
```

Run tests:

```bash
make test
```

Uninstall the shell command:

```bash
make uninstall
```
