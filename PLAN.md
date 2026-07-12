# Wagon Implementation Plan

## Goal

Build Wagon as a macOS-first terminal file manager for `rclone`: a fast two-pane TUI for browsing local folders and cloud remotes, selecting multiple files, and running safe copy, move, sync, and delete operations.

## Recommended Stack

- Language: Go
- CLI framework: Cobra
- TUI framework: Bubble Tea
- Styling: Lip Gloss
- File/remote engine: `rclone` CLI subprocesses first, `rclone rc` later if needed
- Config: YAML or TOML in `~/.config/wagon/config.yaml`
- Distribution: Homebrew tap first, raw binaries later

## Current Status

Implemented:

- Go module with Cobra commands and Bubble Tea browser.
- `wagon`, `wagon browse`, `wagon doctor`, `wagon remotes`, `wagon copy`, `wagon sync`, and `wagon version`.
- Local/local browser defaults: current folder on the left, home folder on the right.
- `--right` for local or remote right-pane paths.
- `--remote` as a shortcut for opening an `rclone` remote.
- Local drive picker with `v`, listing `/`, home, and mounted drives under `/Volumes`.
- Incremental per-pane search with `/`, live filtering, Enter-open, and Esc-clear.
- Multi-select with `Space`, `a`, and `A`.
- On-demand folder size calculation with `z` for selected/current items and `Z` for all visible unknown sizes.
- Browser copy with `c` between local and remote locations through `rclone`.
- Pause/resume for browser copy queues with `p`, pausing between transfer items.
- Browser copy progress strip with spinner, item count, filename, size, destination, and elapsed time.
- CLI sync dry-run by default, with `--apply` and `--yes`.

Not yet implemented:

- TUI byte-level transfer progress, move, delete, mkdir, sync, transfer queue, cancel/retry, saved jobs, packaging.
- Background/scheduled sync jobs, transfer queue resume (not just retry), multi-machine config sync, and a hosted status dashboard.

## Guiding Product Principles

- Make destructive actions explicit and reversible where possible.
- Prefer dry-run summaries before sync/delete operations.
- Keep the interface keyboard-first and scriptable.
- Treat local and remote locations as equal peers.
- Avoid hiding raw `rclone` power; expose it cleanly.

## Milestone 0: Project Setup

Tasks:

- Create a Go module.
- Add Cobra command scaffolding.
- Add Bubble Tea app shell.
- Add a small `rclone` wrapper package.
- Add basic config loading.
- Add logging for failed subprocess calls.

Acceptance:

- `wagon --help` works.
- `wagon version` works.
- `wagon doctor` checks for `rclone` on PATH.
- `wagon remotes` lists configured `rclone` remotes.

## Milestone 1: Browser

Tasks:

- Implement a two-pane browser layout.
- Support local filesystem navigation.
- Default both panes to local filesystem locations.
- Add a local drive picker for mounted drives under `/Volumes`.
- Support remote navigation through `rclone lsjson`.
- Add pane switching with `Tab`.
- Add refresh with `r`.
- Add basic error display.
- Add search/filter within the active pane.

Acceptance:

- User can launch `wagon browse`.
- Left pane can browse local folders.
- Right pane defaults to a local folder.
- User can choose a mounted drive from inside the UI.
- User can still browse an `rclone` remote with `--remote` or `--right remote:`.
- User can filter the active pane as they type and open a matching folder with `Enter`.
- User can calculate selected/current folder sizes or analyze all visible unknown sizes on demand.
- The app handles empty folders, inaccessible folders, and missing remotes gracefully.

## Milestone 2: Selection Model

Tasks:

- Add cursor movement.
- Add single item select/unselect with `Space`.
- Add select all and clear selection.
- Add range selection.
- Track selected item paths per pane.
- Show selected count and total known size.

Acceptance:

- User can select multiple files in either pane.
- Selection survives cursor movement and pane switching.
- Selection clears or updates predictably after navigation.

## Milestone 3: Copy and Move

Tasks:

- Add copy action from active pane to opposite pane.
- Add move action from active pane to opposite pane.
- Use safe subprocess invocation without shell interpolation.
- Support selected files first; fall back to current cursor item.
- Show transfer progress.
- Add cancel support.

Acceptance:

- User can copy one file, many selected files, or a folder.
- User can move one file, many selected files, or a folder.
- Transfer failures are visible and retryable.
- Local-to-local copies work for normal folders and external drives.
- Browser copy shows visible item-level progress while running.

## Milestone 4: Sync with Dry Run

Tasks:

- Add `wagon sync <source> <dest>` as a dry-run-by-default command.
- Add interactive sync from TUI.
- Parse and display dry-run summaries.
- Require explicit confirmation for non-dry-run sync.
- Keep delete behavior visible in the summary.

Acceptance:

