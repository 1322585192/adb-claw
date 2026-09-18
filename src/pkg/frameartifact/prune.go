package frameartifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type entry struct {
	token string
	meta  Metadata
	mod   time.Time
}

// Prune deletes expired frame artifacts and orphan images in Dir().
func Prune() {
	prune(time.Now())
}

func prune(now time.Time) {
	entries := listEntries()
	keep := make(map[string]struct{}, MaxFrames)
	for i, item := range entries {
		if i < MaxFrames && now.Sub(item.mod) <= MaxAge {
			keep[item.token] = struct{}{}
			continue
		}
		removeFrame(item.token, item.meta.Path)
	}
	removeOrphans(keep)
}

func listEntries() []entry {
	files, err := os.ReadDir(Dir())
	if err != nil {
		return nil
	}
	entries := make([]entry, 0, len(files))
	for _, file := range files {
		item, ok := readEntry(file)
		if !ok {
			continue
		}
		entries = append(entries, item)
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].meta.CapturedAt.Equal(entries[j].meta.CapturedAt) {
			return entries[i].meta.CapturedAt.After(entries[j].meta.CapturedAt)
		}
		return entries[i].token > entries[j].token
	})
	return entries
}

func readEntry(file os.DirEntry) (entry, bool) {
	if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
		return entry{}, false
	}
	token := strings.TrimSuffix(file.Name(), ".json")
	data, err := os.ReadFile(filepath.Join(Dir(), file.Name()))
	if err != nil {
		return entry{}, false
	}
	var meta Metadata
	if json.Unmarshal(data, &meta) != nil || meta.Token != token {
		return entry{}, false
	}
	info, err := file.Info()
	if err != nil {
		return entry{}, false
	}
	return entry{token: token, meta: meta, mod: info.ModTime()}, true
}

func removeFrame(token, imagePath string) {
	_ = os.Remove(metadataPath(token))
	if insideDir(imagePath, Dir()) {
		_ = os.Remove(imagePath)
	}
}

func removeOrphans(keep map[string]struct{}) {
	files, err := os.ReadDir(Dir())
	if err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".json" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}
		token := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if !validToken(token) {
			continue
		}
		if _, ok := keep[token]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(Dir(), file.Name()))
	}
}
