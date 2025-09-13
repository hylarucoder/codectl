package models

import (
	"path/filepath"

	cfg "codectl/internal/config"
	"codectl/internal/provider"
	sstore "codectl/internal/store"
)

// filePath returns the models storage file path.
func filePath() (string, error) {
	dir, err := cfg.DotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "models.json"), nil
}

// Load returns the current model list from disk. Missing file yields empty list.
func Load() ([]string, error) {
	p, err := filePath()
	if err != nil {
		return nil, err
	}
	return sstore.LoadStringList(p)
}

// Save writes the model list to disk, creating the directory if needed.
func Save(models []string) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	return sstore.SaveStringList(p, models)
}

// Add adds the given models to the store, returning which were added and which already existed.
func Add(toAdd []string) (added []string, existed []string, err error) {
	p, err := filePath()
	if err != nil {
		return nil, nil, err
	}
	return sstore.AddToStringList(p, toAdd)
}

// Remove removes the given models, returning which were removed and which were missing.
func Remove(toRemove []string) (removed []string, missing []string, err error) {
	p, err := filePath()
	if err != nil {
		return nil, nil, err
	}
	return sstore.RemoveFromStringList(p, toRemove)
}

// ListRemote returns a static list of known remote models as a placeholder.
func ListRemote() []string {
	// Load from provider catalog (~/.codectl/provider.json)
	return provider.Models()
}
