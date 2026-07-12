package jobs

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// RunRecord is the last known outcome of a job run.
type RunRecord struct {
	StartedAt  time.Time `yaml:"started_at"`
	FinishedAt time.Time `yaml:"finished_at"`
	Success    bool      `yaml:"success"`
	Scheduled  bool      `yaml:"scheduled,omitempty"`
	Error      string    `yaml:"error,omitempty"`
}

type runsFile struct {
	Runs map[string]RunRecord `yaml:"runs"`
}

// LoadRuns reads run history keyed by job name; a missing file is empty history.
func LoadRuns(path string) (map[string]RunRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]RunRecord{}, nil
		}
		return nil, fmt.Errorf("read run history: %w", err)
	}

	var file runsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse run history %s: %w", path, err)
	}
	if file.Runs == nil {
		file.Runs = map[string]RunRecord{}
	}
	return file.Runs, nil
}

// RecordRun stores the latest outcome for a job.
func RecordRun(path string, name string, record RunRecord) error {
	runs, err := LoadRuns(path)
	if err != nil {
		return err
	}
	runs[name] = record

	data, err := yaml.Marshal(runsFile{Runs: runs})
	if err != nil {
		return fmt.Errorf("encode run history: %w", err)
	}
	return writeFileAtomic(path, data)
}
