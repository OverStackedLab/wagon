package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ModeCopy = "copy"
	ModeSync = "sync"
)

// Job is a saved copy or sync between two rclone-addressable locations.
type Job struct {
	Name        string   `yaml:"name"`
	Source      string   `yaml:"source"`
	Destination string   `yaml:"destination"`
	Mode        string   `yaml:"mode"`
	Apply       bool     `yaml:"apply,omitempty"`
	Flags       []string `yaml:"flags,omitempty"`
}

// File is the on-disk collection of saved jobs.
type File struct {
	Jobs []Job `yaml:"jobs"`
}

func Validate(job Job) error {
	if strings.TrimSpace(job.Name) == "" {
		return fmt.Errorf("job name is required")
	}
	if SanitizeName(job.Name) == "" {
		return fmt.Errorf("job name needs at least one letter or digit")
	}
	if strings.TrimSpace(job.Source) == "" {
		return fmt.Errorf("job source is required")
	}
	if strings.TrimSpace(job.Destination) == "" {
		return fmt.Errorf("job destination is required")
	}
	if job.Mode != ModeCopy && job.Mode != ModeSync {
		return fmt.Errorf("job mode must be %q or %q", ModeCopy, ModeSync)
	}
	return nil
}

// SanitizeName converts a job name into a token safe for file names and
// launchd labels.
func SanitizeName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-.")
}

// Load reads the jobs file; a missing file is an empty collection.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("read jobs file: %w", err)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse jobs file %s: %w", path, err)
	}
	return file, nil
}

func Save(path string, file File) error {
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode jobs file: %w", err)
	}
	return writeFileAtomic(path, data)
}

func (f File) Find(name string) (Job, bool) {
	for _, job := range f.Jobs {
		if job.Name == name {
			return job, true
		}
	}
	return Job{}, false
}

func (f *File) Add(job Job) error {
	if err := Validate(job); err != nil {
		return err
	}
	if _, exists := f.Find(job.Name); exists {
		return fmt.Errorf("a job named %q already exists", job.Name)
	}
	f.Jobs = append(f.Jobs, job)
	return nil
}

func (f *File) Remove(name string) bool {
	for index, job := range f.Jobs {
		if job.Name == name {
			f.Jobs = append(f.Jobs[:index], f.Jobs[index+1:]...)
			return true
		}
	}
	return false
}

// Sorted returns the jobs ordered by name for stable listings.
func (f File) Sorted() []Job {
	sorted := make([]Job, len(f.Jobs))
	copy(sorted, f.Jobs)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	return sorted
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
