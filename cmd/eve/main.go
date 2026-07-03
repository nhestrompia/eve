package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nhestrompia/eve"
)

const configFileVersion = 2

func main() {
	os.Exit(runWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWithIO(args, strings.NewReader(""), stdout, stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "dev":
		return runDev(args[1:], stdout, stderr)
	case "mcp-stdio":
		return runMCPStdio(args[1:], stdin, stdout, stderr)
	case "snapshot":
		return runSnapshot(args[1:], stdout, stderr)
	case "checkout":
		return runCheckout(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "canonicalize":
		return runCanonicalize(args[1:], stdout, stderr)
	case "version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "eve version takes no arguments")
			return 2
		}
		fmt.Fprintf(stdout, "eve %s (snapshot schema %s)\n", eve.CLIVersion, eve.SnapshotSchemaVersion)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: eve <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  dev [--addr localhost:4317]")
	fmt.Fprintln(w, "  snapshot <snapshot-id>")
	fmt.Fprintln(w, "  checkout [--force] <snapshot-id>")
	fmt.Fprintln(w, "  validate <snapshot.json>")
	fmt.Fprintln(w, "  canonicalize <snapshot.json>")
	fmt.Fprintln(w, "  version")
}

func runInit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve init takes no arguments")
		return 2
	}

	repo, err := resolveRepo(repoRequest{})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	for _, dir := range []string{repo.eveDir, repo.snapshotsDir(), repo.artifactsDir(), repo.cacheDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(stderr, "create %s: %v\n", dir, err)
			return 1
		}
	}
	config := map[string]any{
		"schemaVersion":  configFileVersion,
		"snapshotSchema": eve.SnapshotSchemaVersion,
		"createdAt":      nowUTC(),
	}
	if _, err := os.Stat(repo.configPath()); errors.Is(err, os.ErrNotExist) {
		data, marshalErr := json.MarshalIndent(config, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(stderr, "marshal config: %v\n", marshalErr)
			return 1
		}
		if err := os.WriteFile(repo.configPath(), append(data, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "write config: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Initialized EVE snapshots in %s\n", repo.eveDir)
	return 0
}

func runDev(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "localhost:4317", "local runtime listen address")
	cwd := fs.String("cwd", "", "repository working directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve dev takes no positional arguments")
		return 2
	}
	if !isLocalhostAddr(*addr) {
		fmt.Fprintln(stderr, "eve dev only binds to localhost")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: *cwd})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := repo.ensureDirs(); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	server := newRuntimeServer(repo, strings.TrimSpace(*addr))
	fmt.Fprintf(stdout, "EVE Runtime listening on http://%s\n", server.addr)
	fmt.Fprintf(stdout, "MCP Streamable HTTP endpoint: http://%s/mcp\n", server.addr)
	if err := http.ListenAndServe(server.addr, server.routes()); err != nil {
		fmt.Fprintf(stderr, "serve runtime: %v\n", err)
		return 1
	}
	return 0
}

func runMCPStdio(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp-stdio", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cwd := fs.String("cwd", "", "repository working directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve mcp-stdio takes no positional arguments")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: *cwd})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	server := newRuntimeServer(repo, "")
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		response := server.handleMCPMessage(context.Background(), line)
		if len(response) > 0 {
			fmt.Fprintln(stdout, string(response))
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "read mcp stdio: %v\n", err)
		return 1
	}
	return 0
}

func runSnapshot(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cwd := fs.String("cwd", "", "repository working directory")
	repoID := fs.String("repo-id", "", "repository id")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve snapshot requires a snapshot id")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: *cwd, RepoID: *repoID})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	snapshot, err := repo.loadSnapshot(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	printSnapshot(stdout, snapshot, repo)
	return 0
}

func runCheckout(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkout", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "checkout even when the working tree is dirty")
	cwd := fs.String("cwd", "", "repository working directory")
	repoID := fs.String("repo-id", "", "repository id")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve checkout requires a snapshot id")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: *cwd, RepoID: *repoID})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	snapshot, err := repo.loadSnapshot(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	result := checkoutSnapshot(repo, snapshot, *force)
	fmt.Fprint(stdout, result.Stdout)
	if result.ExitCode != 0 {
		fmt.Fprint(stderr, result.Stderr)
		return result.ExitCode
	}
	return 0
}

