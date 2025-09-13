package mcp

import (
    "path/filepath"

    cfg "codectl/internal/config"
    "codectl/internal/provider"
    sstore "codectl/internal/store"
)

func filePath() (string, error) {
    dir, err := cfg.Dir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "mcp.json"), nil
}

// Load returns configured MCP servers.
func Load() ([]string, error) {
    p, err := filePath()
    if err != nil {
        return nil, err
    }
    return sstore.LoadStringList(p)
}

// Save writes configured MCP servers.
func Save(list []string) error {
    p, err := filePath()
    if err != nil {
        return err
    }
    return sstore.SaveStringList(p, list)
}

// ListRemote returns a placeholder list of known MCP servers.
func ListRemote() []string { return provider.MCPServers() }

// Add adds given servers to the MCP list.
func Add(toAdd []string) (added []string, existed []string, err error) {
    p, err := filePath()
    if err != nil {
        return nil, nil, err
    }
    return sstore.AddToStringList(p, toAdd)
}

// Remove removes given servers from the MCP list.
func Remove(toRemove []string) (removed []string, missing []string, err error) {
    p, err := filePath()
    if err != nil {
        return nil, nil, err
    }
    return sstore.RemoveFromStringList(p, toRemove)
}
