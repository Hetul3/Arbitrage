package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Snapshot struct {
	CollectionID   string `json:"collection_id"`
	CollectionName string `json:"collection_name"`
}

func Path(root string) string {
	return filepath.Join(root, "state.json")
}

func Save(root string, snap Snapshot) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(root), data, 0o644)
}

func Load(root string) (Snapshot, error) {
	data, err := os.ReadFile(Path(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{}, fmt.Errorf("no vector store found; run the create command first")
		}
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, err
	}
	if snap.CollectionID == "" {
		return Snapshot{}, fmt.Errorf("state file missing collection_id")
	}
	return snap, nil
}

func Remove(root string) error {
	if err := os.Remove(Path(root)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