func runValidate(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve validate requires a snapshot JSON file")
		return 2
	}
	if _, err := eve.LoadSnapshotFile(fs.Arg(0)); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s is valid\n", fs.Arg(0))
	return 0
}

func runCanonicalize(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("canonicalize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve canonicalize requires a snapshot JSON file")
		return 2
	}
	snapshot, err := eve.LoadSnapshotFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	canonical, err := eve.CanonicalSnapshotJSON(snapshot)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(canonical))
	return 0
}

type repoRequest struct {
	CWD    string `json:"cwd,omitempty"`
	RepoID string `json:"repoId,omitempty"`
}

type repository struct {
	ID     string `json:"id"`
	Root   string `json:"root"`
	eveDir string
}

type repoSummary struct {
	ID             string `json:"id"`
	Root           string `json:"root"`
	SnapshotCount  int    `json:"snapshotCount"`
	LatestAt       string `json:"latestAt"`
	LatestSnapshot string `json:"latestSnapshot"`
	LatestTitle    string `json:"latestTitle"`
	Branch         string `json:"branch,omitempty"`
	Head           string `json:"head,omitempty"`
	Dirty          bool   `json:"dirty"`
	RemoteURL      string `json:"remoteUrl,omitempty"`
	LatestGitState string `json:"latestGitState,omitempty"`
}

