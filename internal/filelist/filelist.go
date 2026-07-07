package filelist

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OverStackedLab/wagon/internal/rclone"
)

type Kind int

const (
	Local Kind = iota
	Remote
)

func DetectKind(value string) Kind {
	if IsRemotePath(value) {
		return Remote
	}
	return Local
}

func IsRemotePath(value string) bool {
	value = strings.TrimSpace(value)
	colon := strings.Index(value, ":")
	if colon <= 0 {
		return false
	}

	prefix := value[:colon]
	return !strings.Contains(prefix, "/")
}

func Join(kind Kind, base string, name string) string {
	if kind == Remote {
		return JoinRemote(base, name)
	}
	return filepath.Join(base, name)
}

func (k Kind) String() string {
	if k == Remote {
		return "remote"
	}
	return "local"
}

type Item struct {
	Name      string
	Path      string
	IsDir     bool
	IsParent  bool
	Size      int64
	SizeKnown bool
	ModTime   time.Time
	TimeKnown bool
}

func ListLocal(localPath string) (string, []Item, error) {
	resolved, err := ResolveLocal(localPath)
	if err != nil {
		return "", nil, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return resolved, nil, err
	}

	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			items = append(items, Item{
				Name:  entry.Name(),
				Path:  filepath.Join(resolved, entry.Name()),
				IsDir: entry.IsDir(),
			})
			continue
		}
		items = append(items, Item{
			Name:      entry.Name(),
			Path:      filepath.Join(resolved, entry.Name()),
			IsDir:     entry.IsDir(),
			Size:      info.Size(),
			SizeKnown: !entry.IsDir(),
			ModTime:   info.ModTime(),
			TimeKnown: true,
		})
	}
	sortItems(items)

	if parent := filepath.Dir(resolved); parent != resolved {
		items = append([]Item{{
			Name:     "..",
			Path:     parent,
			IsDir:    true,
			IsParent: true,
		}}, items...)
	}

	return resolved, items, nil
}

func ResolveLocal(localPath string) (string, error) {
	if strings.TrimSpace(localPath) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		localPath = wd
	}

	if localPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		localPath = home
	} else if strings.HasPrefix(localPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		localPath = filepath.Join(home, strings.TrimPrefix(localPath, "~/"))
	}

	return filepath.Abs(localPath)
}

func SizeLocal(ctx context.Context, localPath string) (int64, error) {
	resolved, err := ResolveLocal(localPath)
	if err != nil {
		return 0, err
	}

	var total int64
	err = filepath.WalkDir(resolved, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

func FromRemoteEntries(remotePath string, entries []rclone.Entry) []Item {
	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name
		if name == "" {
			name = path.Base(entry.Path)
		}

		modTime, timeKnown := parseRcloneTime(entry.ModTime)
		items = append(items, Item{
			Name:      name,
			Path:      JoinRemote(remotePath, name),
			IsDir:     entry.IsDir,
			Size:      entry.Size,
			SizeKnown: !entry.IsDir && entry.Size >= 0,
			ModTime:   modTime,
			TimeKnown: timeKnown,
		})
	}
	sortItems(items)

	if parent := RemoteParent(remotePath); parent != remotePath {
		items = append([]Item{{
			Name:     "..",
			Path:     parent,
			IsDir:    true,
			IsParent: true,
		}}, items...)
	}

	return items
}

func JoinRemote(base string, name string) string {
	base = strings.TrimSpace(base)
	name = strings.TrimLeft(name, "/")
	if base == "" {
		return name
	}
	if strings.HasSuffix(base, ":") {
		return base + name
	}
	return strings.TrimRight(base, "/") + "/" + name
}

func RemoteParent(remotePath string) string {
	remotePath = strings.TrimRight(remotePath, "/")
	colon := strings.Index(remotePath, ":")
	if colon < 0 {
		return remotePath
	}

	root := remotePath[:colon+1]
	rest := strings.Trim(remotePath[colon+1:], "/")
	if rest == "" {
		return remotePath
	}

	parent := path.Dir(rest)
	if parent == "." {
		return root
	}
	return root + parent
}

func FormatSize(item Item) string {
	if item.IsParent {
		return "-"
	}
	if !item.SizeKnown {
		return "?"
	}

	return FormatBytes(item.Size)
}

func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "?"
	}

	size := float64(bytes)
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size = size / 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}

func FormatTime(item Item) string {
	if !item.TimeKnown {
		return "-"
	}
	now := time.Now()
	if item.ModTime.Year() == now.Year() {
		return item.ModTime.Format("Jan 02")
	}
	return item.ModTime.Format("2006")
}

func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}

func parseRcloneTime(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
