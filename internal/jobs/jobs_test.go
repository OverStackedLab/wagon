package jobs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.yaml")
	file := File{}
	job := Job{
		Name:        "photos to b2",
		Source:      "~/Pictures",
		Destination: "b2:photos",
		Mode:        ModeSync,
		Apply:       true,
		Flags:       []string{"--exclude=.DS_Store"},
	}

	if err := file.Add(job); err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if err := Save(path, file); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	got, ok := loaded.Find("photos to b2")
	if !ok {
		t.Fatal("Find() did not find the saved job")
	}
	if !reflect.DeepEqual(got, job) {
		t.Fatalf("loaded job = %+v, want %+v", got, job)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	loaded, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load() error for missing file: %v", err)
	}
	if len(loaded.Jobs) != 0 {
		t.Fatalf("Load() returned %d jobs for a missing file, want 0", len(loaded.Jobs))
	}
}

func TestAddRejectsDuplicateNames(t *testing.T) {
	file := File{}
	job := Job{Name: "backup", Source: "a", Destination: "b", Mode: ModeCopy}
	if err := file.Add(job); err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if err := file.Add(job); err == nil {
		t.Fatal("Add() accepted a duplicate job name")
	}
}

func TestValidateRejectsBadJobs(t *testing.T) {
	bad := []Job{
		{Name: "", Source: "a", Destination: "b", Mode: ModeCopy},
		{Name: "!!!", Source: "a", Destination: "b", Mode: ModeCopy},
		{Name: "x", Source: "", Destination: "b", Mode: ModeCopy},
		{Name: "x", Source: "a", Destination: "", Mode: ModeCopy},
		{Name: "x", Source: "a", Destination: "b", Mode: "move"},
	}
	for _, job := range bad {
		if err := Validate(job); err == nil {
			t.Fatalf("Validate(%+v) accepted an invalid job", job)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	if got := SanitizeName("My Photos Backup!"); got != "my-photos-backup" {
		t.Fatalf("SanitizeName() = %q, want %q", got, "my-photos-backup")
	}
}

func TestArgsCopyInteractive(t *testing.T) {
	job := Job{Name: "j", Source: "src", Destination: "dst", Mode: ModeCopy}
	got := Args(job, false)
	want := []string{"copy", "src", "dst", "--progress"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args() = %v, want %v", got, want)
	}
}

func TestArgsSyncIsDryRunWithoutApply(t *testing.T) {
	job := Job{Name: "j", Source: "src", Destination: "dst", Mode: ModeSync}
	got := Args(job, false)
	want := []string{"sync", "src", "dst", "--dry-run", "--progress"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args() = %v, want %v", got, want)
	}
}

func TestArgsScheduledSyncWithApplyAndFlags(t *testing.T) {
	job := Job{
		Name:        "j",
		Source:      "src",
		Destination: "dst",
		Mode:        ModeSync,
		Apply:       true,
		Flags:       []string{"--exclude=.DS_Store"},
	}
	got := Args(job, true)
	want := []string{"sync", "src", "dst", "--stats-one-line", "--exclude=.DS_Store"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args() = %v, want %v", got, want)
	}
}

func TestRecordRunRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job-runs.yaml")
	record := RunRecord{
		StartedAt:  time.Date(2026, 7, 12, 3, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 7, 12, 3, 1, 30, 0, time.UTC),
		Success:    false,
		Scheduled:  true,
		Error:      "rclone sync failed",
	}

	if err := RecordRun(path, "backup", record); err != nil {
		t.Fatalf("RecordRun() error: %v", err)
	}
	runs, err := LoadRuns(path)
	if err != nil {
		t.Fatalf("LoadRuns() error: %v", err)
	}
	got, ok := runs["backup"]
	if !ok {
		t.Fatal("LoadRuns() is missing the recorded job")
	}
	if !got.StartedAt.Equal(record.StartedAt) || got.Success || !got.Scheduled || got.Error != record.Error {
		t.Fatalf("loaded record = %+v, want %+v", got, record)
	}
}

func TestAcquireLockConflictsWithLiveProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.lock")

	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("acquireLock() error: %v", err)
	}
	defer unlock()

	if _, err := acquireLock(path); err == nil {
		t.Fatal("acquireLock() succeeded while the lock is held by a live process")
	}
}

func TestAcquireLockStealsStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.lock")
	// A PID far beyond any real process, so the holder looks dead.
	if err := os.WriteFile(path, []byte("999999999\n"), 0o644); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("acquireLock() did not steal a stale lock: %v", err)
	}
	unlock()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("unlock did not remove the lock file")
	}
}