type repoDetail struct {
	repoSummary
	Readme          string `json:"readme,omitempty"`
	PrimaryLanguage string `json:"primaryLanguage,omitempty"`
	SizeBytes       int64  `json:"sizeBytes,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

func resolveRepo(req repoRequest) (repository, error) {
	if strings.TrimSpace(req.RepoID) != "" {
		root, err := filepath.Abs(strings.TrimSpace(req.RepoID))
		if err != nil {
			return repository{}, err
		}
		return repoFromRoot(root), nil
	}
	start := strings.TrimSpace(req.CWD)
	if start == "" {
		if env := strings.TrimSpace(os.Getenv("EVE_CWD")); env != "" {
			start = env
		}
	}
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return repository{}, err
		}
		start = cwd
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return repository{}, err
	}
	root, err := findGitRoot(abs)
	if err != nil {
		return repository{}, err
	}
	return repoFromRoot(root), nil
}

func findGitRoot(start string) (string, error) {
	current := start
	if info, err := os.Stat(current); err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no .git directory found from %s", start)
		}
		current = parent
	}
}

func repoFromRoot(root string) repository {
	return repository{
		ID:     filepath.Base(root),
		Root:   root,
		eveDir: filepath.Join(root, ".eve"),
	}
}

func (repo repository) configPath() string   { return filepath.Join(repo.eveDir, "config.json") }
func (repo repository) snapshotsDir() string { return filepath.Join(repo.eveDir, "snapshots") }
func (repo repository) artifactsDir() string { return filepath.Join(repo.eveDir, "artifacts") }
func (repo repository) cacheDir() string     { return filepath.Join(repo.eveDir, "cache") }

func (repo repository) ensureDirs() error {
	for _, dir := range []string{repo.snapshotsDir(), repo.artifactsDir(), repo.cacheDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

func (repo repository) snapshotPath(id string) string {
	return filepath.Join(repo.snapshotsDir(), id+".json")
}

func (repo repository) loadSnapshot(id string) (*eve.Snapshot, error) {
	return eve.LoadSnapshotFile(repo.snapshotPath(id))
}

func (repo repository) saveSnapshot(snapshot *eve.Snapshot) error {
	if err := repo.ensureDirs(); err != nil {
		return err
	}
	canonical, err := eve.CanonicalSnapshotJSON(snapshot)
	if err != nil {
		return err
	}
	if err := os.WriteFile(repo.snapshotPath(snapshot.ID), append(canonical, '\n'), 0o644); err != nil {
		return fmt.Errorf("write snapshot %s: %w", snapshot.ID, err)
	}
	return repo.rebuildCache()
}

func (repo repository) listSnapshots(snapshotType string) ([]*eve.Snapshot, error) {
	entries, err := os.ReadDir(repo.snapshotsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshots: %w", err)
	}
	var snapshots []*eve.Snapshot
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		snapshot, err := eve.LoadSnapshotFile(filepath.Join(repo.snapshotsDir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		if snapshotType != "" && snapshot.Type != snapshotType {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt == snapshots[j].CreatedAt {
			return snapshots[i].ID < snapshots[j].ID
		}
		return snapshots[i].CreatedAt > snapshots[j].CreatedAt
	})
	return snapshots, nil
}

func (repo repository) summary() (repoSummary, error) {
	snapshots, err := repo.listSnapshots("")
	if err != nil {
		return repoSummary{}, err
	}
	summary := repoSummary{ID: repo.ID, Root: repo.Root, SnapshotCount: len(snapshots)}
	if facts, err := deriveGitFacts(repo); err == nil {
		summary.Branch = facts.Branch
		summary.Head = facts.GitState
		summary.Dirty = facts.Dirty
	}
	if remote, err := gitOutput(repo.Root, "remote", "get-url", "origin"); err == nil {
		summary.RemoteURL = normalizeRemoteURL(remote)
	}
	if len(snapshots) > 0 {
		summary.LatestAt = snapshots[0].CreatedAt
		summary.LatestSnapshot = snapshots[0].ID
		summary.LatestTitle = snapshots[0].Title
		summary.LatestGitState = snapshots[0].Implementation.GitState
	}
	return summary, nil
}

func (repo repository) detail() (repoDetail, error) {
	summary, err := repo.summary()
	if err != nil {
		return repoDetail{}, err
	}
	detail := repoDetail{repoSummary: summary}
	for _, name := range []string{"README.md", "README"} {
		data, err := os.ReadFile(filepath.Join(repo.Root, name))
		if err == nil {
			detail.Readme = string(data)
			break
		}
	}
	detail.PrimaryLanguage = detectPrimaryLanguage(repo.Root)
	detail.SizeBytes = repositorySize(repo.Root)
	if createdAt, err := gitOutput(repo.Root, "log", "--reverse", "--format=%cI"); err == nil {
		detail.CreatedAt = firstNonEmptyLine(createdAt)
	}
	return detail, nil
}

func detectPrimaryLanguage(root string) string {
	checks := []struct {
		Path     string
		Language string
	}{
		{"go.mod", "Go"},
		{"package.json", "TypeScript"},
		{"Cargo.toml", "Rust"},
		{"pyproject.toml", "Python"},
		{"Gemfile", "Ruby"},
	}
	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(root, check.Path)); err == nil {
			return check.Language
		}
	}
	return "Unknown"
}

func repositorySize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "ui_dist", "cache":
				return filepath.SkipDir
			}
			if current == filepath.Join(root, ".eve", "cache") {
				return filepath.SkipDir
			}
			return nil
		}
		if info, err := entry.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func firstNonEmptyLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeRemoteURL(remote string) string {
	value := strings.TrimSpace(remote)
	value = strings.TrimSuffix(value, ".git")
	if strings.HasPrefix(value, "git@github.com:") {
		return "https://github.com/" + strings.TrimPrefix(value, "git@github.com:")
	}
	if strings.HasPrefix(value, "ssh://git@github.com/") {
		return "https://github.com/" + strings.TrimPrefix(value, "ssh://git@github.com/")
	}
	return value
}

func (repo repository) rebuildCache() error {
	summary, err := repo.summary()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(repo.cacheDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repo.cacheDir(), "index.json"), append(data, '\n'), 0o644)
}

type gitFacts struct {
	Branch     string   `json:"branch"`
	GitState   string   `json:"gitState"`
	BaseCommit string   `json:"baseCommit,omitempty"`
	Commits    []string `json:"commits"`
	Dirty      bool     `json:"dirty"`
}

func deriveGitFacts(repo repository) (gitFacts, error) {
	branch, err := gitOutput(repo.Root, "branch", "--show-current")
	if err != nil {
		return gitFacts{}, err
	}
	head, err := gitOutput(repo.Root, "rev-parse", "HEAD")
	if err != nil {
		return gitFacts{}, err
	}
	status, err := gitOutput(repo.Root, "status", "--porcelain")
	if err != nil {
		return gitFacts{}, err
	}
	commitsText, err := gitOutput(repo.Root, "log", "--format=%H", "-n", "50")
	if err != nil {
		return gitFacts{}, err
	}
	commits := []string{}
	for _, line := range strings.Split(commitsText, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			commits = append(commits, trimmed)
		}
	}
	return gitFacts{
		Branch:   strings.TrimSpace(branch),
		GitState: strings.TrimSpace(head),
		Commits:  commits,
		Dirty:    strings.TrimSpace(status) != "",
	}, nil
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

type openEditorResponse struct {
	Repository string `json:"repository"`
	Root       string `json:"root"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
}

