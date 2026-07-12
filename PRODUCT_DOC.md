# Wagon Product Doc

## Summary

Wagon is a terminal file manager for people who use `rclone` to move files between local storage and cloud remotes. It gives `rclone` a fast, visual, keyboard-driven interface while keeping the safety and scriptability of a CLI.

The core experience is a two-pane file manager: local folders are the default, external drives can be chosen from inside the UI, and cloud remotes can be opened side by side, with multi-file selection, transfer progress, dry-run previews, and saved jobs for repeated workflows.

## Working Tagline

File hauling for `rclone`.

## Target Users

- People who already use `rclone` but want a better everyday interface.
- Mac users managing files across Google Drive, Dropbox, OneDrive, S3, Backblaze, NAS, or external drives.
- Power users who like terminal tools but do not want to type every long copy/sync command by hand.
- People who run repeated backup or archive jobs and want confirmation before destructive changes.

## Problem

`rclone` is powerful, but common file-management tasks can be slow to compose manually:

- Browsing remotes requires remembering command flags.
- Moving many files safely requires careful path handling.
- Sync operations can be risky without a clear dry-run summary.
- Long transfers are hard to manage across repeated terminal commands.
- Repeated jobs often live as shell history instead of named workflows.

## Product Goals

- Make local-to-local, local-to-remote, and remote-to-remote movement easy.
- Keep advanced `rclone` users close to the underlying commands.
- Make destructive operations difficult to trigger accidentally.
- Provide a calm, readable terminal UI.
- Make repeated jobs easy to save and rerun.

## Non-Goals for v1

- Replace every `rclone` feature.
- Implement a custom cloud-storage engine.
- Build a graphical macOS app.
- Ship through the Mac App Store.
- Hide all `rclone` terminology from users.

## Current Implementation

The current app is a Go CLI/TUI with these working features:

- `wagon browse` opens two local panes by default: current folder on the left and home folder on the right.
- `--right` opens a local path or remote path in the right pane.
- `--remote` remains a shortcut for opening a configured `rclone` remote in the right pane.
- `v` opens a local drive picker for the active pane, listing `/`, home, and mounted drives under `/Volumes`.
- `/` starts incremental search in the active pane and filters matches as the user types.
- Multi-select works with `Space`, `a`, and `A`.
- `c` copies the selected/current item into the opposite pane using `rclone`.
- `p` pauses or resumes an active browser copy queue after the current item finishes.
- `z` calculates the selected/current folder size on demand.
- `Z` analyzes sizes for all visible unknown-size items in the active pane.
- Size analysis shows a spinner strip with current item count, filename, elapsed time, and an `Esc` cancel hint.
- Browser copy shows an item-level progress strip with a spinner, current item count, filename, size, destination, and elapsed time.
- `wagon copy` supports local and remote paths, with `--dry-run`.
- `wagon sync` is dry-run by default; `--apply` performs the sync and `--yes` skips confirmation.

## Core UX

### Main Browser

```text
Wagon                                                   rclone file manager

 Local: /Users/jqn/Sites/wagon           Local: /Users/jqn
+-----------------------------------+    +-----------------------------------+
| Name                 Size   Date   |    | Name                 Size   Date   |
| ..                                  |    | ..                                  |
| bin/                        Jul 07 |    | Desktop/                    Jul 06 |
| cmd/                        Jul 07 |    | Documents/                  Jul 05 |
| internal/                   Jul 07 |    | Downloads/                  Jul 04 |
| README.md           2.1 KB  Jul 07 |    | Pictures/                   Jun 30 |
+-----------------------------------+    +-----------------------------------+

 Local: 0 selected
 Tab switches panes. / searches. Enter opens folders. v chooses a drive.

 / Copying item 2/5: tax-2025.pdf  size 2.4 MB / 6.9 MB+?  -> /Volumes/Backup  elapsed 3s

 [/] search   [Tab] pane   [Enter] open   [c] copy   [p] pause   [z] size   [Z] analyze   [v] drives
```

### Drive Picker

```text
Choose location for Local pane
Enter opens location. Esc cancels.

> Computer root  /
  Home           /Users/jqn
  Macintosh HD   /Volumes/Macintosh HD
  Seagate Hub    /Volumes/Seagate Hub
```

### Incremental Search

```text
 Local: /Users/jqn/Documents
 Search: tax_
+------------------------------------------+
|   Name                 Size      Date     |
| > tax-2025.pdf         2.4 MB    Jun 22   |
|   tax-notes.md         18 KB     Jun 10   |
+------------------------------------------+

2 match(es)
Enter opens the highlighted match. Esc clears search.
```

