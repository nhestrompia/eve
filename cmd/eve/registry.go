package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const repoRegistryVersion = 1

type repoRegistry struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Repositories  []repoRegistryEntry `json:"repositories"`
}

type repoRegistryEntry struct {
	ID         string `json:"id"`
	Root       string `json:"root"`
	LastSeenAt string `json:"lastSeenAt"`
}

func rememberRepository(repo repository) {
	_ = upsertRepositoryRegistry(repo)
}

func knownRepositories(primary repository) []repository {
	repositories := []repository{primary}
	seenRoots := map[string]bool{cleanRegistryRoot(primary.Root): true}
	seenIDs := map[string]bool{primary.ID: true}

	registry, err := loadRepositoryRegistry()
	if err != nil {
		return repositories
	}
	for _, entry := range registry.Repositories {
		root := cleanRegistryRoot(entry.Root)
		if root == "" || seenRoots[root] {
			continue
		}
		repo := repoFromRoot(root)
		if seenIDs[repo.ID] || !repoLooksUsable(repo) {
			continue
		}
		seenRoots[root] = true
		seenIDs[repo.ID] = true
		repositories = append(repositories, repo)
	}
	if len(repositories) > 2 {
		sort.Slice(repositories[1:], func(i, j int) bool {
			return repositories[i+1].ID < repositories[j+1].ID
		})
	}
	return repositories
}

func upsertRepositoryRegistry(repo repository) error {
	root := cleanRegistryRoot(repo.Root)
	if root == "" {
		return nil
	}
	registry, err := loadRepositoryRegistry()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		registry = repoRegistry{}
	}
	registry.SchemaVersion = repoRegistryVersion

	updated := false
	for i, entry := range registry.Repositories {
		if cleanRegistryRoot(entry.Root) != root {
			continue
		}
		registry.Repositories[i].ID = repo.ID
		registry.Repositories[i].Root = root
		registry.Repositories[i].LastSeenAt = nowUTC()
		updated = true
		break
	}
	if !updated {
		registry.Repositories = append(registry.Repositories, repoRegistryEntry{
			ID:         repo.ID,
			Root:       root,
			LastSeenAt: nowUTC(),
		})
	}
	sort.Slice(registry.Repositories, func(i, j int) bool {
		return registry.Repositories[i].ID < registry.Repositories[j].ID
	})

	path, err := repositoryRegistryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func loadRepositoryRegistry() (repoRegistry, error) {
	path, err := repositoryRegistryPath()
	if err != nil {
		return repoRegistry{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return repoRegistry{}, err
	}
	var registry repoRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return repoRegistry{}, err
	}
	return registry, nil
}

func repositoryRegistryPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("EVE_REPOSITORY_REGISTRY")); path != "" {
		return filepath.Abs(path)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "eve", "repositories.json"), nil
}

func repoLooksUsable(repo repository) bool {
	if _, err := os.Stat(filepath.Join(repo.Root, ".git")); err != nil {
		return false
	}
	if _, err := os.Stat(repo.eveDir); err != nil {
		return false
	}
	return true
}

func cleanRegistryRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return filepath.Clean(abs)
}
