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
const defaultDiscoveryDepth = 3

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
	add := func(repo repository) {
		root := cleanRegistryRoot(repo.Root)
		if root == "" || seenRoots[root] || seenIDs[repo.ID] || !repoLooksUsable(repo) {
			return
		}
		seenRoots[root] = true
		seenIDs[repo.ID] = true
		repositories = append(repositories, repo)
	}

	registry, err := loadRepositoryRegistry()
	if err == nil {
		for _, entry := range registry.Repositories {
			add(repoFromRoot(entry.Root))
		}
	}

	for _, repo := range discoverRepositories(primary, registry) {
		add(repo)
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
	registry.Repositories = pruneRegistryEntries(registry.Repositories, root)

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

func pruneRegistryEntries(entries []repoRegistryEntry, keepRoot string) []repoRegistryEntry {
	pruned := make([]repoRegistryEntry, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		root := cleanRegistryRoot(entry.Root)
		if root == "" || seen[root] {
			continue
		}
		repo := repoFromRoot(root)
		if root != keepRoot && !repoLooksUsable(repo) {
			continue
		}
		entry.ID = repo.ID
		entry.Root = root
		pruned = append(pruned, entry)
		seen[root] = true
	}
	return pruned
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

func discoverRepositories(primary repository, registry repoRegistry) []repository {
	roots := discoveryRoots(primary, registry)
	discovered := make([]repository, 0)
	seenRoots := map[string]bool{}
	for _, root := range roots {
		root = cleanRegistryRoot(root)
		if root == "" || seenRoots[root] {
			continue
		}
		seenRoots[root] = true
		discovered = append(discovered, discoverRepositoriesUnder(root, defaultDiscoveryDepth)...)
	}
	return discovered
}

func discoveryRoots(primary repository, registry repoRegistry) []string {
	if value := strings.TrimSpace(os.Getenv("EVE_DISCOVERY_ROOTS")); value != "" {
		return strings.FieldsFunc(value, func(r rune) bool {
			return r == filepath.ListSeparator || r == '\n'
		})
	}

	roots := []string{filepath.Dir(primary.Root)}
	for _, entry := range registry.Repositories {
		root := cleanRegistryRoot(entry.Root)
		if root != "" && repoLooksUsable(repoFromRoot(root)) {
			roots = append(roots, filepath.Dir(root))
		}
	}
	return roots
}

func discoverRepositoriesUnder(root string, maxDepth int) []repository {
	root = filepath.Clean(root)
	var repositories []repository
	_ = filepath.WalkDir(root, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if current != root {
			name := entry.Name()
			if isDiscoveryIgnoredDir(name) {
				return filepath.SkipDir
			}
			if relative, err := filepath.Rel(root, current); err == nil && directoryDepth(relative) > maxDepth {
				return filepath.SkipDir
			}
		}
		repo := repoFromRoot(current)
		if repoLooksUsable(repo) {
			repositories = append(repositories, repo)
			return filepath.SkipDir
		}
		return nil
	})
	return repositories
}

func isDiscoveryIgnoredDir(name string) bool {
	switch name {
	case ".git", ".eve", ".cache", ".config", ".next", "node_modules", "ui_dist", "dist", "build", "vendor":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func directoryDepth(relative string) int {
	if relative == "." || relative == "" {
		return 0
	}
	return len(strings.Split(filepath.Clean(relative), string(os.PathSeparator)))
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
