package kitchen

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileStore persists attempt counters in a JSON file so retries on
// later OpenTofu runs see the same attempt history. The default state
// lives in ~/.local/share/oven/kitchen.json; tests point it at a
// temp dir via OVEN_STATE_DIR.
type FileStore struct {
	Path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{Path: path}
}

func (s *FileStore) readAll() map[string]int {
	out := map[string]int{}
	data, err := os.ReadFile(s.Path)
	if err != nil || json.Unmarshal(data, &out) != nil {
		return map[string]int{} // missing or corrupt file behaves as empty
	}
	return out
}

func (s *FileStore) Load(key string) (int, bool) {
	n, ok := s.readAll()[key]
	return n, ok
}

func (s *FileStore) Set(key string, value int) error {
	all := s.readAll()
	all[key] = value
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, data, 0o600)
}