func openRepositoryInEditor(repo repository) openEditorResponse {
	response := openEditorResponse{Repository: repo.ID, Root: repo.Root}
	for _, candidate := range editorOpenCandidates(repo.Root) {
		cmd := exec.Command(candidate[0], candidate[1:]...)
		output, err := cmd.CombinedOutput()
		response.Command = strings.Join(candidate, " ")
		if err == nil {
			response.ExitCode = 0
			response.Stdout = strings.TrimSpace(string(output))
			return response
		}
		response.ExitCode = 1
		response.Stderr = strings.TrimSpace(fmt.Sprintf("%v\n%s", err, output))
	}
	if response.Command == "" {
		response.ExitCode = 1
		response.Stderr = "No supported editor launcher was found. Install a CLI such as code, cursor, or zed."
	}
	return response
}

func editorOpenCandidates(root string) [][]string {
	var candidates [][]string
	for _, name := range []string{"code", "cursor", "zed", "subl"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, []string{path, root})
		}
	}
	if path, err := exec.LookPath("open"); err == nil {
		for _, app := range []string{"Visual Studio Code", "Cursor", "Zed", "Sublime Text", "Xcode"} {
			candidates = append(candidates, []string{path, "-a", app, root})
		}
	}
	return candidates
}

func printSnapshot(w io.Writer, snapshot *eve.Snapshot, repo repository) {
	fmt.Fprintln(w, "Snapshot")
	fmt.Fprintf(w, "%s\n", snapshot.Title)
	fmt.Fprintf(w, "%s\n", snapshot.Summary)
	fmt.Fprintf(w, "Repository: %s\n", repo.ID)
	fmt.Fprintf(w, "Commit: %s\n", snapshot.Implementation.GitState)
	if len(snapshot.Validation) > 0 {
		fmt.Fprintln(w, "Validation")
		for _, validation := range snapshot.Validation {
			fmt.Fprintf(w, "- %s: %s\n", validation.Status, validation.Command)
		}
	}
}

type checkoutResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
}

func checkoutSnapshot(repo repository, snapshot *eve.Snapshot, force bool) checkoutResponse {
	response := checkoutResponse{
		ID:         snapshot.ID,
		Title:      snapshot.Title,
		Repository: repo.ID,
		Commit:     snapshot.Implementation.GitState,
		Command:    "eve checkout " + snapshot.ID,
	}
	if strings.TrimSpace(snapshot.Implementation.GitState) == "" {
		response.ExitCode = 1
		response.Stderr = fmt.Sprintf("Snapshot %s has no implementation.gitState.\n", snapshot.ID)
		return response
	}
	if !force {
		status, err := gitOutput(repo.Root, "status", "--porcelain")
		if err != nil {
			response.ExitCode = 1
			response.Stderr = fmt.Sprintf("check working tree: %v\n", err)
			return response
		}
		if strings.TrimSpace(status) != "" {
			response.ExitCode = 1
			response.Stderr = "Working tree has uncommitted changes.\nUse --force to checkout anyway.\n"
			return response
		}
	}
	cmd := exec.Command("git", "checkout", snapshot.Implementation.GitState)
	cmd.Dir = repo.Root
	output, err := cmd.CombinedOutput()
	if err != nil {
		response.ExitCode = 1
		response.Stderr = fmt.Sprintf("git checkout %s: %v\n%s", snapshot.Implementation.GitState, err, output)
		return response
	}
	response.Stdout = fmt.Sprintf("Product snapshot restored\nRepository: %s\nCommit: %s\n", repo.ID, snapshot.Implementation.GitState)
	return response
}

func newSnapshotID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "snap_" + time.Now().UTC().Format("20060102150405")
	}
	return "snap_" + hex.EncodeToString(raw[:])
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func isLocalhostAddr(addr string) bool {
	host := strings.Split(strings.TrimSpace(addr), ":")[0]
	return host == "localhost" || host == "127.0.0.1" || host == "[::1]" || host == ""
}