- User can preview sync changes before applying them.
- Deletes are clearly called out.
- Sync can be run from both CLI and TUI.

## Milestone 5: Transfer Queue

Tasks:

- Add queue state model.
- Add transfer list view.
- Add statuses: queued, running, complete, failed, canceled.
- Add retry failed transfer.
- Add resume for an interrupted or canceled transfer, continuing from the last completed item instead of restarting.
- Persist recent transfer history.

Acceptance:

- Long-running copies can be monitored.
- Multiple selected items can become a visible queue.
- User can cancel current transfer and continue working.
- User can resume a canceled or interrupted transfer without redoing completed items.

## Milestone 6: Saved Jobs

Tasks:

- Add `wagon jobs` commands.
- Add saved copy/sync jobs.
- Add job names, source, destination, mode, flags, and last run status.
- Add TUI job picker.

Acceptance:

- User can create a job like `Pictures to Backblaze`.
- User can run a saved job from CLI or TUI.
- Last run outcome is visible.

## Milestone 7: Packaging

Tasks:

- Add release build script.
- Add Homebrew formula.
- Add GitHub Actions release workflow.
- Add checks for Apple Silicon and Intel builds.
- Write installation docs.

Acceptance:

- User can install with Homebrew.
- `wagon doctor` confirms setup after install.
- Release artifacts are reproducible.

## Milestone 8: Background/Scheduled Sync and Multi-Machine Config

Tasks:

- Add a background scheduler that can run a saved job on an interval or cron-like schedule without the TUI open.
- Add `wagon jobs schedule "<job>" --every 1h` (or cron expression) and `wagon jobs unschedule`.
- Run scheduled jobs via a lightweight daemon or OS scheduler integration (e.g. launchd on macOS) rather than a long-lived foreground process.
- Add failure notifications (desktop notification and/or log file) for scheduled runs.
- Add multi-machine config sync: let `~/.config/wagon/config.yaml` and saved jobs sync across machines through an existing configured remote, so a job created on one machine is available on another.
- Add `wagon config push` / `wagon config pull` for explicit config sync, plus a conflict warning if both sides changed.

Acceptance:

- User can schedule a saved job to run automatically on a recurring basis without keeping a terminal open.
- User is notified when a scheduled run fails.
- User can carry saved jobs and settings to a second machine without manually recreating them.
- Scheduled jobs respect the same dry-run/confirm safety model as interactive sync.

## Potential Monetization Track: Hosted Status Dashboard

This is a candidate paid add-on, not part of the free/open-source core CLI and TUI. It should stay decoupled so the local tool keeps working fully offline with no account required.

Tasks:

- Define an opt-in, anonymized job/run reporting format (job name, source/dest kind, start/end time, bytes moved, status) that `wagon` can optionally POST after a run.
- Build a minimal hosted API to ingest run reports per authenticated user/API key.
- Build a hosted web dashboard showing recent runs, schedules, and failures across all of a user's machines.
- Add `wagon login` / `wagon logout` and an API key stored in local config to enable reporting.
- Keep everything behind an explicit opt-in; the CLI and TUI must never require an account to browse, copy, or sync.

Acceptance:

- A user with the paid tier can see the status of scheduled jobs across multiple machines from a web page.
- Reporting is fully optional; disabling it removes all network calls beyond `rclone` itself.
- Free-tier users are unaffected and see no prompts to upgrade during normal use.

## Current Command Shape

```bash
wagon
wagon browse
wagon browse --local ~/Documents --right /Volumes/Backup
wagon browse --local ~/Documents --right gdrive:
wagon doctor
wagon remotes
wagon copy ~/Documents gdrive:Documents
wagon copy ~/Documents /Volumes/Backup/Documents
wagon sync ~/Pictures b2:photos
wagon sync ~/Pictures b2:photos --apply
```

## Planned Command Shape

```bash
wagon move ~/Downloads/archive.zip b2:archives
wagon jobs list
wagon jobs run "Pictures to Backblaze"
wagon jobs schedule "Pictures to Backblaze" --every 1h
wagon jobs unschedule "Pictures to Backblaze"
wagon config push
wagon config pull
```

## Open Questions

- Should Wagon bundle `rclone`, or require the user to install it?
- Should v1 support only configured remotes, or also guide users through `rclone config`?
- Should delete move to trash where possible, or call `rclone delete` directly after confirmation?
- Should the TUI support mouse input, or stay keyboard-only for v1?
- Should the app include a command log that can replay failed operations?

## Suggested First Build Slice

Start with:

1. `wagon doctor`
2. `wagon remotes`
3. `wagon browse`
4. Local/local two-pane navigation
5. Multi-select model
6. Browser copy between panes

This first slice is implemented. The next useful slice is a transfer queue with byte-level `rclone` progress for browser copy operations.
