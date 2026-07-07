package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

type Client struct {
	bin string
}

type Entry struct {
	Path    string `json:"Path"`
	Name    string `json:"Name"`
	Size    int64  `json:"Size"`
	ModTime string `json:"ModTime"`
	IsDir   bool   `json:"IsDir"`
}

func NewClient() Client {
	return Client{bin: "rclone"}
}

func (c Client) Path() (string, error) {
	return exec.LookPath(c.bin)
}

func (c Client) Version(ctx context.Context) (string, error) {
	output, err := c.combined(ctx, "version")
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (c Client) ListRemotes(ctx context.Context) ([]string, error) {
	output, err := c.combined(ctx, "listremotes")
	if err != nil {
		return nil, err
	}

	var remotes []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	sort.Strings(remotes)
	return remotes, nil
}

func (c Client) LSJSON(ctx context.Context, remotePath string) ([]Entry, error) {
	output, err := c.combined(ctx, "lsjson", remotePath)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("parse rclone lsjson output: %w", err)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func (c Client) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", CommandString(c.bin, args...), err)
	}
	return nil
}

func (c Client) CopyItem(ctx context.Context, source string, destination string, isDir bool) (string, error) {
	args := []string{"copyto", source, destination}
	if isDir {
		args = []string{"copy", source, destination}
	}
	return c.RunOutput(ctx, args...)
}

func (c Client) RunOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.bin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return stdout.String(), fmt.Errorf("%s failed: %w: %s", CommandString(c.bin, args...), err, detail)
		}
		return stdout.String(), fmt.Errorf("%s failed: %w", CommandString(c.bin, args...), err)
	}

	if stdout.Len() > 0 && stderr.Len() > 0 {
		return stdout.String() + "\n" + stderr.String(), nil
	}
	return stdout.String() + stderr.String(), nil
}

func (c Client) combined(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return nil, fmt.Errorf("%s failed: %w: %s", CommandString(c.bin, args...), err, detail)
		}
		return nil, fmt.Errorf("%s failed: %w", CommandString(c.bin, args...), err)
	}
	return output, nil
}

func CommandString(binary string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, binary)
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '\\' || r == '$' || r == '&' || r == ';' || r == '(' || r == ')' || r == '<' || r == '>' || r == '|' || r == '*'
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