### Multi-Select

```text
 Local: /Users/jqn/Documents
+------------------------------------------+
|   Name                 Size      Date     |
|   ..                                      |
| x invoice-001.pdf      240 KB    Jul 06   |
| x invoice-002.pdf      238 KB    Jul 06   |
|   Photos/                        Jul 04   |
| x notes.md             18 KB     Jun 30   |
|   archive.zip          1.1 GB    Jun 12   |
+------------------------------------------+

3 selected - 496 KB
Action: copy selected -> opposite pane
```

### Dry-Run Sync

```text
wagon sync ~/Pictures b2:photos

Dry run only. Re-run with --apply to perform the sync.
Running: rclone sync ~/Pictures b2:photos --dry-run

To perform the sync:

wagon sync ~/Pictures b2:photos --apply
```

## Implemented Keybindings

- `Tab`: switch pane
- `Up/Down`: move cursor
- `Enter`: open folder
- `Backspace`: go up
- `Space`: select or unselect current item
- `/`: search/filter the active pane as you type
- `c`: copy selected/current item into the opposite pane
- `p`: pause or resume the active transfer queue after the current item
- `z`: calculate the size of the selected/current folder or unknown-size item
- `Z`: analyze sizes for all visible unknown-size items in the active pane
- `v`: choose a local drive or location for the active pane
- `Esc`: clear search, cancel size analysis, or close the drive picker
- `a`: select all
- `A`: clear selection
- `r`: refresh
- `?`: help
- `q`: quit

## Planned Keybindings

- `m`: move
- `s`: sync
- `d`: delete
- `j`: jobs

## Safety Model

- Browser copy operations run immediately.
- Browser copy progress is item-level in the current TUI; byte-level progress is planned for the transfer queue.
- Browser copy can pause between transfer items with `p`; the current `rclone` item is allowed to finish first.
- Size analysis can be canceled with `Esc`; starting another `z` or `Z` analysis cancels the previous one.
- Move operations require confirmation.
- Delete operations require confirmation and should show item count.
- CLI sync operations default to dry-run preview before execution.
- Any operation that may delete remote files must call that out clearly.
- Commands should be invoked without shell interpolation.
- Failed operations should preserve enough detail to retry or inspect.

## Data Model

### Location

- Kind: local or remote
- Display name
- Resolved path
- Remote name, if applicable
- Current directory

### File Item

- Name
- Path
- Is directory
- Size, if known
- Size calculation state: unknown sizes display as `?` until calculated
- Modified time, if known
- Source location
- Selection state

### Transfer

- ID
- Operation: copy, move, sync, delete
- Source
- Destination
- Status
- Progress
- Speed
- ETA
- Started at
- Finished at
- Error, if any

### Saved Job

- Name
- Operation
- Source
- Destination
- Flags
- Created at
- Last run at
- Last status

## Configuration

Suggested path:

```text
~/.config/wagon/config.yaml
```

Example:

```yaml
default_local_path: ~/Documents
default_remote: gdrive:
confirm_destructive: true
dry_run_sync_by_default: true
recent_locations:
  - ~/Documents
  - gdrive:Archive
  - b2:photos
```

## Rclone Integration

Initial commands:

```bash
rclone version
rclone listremotes
rclone lsjson <path>
rclone copyto <source> <dest>
rclone copy <source> <dest> --progress
rclone move <source> <dest> --progress
rclone sync <source> <dest> --dry-run
rclone sync <source> <dest> --progress
```

Implementation notes:

- Call `rclone` as a subprocess with argument arrays.
- Never build commands through shell strings.
- Capture stdout and stderr separately.
- Normalize local and remote paths before display.
- Keep the raw command available in logs for debugging.

## v1 Success Criteria

- A user can browse a local folder and an `rclone` remote side by side.
- A user can browse two local folders, including external drives under `/Volumes`.
- A user can switch either pane to a mounted drive from inside the UI.
- A user can select multiple files.
- A user can copy selected files between any two loaded panes.
- A user can preview a sync from the CLI before running it.
- A user can see item-level browser copy progress and command-level transfer progress.
- A user can install and run Wagon without complex setup beyond installing `rclone`.

## Future Ideas

- Built-in `rclone config` helper.
- Remote-to-remote transfer presets.
- Mount/unmount remote commands.
- Scheduled jobs through launchd.
- Transfer history search.
- Checksums and verify mode.
- Optional mouse support.
- Optional native macOS companion app.
