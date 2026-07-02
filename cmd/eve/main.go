package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nhestrompia/eve"
)

const configFileVersion = 1

var allowedWorkflowTypes = map[string]struct{}{
	"feature":  {},
	"fix":      {},
	"refactor": {},
	"docs":     {},
	"test":     {},
	"chore":    {},
}

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
	case "add":
		return runAdd(args[1:], stdout, stderr)
	case "status":
		return runStatus(args[1:], stdout, stderr)
	case "commit":
		return runEVECommit(args[1:], stdout, stderr)
	case "snapshot":
		return runSnapshot(args[1:], stdout, stderr)
	case "checkout":
		return runCheckout(args[1:], stdout, stderr)
	case "list", "ls":
		return runList(args[1:], stdout, stderr)
	case "show":
		return runShow(args[1:], stdout, stderr)
	case "timeline":
		return runTimeline(args[1:], stdout, stderr)
	case "graph":
		return runGraph(args[1:], stdout, stderr)
	case "search":
		return runSearch(args[1:], stdout, stderr)
	case "ui":
		return runUI(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "canonicalize":
		return runCanonicalize(args[1:], stdout, stderr)
	case "version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "eve version takes no arguments")
			return 2
		}
		fmt.Fprintf(stdout, "eve %s (protocol v%d)\n", eve.CLIVersion, eve.ProtocolVersion)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
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

	store := newStore()
	if err := os.MkdirAll(store.stagedDir(), 0o755); err != nil {
		fmt.Fprintf(stderr, "create staged directory: %v\n", err)
		return 2
	}
	if err := os.MkdirAll(store.evolutionsDir(), 0o755); err != nil {
		fmt.Fprintf(stderr, "create evolutions directory: %v\n", err)
		return 2
	}
	if err := os.MkdirAll(store.sessionsRootDir(), 0o755); err != nil {
		fmt.Fprintf(stderr, "create sessions directory: %v\n", err)
		return 2
	}
	config := map[string]any{
		"eve": map[string]any{
			"version": eve.ProtocolVersion,
		},
		"config_version": configFileVersion,
		"created_at":     nowUTC(),
	}
	if _, err := os.Stat(store.configPath()); errors.Is(err, os.ErrNotExist) {
		data, marshalErr := json.MarshalIndent(config, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(stderr, "marshal config: %v\n", marshalErr)
			return 2
		}
		if err := os.WriteFile(store.configPath(), append(data, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "write config: %v\n", err)
			return 2
		}
	}

	fmt.Fprintf(stdout, "Initialized EVE in %s\n", store.root)
	return 0
}

func runAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runAddCompact(args, stdout, stderr)
	}

	switch args[0] {
	case "title":
		return runAddTitle(args[1:], stdout, stderr)
	case "behavior":
		return runAddBehavior(args[1:], stdout, stderr)
	case "verification":
		return runAddVerification(args[1:], stdout, stderr)
	case "session":
		return runAddSession(args[1:], stdout, stderr)
	case "outcome":
		return runAddOutcome(args[1:], stdout, stderr)
	case "implementation":
		return runAddImplementation(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eve add target %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runAddCompact(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	title := fs.String("title", "", "evolution title")
	evolutionType := fs.String("type", "", "evolution type")
	outcome := fs.String("outcome", "", "evolution outcome")
	sessionSource := fs.String("session-source", "", "session source transcript")
	sanitize := fs.Bool("sanitize", true, "sanitize session transcript")
	implementation := fs.String("implementation", "", "implementation snapshot commit")
	repository := fs.String("repository", currentRepositoryName(), "repository name")
	repositoryStatus := fs.String("repository-status", "merged", "repository status")
	added := repeatedFlag{}
	changed := repeatedFlag{}
	removed := repeatedFlag{}
	fixed := repeatedFlag{}
	verification := repeatedFlag{}
	sessions := repeatedFlag{}
	fs.Var(&added, "behavior-added", "added behavior")
	fs.Var(&changed, "behavior-changed", "changed behavior")
	fs.Var(&removed, "behavior-removed", "removed behavior")
	fs.Var(&fixed, "behavior-fixed", "fixed behavior")
	fs.Var(&verification, "verification", "verification claim, e.g. passed: go test ./...")
	fs.Var(&sessions, "session", "session reference provider:id")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve add compact form does not accept positional arguments")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		var messages []string
		if strings.TrimSpace(*title) != "" {
			evolution.Metadata.Title = strings.TrimSpace(*title)
			evolution.Intent = strings.TrimSpace(*title)
			messages = append(messages, "title")
		}
		if strings.TrimSpace(*evolutionType) != "" {
			evolution.Metadata.Type = strings.TrimSpace(*evolutionType)
			messages = append(messages, "type")
		}
		if strings.TrimSpace(*outcome) != "" {
			evolution.Outcome = strings.TrimSpace(*outcome)
			messages = append(messages, "outcome")
		}
		addBehaviorClaims(evolution, "added", added)
		addBehaviorClaims(evolution, "changed", changed)
		addBehaviorClaims(evolution, "removed", removed)
		addBehaviorClaims(evolution, "fixed", fixed)
		if len(added)+len(changed)+len(removed)+len(fixed) > 0 {
			messages = append(messages, "behavior")
		}
		for _, claim := range verification {
			parsed, err := parseVerificationClaim(claim)
			if err != nil {
				return "", err
			}
			evolution.Verification = append(evolution.Verification, parsed)
			messages = append(messages, "verification")
		}
		for i, session := range sessions {
			source := ""
			if i == 0 {
				source = strings.TrimSpace(*sessionSource)
			}
			if err := addSessionReference(store, evolution, session, source, *sanitize); err != nil {
				return "", err
			}
			messages = append(messages, "session")
		}
		if strings.TrimSpace(*implementation) != "" {
			commit, err := resolveCommit(strings.TrimSpace(*implementation))
			if err != nil {
				return "", err
			}
			setImplementationSnapshot(evolution, commit, strings.TrimSpace(*repository), strings.TrimSpace(*repositoryStatus))
			messages = append(messages, "implementation")
		}
		addTimeline(evolution, "staged", "Staged product meaning.")
		if len(messages) == 0 {
			return "", fmt.Errorf("eve add requires at least one field to stage")
		}
		return "Staged: " + strings.Join(uniqueStrings(messages), ", ") + "\n", nil
	})
}

func runAddTitle(args []string, stdout io.Writer, stderr io.Writer) int {
	title, evolutionType, err := parseTitleArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if title == "" {
		fmt.Fprintln(stderr, "eve add title requires a title")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		evolution.Metadata.Title = title
		evolution.Intent = title
		if strings.TrimSpace(evolutionType) != "" {
			evolution.Metadata.Type = strings.TrimSpace(evolutionType)
		}
		addTimeline(evolution, "title_staged", "Staged title.")
		return "Staged title\n", nil
	})
}

func runAddBehavior(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add behavior", flag.ContinueOnError)
	fs.SetOutput(stderr)
	added := fs.String("added", "", "added behavior")
	changed := fs.String("changed", "", "changed behavior")
	removed := fs.String("removed", "", "removed behavior")
	fixed := fs.String("fixed", "", "fixed behavior")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		count := 0
		for _, item := range []struct {
			kind  string
			value string
		}{
			{"added", *added},
			{"changed", *changed},
			{"removed", *removed},
			{"fixed", *fixed},
		} {
			if strings.TrimSpace(item.value) == "" {
				continue
			}
			addBehaviorClaims(evolution, item.kind, []string{item.value})
			count++
		}
		if count == 0 {
			return "", fmt.Errorf("eve add behavior requires --added, --changed, --removed, or --fixed")
		}
		addTimeline(evolution, "behavior_staged", "Staged behavior.")
		return fmt.Sprintf("Staged %d behavior claim(s)\n", count), nil
	})
}

func runAddVerification(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add verification", flag.ContinueOnError)
	fs.SetOutput(stderr)
	status := fs.String("status", "", "verification status")
	reference := fs.String("reference", "", "verification reference")
	verificationType := fs.String("type", "tests", "verification type")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*status) == "" || strings.TrimSpace(*reference) == "" {
		fmt.Fprintln(stderr, "eve add verification requires --status and --reference")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		evolution.Verification = append(evolution.Verification, eve.Verification{
			Type:      strings.TrimSpace(*verificationType),
			Status:    strings.TrimSpace(*status),
			Reference: strings.TrimSpace(*reference),
		})
		addTimeline(evolution, "verification_staged", "Staged verification.")
		return "Staged verification\n", nil
	})
}

func runAddSession(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add session", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("source", "", "session source transcript")
	sanitize := fs.Bool("sanitize", true, "sanitize session transcript")
	sessionRef := ""
	sessionArgs := args
	if len(sessionArgs) > 0 && !strings.HasPrefix(sessionArgs[0], "-") {
		sessionRef = sessionArgs[0]
		sessionArgs = sessionArgs[1:]
	}
	if err := fs.Parse(sessionArgs); err != nil {
		return 2
	}
	if sessionRef == "" && fs.NArg() == 1 {
		sessionRef = fs.Arg(0)
	}
	if sessionRef == "" || fs.NArg() > 1 {
		fmt.Fprintln(stderr, "eve add session requires provider:id")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		if err := addSessionReference(store, evolution, sessionRef, *source, *sanitize); err != nil {
			return "", err
		}
		addTimeline(evolution, "session_staged", "Staged session.")
		return "Staged session\n", nil
	})
}

func runAddOutcome(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add outcome", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	outcome := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if outcome == "" {
		fmt.Fprintln(stderr, "eve add outcome requires outcome text")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		evolution.Outcome = outcome
		addTimeline(evolution, "outcome_staged", "Staged outcome.")
		return "Staged outcome\n", nil
	})
}

func runAddImplementation(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("add implementation", flag.ContinueOnError)
	fs.SetOutput(stderr)
	commitRef := fs.String("commit", "", "commit ref")
	snapshotRef := fs.String("snapshot", "", "snapshot commit ref")
	repository := fs.String("repository", currentRepositoryName(), "repository name")
	status := fs.String("status", "merged", "repository status")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*commitRef) == "" && strings.TrimSpace(*snapshotRef) == "" {
		fmt.Fprintln(stderr, "eve add implementation requires --commit or --snapshot")
		return 2
	}

	return mutateStaged(stderr, stdout, func(store localStore, evolution *eve.Evolution) (string, error) {
		var messages []string
		if strings.TrimSpace(*commitRef) != "" {
			commit, err := resolveCommit(strings.TrimSpace(*commitRef))
			if err != nil {
				return "", err
			}
			addImplementationCommit(evolution, commit, strings.TrimSpace(*repository), strings.TrimSpace(*status))
			messages = append(messages, "commit "+commit)
		}
		if strings.TrimSpace(*snapshotRef) != "" {
			snapshot, err := resolveCommit(strings.TrimSpace(*snapshotRef))
			if err != nil {
				return "", err
			}
			setImplementationSnapshot(evolution, snapshot, strings.TrimSpace(*repository), strings.TrimSpace(*status))
			messages = append(messages, "snapshot "+snapshot)
		}
		addTimeline(evolution, "implementation_staged", "Staged implementation.")
		return "Staged implementation " + strings.Join(messages, ", ") + "\n", nil
	})
}

func runStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve status takes no arguments")
		return 2
	}

	store := newStore()
	if err := store.requireInitialized(); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	evolution, err := store.loadStaged()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	report := readinessReport(evolution, store.loadStagedSessionManifest())
	printStatus(stdout, evolution, report, store.nextID())
	if len(report.Missing) > 0 {
		return 1
	}
	return 0
}

func runEVECommit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve commit takes no arguments")
		return 2
	}

	store := newStore()
	if err := store.requireInitialized(); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	evolution, err := store.loadStaged()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	manifest := store.loadStagedSessionManifest()
	report := readinessReport(evolution, manifest)
	if len(report.Missing) > 0 {
		printStatus(stdout, evolution, report, store.nextID())
		return 1
	}

	id := store.nextID()
	finalPath := store.evolutionPath(id)
	if _, err := os.Stat(finalPath); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", finalPath)
		return 1
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "check evolution path: %v\n", err)
		return 1
	}

	now := nowUTC()
	evolution.Metadata.ID = id
	evolution.Metadata.Status = "completed"
	if evolution.Metadata.CreatedAt == "" {
		evolution.Metadata.CreatedAt = now
	}
	evolution.Metadata.UpdatedAt = now
	addTimeline(evolution, "committed", "Committed evolution.")
	if err := store.promoteStagedSessions(id, evolution); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := store.saveCommitted(evolution); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := os.RemoveAll(store.stagedDir()); err != nil {
		fmt.Fprintf(stderr, "clear staging area: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(store.stagedDir(), 0o755); err != nil {
		fmt.Fprintf(stderr, "recreate staging area: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Created %s %s\n", id, evolution.Metadata.Title)
	fmt.Fprintf(stdout, "Wrote %s\n", filepath.ToSlash(finalPath))
	if len(evolution.Sessions) > 0 {
		fmt.Fprintf(stdout, "Wrote %s\n", filepath.ToSlash(store.sessionDir(id)))
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "git add .eve/")
	fmt.Fprintf(stdout, "git commit -m %q\n", id+" "+evolution.Metadata.Title)
	return 0
}

func runSnapshot(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "repository name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve snapshot requires an evolution id")
		return 2
	}

	store := newStore()
	evolution, err := store.loadCommitted(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	target, err := resolveSnapshotTarget(evolution, *repo)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	printSnapshot(stdout, evolution, target)
	return 0
}

func runCheckout(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkout", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "repository name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve checkout requires an evolution id")
		return 2
	}

	if dirty, err := workingTreeDirty(); err != nil {
		fmt.Fprintf(stderr, "check working tree: %v\n", err)
		return 1
	} else if dirty {
		fmt.Fprintln(stderr, "Working tree has uncommitted changes.")
		fmt.Fprintf(stderr, "Commit or stash them before checking out %s.\n", fs.Arg(0))
		return 1
	}

	store := newStore()
	evolution, err := store.loadCommitted(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	target, err := resolveSnapshotTarget(evolution, *repo)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Checking out %s %s\n", evolution.Metadata.ID, evolution.Metadata.Title)
	fmt.Fprintf(stdout, "Repository: %s\n", target.Repository)
	fmt.Fprintf(stdout, "Commit: %s\n", target.Commit)
	cmd := exec.Command("git", "checkout", target.Commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(stderr, "git checkout %s: %v\n%s", target.Commit, err, output)
		return 1
	}
	fmt.Fprintln(stdout, "Product snapshot restored")
	fmt.Fprintln(stdout)
	printSnapshot(stdout, evolution, target)
	return 0
}

func runList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve list takes no arguments")
		return 2
	}

	store := newStore()
	evolutions, err := store.loadAllCommitted()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if len(evolutions) == 0 {
		fmt.Fprintln(stdout, "No evolutions found.")
		return 0
	}

	printEvolutionTable(stdout, evolutions)
	return 0
}

func runShow(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve show requires an evolution id")
		return 2
	}

	store := newStore()
	evolution, err := store.loadCommitted(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	printEvolution(stdout, store, evolution)
	return 0
}

func runTimeline(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("timeline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eve timeline requires an evolution id")
		return 2
	}

	store := newStore()
	evolution, err := store.loadCommitted(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "%s timeline\n", evolution.Metadata.ID)
	for _, entry := range evolution.Timeline {
		when := entry.Timestamp
		if parsed, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
			when = parsed.Local().Format("15:04")
		}
		fmt.Fprintf(stdout, "[%s] %s", when, humanEvent(entry.Event))
		if entry.Description != "" {
			fmt.Fprintf(stdout, " - %s", entry.Description)
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func runGraph(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	store := newStore()
	evolutions, err := store.loadAllCommitted()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if len(evolutions) == 0 {
		fmt.Fprintln(stdout, "No evolutions found.")
		return 0
	}

	byID := map[string]*eve.Evolution{}
	children := map[string][]string{}
	hasParent := map[string]bool{}
	for _, evolution := range evolutions {
		byID[evolution.Metadata.ID] = evolution
		for _, parent := range evolution.Relationships.Extends {
			children[parent] = append(children[parent], evolution.Metadata.ID)
			hasParent[evolution.Metadata.ID] = true
		}
	}

	var roots []string
	target := firstArgOrEmpty(fs.Args())
	if target != "" {
		if _, ok := byID[target]; !ok {
			fmt.Fprintf(stderr, "evolution %s not found\n", target)
			return 2
		}
		roots = []string{target}
	} else {
		for id := range byID {
			if !hasParent[id] {
				roots = append(roots, id)
			}
		}
	}
	sort.Strings(roots)
	for _, childIDs := range children {
		sort.Strings(childIDs)
	}

	for _, root := range roots {
		printGraphNode(stdout, byID, children, root, "", true, true, map[string]bool{})
	}
	return 0
}

func runSearch(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	query := strings.ToLower(strings.TrimSpace(strings.Join(fs.Args(), " ")))
	if query == "" {
		fmt.Fprintln(stderr, "eve search requires a query")
		return 2
	}

	store := newStore()
	evolutions, err := store.loadAllCommitted()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	var matches []*eve.Evolution
	for _, evolution := range evolutions {
		if evolutionMatches(evolution, query) {
			matches = append(matches, evolution)
		}
	}
	if len(matches) == 0 {
		fmt.Fprintf(stdout, "No evolutions matched %q.\n", query)
		return 0
	}

	printEvolutionTable(stdout, matches)
	return 0
}

func runValidate(paths []string, stdout io.Writer, stderr io.Writer) int {
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "eve validate requires at least one file")
		return 2
	}

	exitCode := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			exitCode = 2
			continue
		}

		if _, err := eve.Parse(data); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			if exitCode != 2 {
				exitCode = 1
			}
			continue
		}
		fmt.Fprintf(stdout, "%s: valid\n", path)
	}

	return exitCode
}

func runCanonicalize(paths []string, stdout io.Writer, stderr io.Writer) int {
	if len(paths) != 1 {
		fmt.Fprintln(stderr, "eve canonicalize requires exactly one file")
		return 2
	}

	data, err := os.ReadFile(paths[0])
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 2
	}

	evolution, err := eve.Parse(data)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 1
	}

	canonical, err := eve.CanonicalJSON(evolution)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 1
	}

	fmt.Fprintln(stdout, string(canonical))
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  eve init")
	fmt.Fprintln(w, "  eve add --title title --type type --behavior-added claim --outcome text --verification 'passed: command' --session provider:id --implementation HEAD")
	fmt.Fprintln(w, "  eve add title <title> --type feature")
	fmt.Fprintln(w, "  eve add behavior --added claim")
	fmt.Fprintln(w, "  eve add verification --status passed --reference command")
	fmt.Fprintln(w, "  eve add session provider:id --source transcript.jsonl --sanitize")
	fmt.Fprintln(w, "  eve add outcome <outcome>")
	fmt.Fprintln(w, "  eve add implementation --snapshot HEAD --commit HEAD --repository name --status merged")
	fmt.Fprintln(w, "  eve status")
	fmt.Fprintln(w, "  eve commit")
	fmt.Fprintln(w, "  eve snapshot EV-001")
	fmt.Fprintln(w, "  eve checkout EV-001 [--repo name]")
	fmt.Fprintln(w, "  eve list")
	fmt.Fprintln(w, "  eve show EV-001")
	fmt.Fprintln(w, "  eve timeline EV-001")
	fmt.Fprintln(w, "  eve graph [EV-001]")
	fmt.Fprintln(w, "  eve search <query>")
	fmt.Fprintln(w, "  eve ui [--addr localhost:4317] [--repo name] [--open=false]")
	fmt.Fprintln(w, "  eve validate <file...>")
	fmt.Fprintln(w, "  eve canonicalize <file>")
	fmt.Fprintln(w, "  eve version")
}

type localStore struct {
	root string
}

type configFile struct {
	EVE           eve.EVEHeader `json:"eve"`
	ConfigVersion int           `json:"config_version"`
	CreatedAt     string        `json:"created_at"`
}

type readiness struct {
	Missing []string
}

type snapshotTarget struct {
	Repository string
	Commit     string
}

type sessionManifest struct {
	EvolutionID string            `json:"evolution_id,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
	Sessions    []sessionArtifact `json:"sessions"`
}

type sessionArtifact struct {
	Provider   string            `json:"provider"`
	ID         string            `json:"id"`
	Title      string            `json:"title,omitempty"`
	AttachedAt string            `json:"attached_at"`
	Sanitized  bool              `json:"sanitized"`
	Format     string            `json:"format"`
	Transcript string            `json:"transcript"`
	Raw        string            `json:"raw"`
	Source     string            `json:"source,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type sessionExtension struct {
	Sessions []sessionArtifact `json:"sessions"`
}

type sessionConversation struct {
	Messages       []sessionMessage
	EventCount     int
	MessageCount   int
	UserMessages   int
	AgentMessages  int
	ToolCalls      int
	OmittedEvents  int
	FirstTimestamp string
	LastTimestamp  string
}

type sessionMessage struct {
	Role      string
	Text      string
	Timestamp string
}

type repeatedFlag []string

func (flag *repeatedFlag) String() string {
	return strings.Join(*flag, ",")
}

func (flag *repeatedFlag) Set(value string) error {
	*flag = append(*flag, value)
	return nil
}

func newStore() localStore {
	root := os.Getenv("EVE_DIR")
	if strings.TrimSpace(root) == "" {
		root = ".eve"
	}
	return localStore{root: root}
}

func (store localStore) configPath() string {
	return filepath.Join(store.root, "config.json")
}

func (store localStore) stagedDir() string {
	return filepath.Join(store.root, "staged")
}

func (store localStore) stagedPath() string {
	return filepath.Join(store.stagedDir(), "evolution.json")
}

func (store localStore) stagedSessionsDir() string {
	return filepath.Join(store.stagedDir(), "sessions")
}

func (store localStore) stagedSessionManifestPath() string {
	return filepath.Join(store.stagedSessionsDir(), "manifest.json")
}

func (store localStore) evolutionsDir() string {
	return filepath.Join(store.root, "evolutions")
}

func (store localStore) evolutionPath(id string) string {
	return filepath.Join(store.evolutionsDir(), id+".json")
}

func (store localStore) sessionsRootDir() string {
	return filepath.Join(store.root, "sessions")
}

func (store localStore) sessionDir(id string) string {
	return filepath.Join(store.sessionsRootDir(), id)
}

func (store localStore) sessionManifestPath(id string) string {
	return filepath.Join(store.sessionDir(id), "manifest.json")
}

func (store localStore) requireInitialized() error {
	if _, err := os.Stat(store.configPath()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("EVE is not initialized. Run `eve init` first.")
		}
		return fmt.Errorf("read EVE config: %w", err)
	}
	return nil
}

func (store localStore) ensureStagedDirs() error {
	if err := store.requireInitialized(); err != nil {
		return err
	}
	return os.MkdirAll(store.stagedSessionsDir(), 0o755)
}

func (store localStore) emptyStaged() *eve.Evolution {
	now := nowUTC()
	return &eve.Evolution{
		EVE: eve.EVEHeader{Version: eve.ProtocolVersion},
		Metadata: eve.Metadata{
			Status:    "draft",
			CreatedBy: "eve-cli",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Behavior:      eve.Behavior{},
		Decisions:     []json.RawMessage{},
		Risks:         []json.RawMessage{},
		Verification:  []eve.Verification{},
		Sessions:      []eve.Session{},
		Timeline:      []eve.TimelineEntry{},
		Relationships: eve.Relationships{},
		Implementation: eve.Implementation{
			Repositories: map[string]eve.Repository{},
		},
		Extensions: map[string]json.RawMessage{},
	}
}

func (store localStore) loadStaged() (*eve.Evolution, error) {
	data, err := os.ReadFile(store.stagedPath())
	if errors.Is(err, os.ErrNotExist) {
		return store.emptyStaged(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read staged evolution: %w", err)
	}
	return eve.Parse(data)
}

func (store localStore) saveStaged(evolution *eve.Evolution) error {
	if err := store.ensureStagedDirs(); err != nil {
		return err
	}
	evolution.Metadata.UpdatedAt = nowUTC()
	canonical, err := eve.CanonicalJSON(evolution)
	if err != nil {
		return err
	}
	if err := os.WriteFile(store.stagedPath(), append(canonical, '\n'), 0o644); err != nil {
		return fmt.Errorf("write staged evolution: %w", err)
	}
	return nil
}

func (store localStore) saveCommitted(evolution *eve.Evolution) error {
	if err := os.MkdirAll(store.evolutionsDir(), 0o755); err != nil {
		return fmt.Errorf("create evolutions directory: %w", err)
	}
	canonical, err := eve.CanonicalJSON(evolution)
	if err != nil {
		return err
	}
	if err := os.WriteFile(store.evolutionPath(evolution.Metadata.ID), append(canonical, '\n'), 0o644); err != nil {
		return fmt.Errorf("write evolution %s: %w", evolution.Metadata.ID, err)
	}
	return nil
}

func (store localStore) loadCommitted(id string) (*eve.Evolution, error) {
	return eve.LoadFile(store.evolutionPath(id))
}

func (store localStore) loadAllCommitted() ([]*eve.Evolution, error) {
	entries, err := os.ReadDir(store.evolutionsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read evolutions: %w", err)
	}

	var evolutions []*eve.Evolution
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		evolution, err := eve.LoadFile(filepath.Join(store.evolutionsDir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		evolutions = append(evolutions, evolution)
	}
	sort.Slice(evolutions, func(i, j int) bool {
		return evolutions[i].Metadata.ID < evolutions[j].Metadata.ID
	})
	return evolutions, nil
}

func (store localStore) nextID() string {
	maxID := 0
	entries, err := os.ReadDir(store.evolutionsDir())
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			n, ok := parseEvolutionNumber(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			if ok && n > maxID {
				maxID = n
			}
		}
	}
	return fmt.Sprintf("EV-%03d", maxID+1)
}

func parseEvolutionNumber(id string) (int, bool) {
	if !strings.HasPrefix(id, "EV-") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, "EV-"))
	return n, err == nil
}

func mutateStaged(stderr io.Writer, stdout io.Writer, mutate func(localStore, *eve.Evolution) (string, error)) int {
	store := newStore()
	evolution, err := store.loadStaged()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	message, err := mutate(store, evolution)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := store.saveStaged(evolution); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	fmt.Fprint(stdout, message)
	return 0
}

func addBehaviorClaims(evolution *eve.Evolution, kind string, claims []string) {
	for _, claim := range claims {
		claim = strings.TrimSpace(claim)
		if claim == "" {
			continue
		}
		switch kind {
		case "added":
			evolution.Behavior.Added = append(evolution.Behavior.Added, eve.BehaviorClaim{Description: claim})
		case "changed":
			evolution.Behavior.Changed = append(evolution.Behavior.Changed, eve.BehaviorClaim{Description: claim})
		case "removed":
			evolution.Behavior.Removed = append(evolution.Behavior.Removed, eve.BehaviorClaim{Description: claim})
		case "fixed":
			evolution.Behavior.Fixed = append(evolution.Behavior.Fixed, eve.BehaviorClaim{Description: claim})
		}
	}
}

func parseTitleArgs(args []string) (string, string, error) {
	var titleParts []string
	evolutionType := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing value for --type")
			}
			evolutionType = strings.TrimSpace(args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", fmt.Errorf("unknown flag %s", args[i])
			}
			titleParts = append(titleParts, args[i])
		}
	}
	return strings.TrimSpace(strings.Join(titleParts, " ")), evolutionType, nil
}

func parseVerificationClaim(claim string) (eve.Verification, error) {
	parts := strings.SplitN(claim, ":", 2)
	if len(parts) != 2 {
		return eve.Verification{}, fmt.Errorf("verification must be formatted as `status: reference`")
	}
	return eve.Verification{
		Type:      "tests",
		Status:    strings.TrimSpace(parts[0]),
		Reference: strings.TrimSpace(parts[1]),
	}, nil
}

func addSessionReference(store localStore, evolution *eve.Evolution, reference string, source string, sanitize bool) error {
	provider, sessionID, err := parseSessionRef(reference)
	if err != nil {
		return err
	}
	session := eve.Session{Provider: provider, ID: sessionID}
	if strings.TrimSpace(source) != "" {
		raw, resolvedSource, resolvedID, err := loadSessionExport(provider, sessionID, source, "", sanitize)
		if err != nil {
			return err
		}
		if sanitize {
			raw = sanitizeSession(raw)
		}
		artifact, err := store.writeSessionArtifacts(store.stagedSessionsDir(), store.stagedSessionManifestPath(), "", provider, resolvedID, "", detectRawFormat(resolvedSource, raw), raw, sanitize, resolvedSource)
		if err != nil {
			return err
		}
		session.ID = resolvedID
		session.URI = artifact.Transcript
		if err := upsertSessionExtension(evolution, artifact); err != nil {
			return err
		}
	}
	evolution.Sessions = upsertSession(evolution.Sessions, session)
	return nil
}

func parseSessionRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("session must be formatted as provider:id")
	}
	provider := normalizeProvider(parts[0])
	if !isSupportedSessionProvider(provider) {
		return "", "", fmt.Errorf("unsupported session provider %q; supported providers: codex, claude, opencode, pi", parts[0])
	}
	return provider, strings.TrimSpace(parts[1]), nil
}

func addImplementationCommit(evolution *eve.Evolution, commit string, repository string, status string) {
	ensureImplementationRepository(evolution, repository, status)
	evolution.Implementation.Commits = appendUnique(evolution.Implementation.Commits, commit)
}

func setImplementationSnapshot(evolution *eve.Evolution, snapshot string, repository string, status string) {
	ensureImplementationRepository(evolution, repository, status)
	evolution.Implementation.Snapshot = snapshot
}

func ensureImplementationRepository(evolution *eve.Evolution, repository string, status string) {
	if repository == "" {
		repository = currentRepositoryName()
	}
	if status == "" {
		status = "merged"
	}
	if evolution.Implementation.Repositories == nil {
		evolution.Implementation.Repositories = map[string]eve.Repository{}
	}
	evolution.Implementation.Repositories[repository] = eve.Repository{Status: status}
}

func resolveCommit(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty commit ref")
	}
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve commit %q: %w", ref, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func readinessReport(evolution *eve.Evolution, manifest sessionManifest) readiness {
	var missing []string
	if strings.TrimSpace(evolution.Metadata.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(evolution.Metadata.Type) == "" {
		missing = append(missing, "type")
	} else if _, ok := allowedWorkflowTypes[evolution.Metadata.Type]; !ok {
		missing = append(missing, "type must be one of feature, fix, refactor, docs, test, chore")
	}
	if strings.TrimSpace(evolution.Outcome) == "" {
		missing = append(missing, "outcome")
	}
	if behaviorCount(evolution) == 0 {
		missing = append(missing, "behavior")
	}
	if !hasVerification(evolution) {
		missing = append(missing, "verification")
	}
	if len(evolution.Sessions) == 0 && len(manifest.Sessions) == 0 {
		missing = append(missing, "session")
	}
	if strings.TrimSpace(evolution.Implementation.Snapshot) == "" {
		missing = append(missing, "implementation snapshot")
	}
	return readiness{Missing: missing}
}

func printStatus(w io.Writer, evolution *eve.Evolution, report readiness, nextID string) {
	fmt.Fprintln(w, "EVE status")
	fmt.Fprintf(w, "Next ID: %s\n", nextID)
	fmt.Fprintf(w, "Title: %s\n", fallback(evolution.Metadata.Title, "(missing)"))
	fmt.Fprintf(w, "Type: %s\n", fallback(evolution.Metadata.Type, "(missing)"))
	fmt.Fprintf(w, "Outcome: %s\n", presentMissing(evolution.Outcome))
	fmt.Fprintf(w, "Behavior: %d\n", behaviorCount(evolution))
	fmt.Fprintf(w, "Verification: %d\n", len(evolution.Verification))
	fmt.Fprintf(w, "Sessions: %d\n", len(evolution.Sessions))
	fmt.Fprintf(w, "Implementation snapshot: %s\n", presentMissing(evolution.Implementation.Snapshot))
	fmt.Fprintf(w, "Contributed commits: %d\n", len(evolution.Implementation.Commits))
	if len(report.Missing) == 0 {
		fmt.Fprintln(w, "Ready: yes")
		fmt.Fprintf(w, "Commit message: %s %s\n", nextID, evolution.Metadata.Title)
		return
	}
	fmt.Fprintln(w, "Ready: no")
	fmt.Fprintln(w, "Missing:")
	for _, item := range report.Missing {
		fmt.Fprintf(w, "- %s\n", item)
	}
}

func behaviorCount(evolution *eve.Evolution) int {
	return len(evolution.Behavior.Added) + len(evolution.Behavior.Changed) + len(evolution.Behavior.Removed) + len(evolution.Behavior.Fixed)
}

func hasVerification(evolution *eve.Evolution) bool {
	for _, verification := range evolution.Verification {
		if verification.Status != "" && verification.Reference != "" {
			return true
		}
	}
	return false
}

func presentMissing(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(missing)"
	}
	return "present"
}

func (store localStore) writeSessionArtifacts(dir string, manifestPath string, evolutionID string, provider string, sessionID string, title string, rawFormat string, raw []byte, sanitized bool, source string) (sessionArtifact, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return sessionArtifact{}, fmt.Errorf("create session artifact directory: %w", err)
	}
	base := safeArtifactName(provider + "-" + sessionID)
	if base == "" {
		base = provider + "-session"
	}
	rawFormat = strings.Trim(strings.TrimSpace(rawFormat), ".")
	if rawFormat == "" {
		rawFormat = "jsonl"
	}
	rawPath := filepath.Join(dir, base+"."+rawFormat)
	mdPath := filepath.Join(dir, base+".md")
	if err := os.WriteFile(rawPath, raw, 0o644); err != nil {
		return sessionArtifact{}, fmt.Errorf("write raw session artifact: %w", err)
	}
	markdown := renderSessionMarkdown(provider, sessionID, title, rawFormat, raw, sanitized, source)
	if err := os.WriteFile(mdPath, markdown, 0o644); err != nil {
		return sessionArtifact{}, fmt.Errorf("write markdown session artifact: %w", err)
	}
	artifact := sessionArtifact{
		Provider:   provider,
		ID:         sessionID,
		Title:      strings.TrimSpace(title),
		AttachedAt: nowUTC(),
		Sanitized:  sanitized,
		Format:     "md",
		Transcript: filepath.ToSlash(mdPath),
		Raw:        filepath.ToSlash(rawPath),
		Source:     source,
		Metadata: map[string]string{
			"raw_format": rawFormat,
		},
	}
	if artifact.Title == "" {
		artifact.Title = provider + " " + sessionID
	}
	if err := upsertSessionManifest(manifestPath, evolutionID, artifact); err != nil {
		return sessionArtifact{}, err
	}
	return artifact, nil
}

func upsertSessionManifest(path string, evolutionID string, artifact sessionArtifact) error {
	manifest := sessionManifest{EvolutionID: evolutionID, Sessions: []sessionArtifact{}}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("parse session manifest: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read session manifest: %w", err)
	}
	manifest.EvolutionID = evolutionID
	manifest.UpdatedAt = nowUTC()
	replaced := false
	for i, existing := range manifest.Sessions {
		if existing.Provider == artifact.Provider && existing.ID == artifact.ID {
			manifest.Sessions[i] = artifact
			replaced = true
			break
		}
	}
	if !replaced {
		manifest.Sessions = append(manifest.Sessions, artifact)
	}
	sortSessionArtifacts(manifest.Sessions)
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session manifest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write session manifest: %w", err)
	}
	return nil
}

func (store localStore) loadStagedSessionManifest() sessionManifest {
	return loadSessionManifest(store.stagedSessionManifestPath())
}

func (store localStore) loadSessionManifest(id string) sessionManifest {
	return loadSessionManifest(store.sessionManifestPath(id))
}

func loadSessionManifest(path string) sessionManifest {
	var manifest sessionManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return sessionManifest{}
	}
	return manifest
}

func (store localStore) promoteStagedSessions(id string, evolution *eve.Evolution) error {
	manifest := store.loadStagedSessionManifest()
	if len(manifest.Sessions) == 0 {
		return nil
	}
	if err := os.MkdirAll(store.sessionDir(id), 0o755); err != nil {
		return fmt.Errorf("create final session directory: %w", err)
	}
	var finalArtifacts []sessionArtifact
	for _, artifact := range manifest.Sessions {
		rawName := filepath.Base(artifact.Raw)
		mdName := filepath.Base(artifact.Transcript)
		finalRaw := filepath.Join(store.sessionDir(id), rawName)
		finalMD := filepath.Join(store.sessionDir(id), mdName)
		if err := copyFile(artifact.Raw, finalRaw); err != nil {
			return err
		}
		if err := copyFile(artifact.Transcript, finalMD); err != nil {
			return err
		}
		artifact.Raw = filepath.ToSlash(finalRaw)
		artifact.Transcript = filepath.ToSlash(finalMD)
		finalArtifacts = append(finalArtifacts, artifact)
		for i, session := range evolution.Sessions {
			if session.Provider == artifact.Provider && session.ID == artifact.ID {
				evolution.Sessions[i].URI = artifact.Transcript
			}
		}
	}
	manifest.EvolutionID = id
	manifest.UpdatedAt = nowUTC()
	manifest.Sessions = finalArtifacts
	sortSessionArtifacts(manifest.Sessions)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal final session manifest: %w", err)
	}
	if err := os.WriteFile(store.sessionManifestPath(id), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write final session manifest: %w", err)
	}
	extension := sessionExtension{Sessions: finalArtifacts}
	raw, err := json.Marshal(extension)
	if err != nil {
		return fmt.Errorf("marshal session extension: %w", err)
	}
	if evolution.Extensions == nil {
		evolution.Extensions = map[string]json.RawMessage{}
	}
	evolution.Extensions["eve.sessions"] = raw
	return nil
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func sortSessionArtifacts(artifacts []sessionArtifact) {
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Provider == artifacts[j].Provider {
			return artifacts[i].ID < artifacts[j].ID
		}
		return artifacts[i].Provider < artifacts[j].Provider
	})
}

func loadSessionExport(provider string, sessionID string, source string, providerRoot string, sanitize bool) ([]byte, string, string, error) {
	if strings.TrimSpace(source) != "" {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, "", "", fmt.Errorf("read session source %q: %w", source, err)
		}
		return data, source, normalizeSessionID(provider, sessionID, source), nil
	}
	switch provider {
	case "opencode":
		return exportOpenCodeSession(sessionID, sanitize)
	case "codex":
		return loadLatestSessionFile(provider, sessionID, providerRoot, codexSessionRoots())
	case "claude":
		return loadLatestSessionFile(provider, sessionID, providerRoot, claudeSessionRoots())
	case "pi":
		return loadLatestSessionFile(provider, sessionID, providerRoot, piSessionRoots())
	default:
		return nil, "", "", fmt.Errorf("unsupported session provider %q", provider)
	}
}

func exportOpenCodeSession(sessionID string, sanitize bool) ([]byte, string, string, error) {
	resolvedID := sessionID
	if sessionID == "latest" {
		latest, err := latestOpenCodeSessionID()
		if err != nil {
			return nil, "", "", err
		}
		resolvedID = latest
	}
	args := []string{"export", resolvedID}
	if sanitize {
		args = append(args, "--sanitize")
	}
	output, err := exec.Command("opencode", args...).Output()
	if err != nil {
		return nil, "", "", fmt.Errorf("run opencode export: %w; pass --source with an existing export if opencode is unavailable", err)
	}
	return output, "opencode export " + resolvedID, resolvedID, nil
}

func latestOpenCodeSessionID() (string, error) {
	output, err := exec.Command("opencode", "session", "list", "--max-count", "1", "--format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("discover latest opencode session: %w; pass an explicit session id or --source", err)
	}
	var raw any
	if err := json.Unmarshal(output, &raw); err != nil {
		return "", fmt.Errorf("parse opencode session list: %w", err)
	}
	id := firstID(raw)
	if id == "" {
		return "", fmt.Errorf("opencode session list did not include a session id")
	}
	return id, nil
}

func loadLatestSessionFile(provider string, sessionID string, providerRoot string, roots []string) ([]byte, string, string, error) {
	if strings.TrimSpace(providerRoot) != "" {
		roots = []string{providerRoot}
	}
	path, err := findSessionFile(sessionID, roots)
	if err != nil {
		return nil, "", "", fmt.Errorf("%s session import: %w; pass --source with an exported transcript", provider, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", fmt.Errorf("read %s session %q: %w", provider, path, err)
	}
	return data, path, normalizeSessionID(provider, sessionID, path), nil
}

func findSessionFile(sessionID string, roots []string) (string, error) {
	var candidates []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".jsonl" && ext != ".json" && ext != ".md" && ext != ".txt" {
				return nil
			}
			if sessionID != "latest" && !strings.Contains(entry.Name(), sessionID) && !strings.Contains(path, sessionID) {
				return nil
			}
			candidates = append(candidates, path)
			return nil
		})
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no matching transcript files found")
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, leftErr := os.Stat(candidates[i])
		right, rightErr := os.Stat(candidates[j])
		if leftErr != nil || rightErr != nil {
			return candidates[i] < candidates[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	return candidates[0], nil
}

func codexSessionRoots() []string {
	home, _ := os.UserHomeDir()
	var roots []string
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		roots = append(roots, filepath.Join(codexHome, "sessions"))
	}
	if home != "" {
		roots = append(roots, filepath.Join(home, ".codex", "sessions"))
	}
	return roots
}

func claudeSessionRoots() []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".claude", "sessions"), filepath.Join(home, ".claude", "projects")}
}

func piSessionRoots() []string {
	home, _ := os.UserHomeDir()
	var roots []string
	if piHome := os.Getenv("PI_SESSION_DIR"); piHome != "" {
		roots = append(roots, piHome)
	}
	if home != "" {
		roots = append(roots, filepath.Join(home, ".pi", "sessions"))
	}
	return roots
}

func renderSessionMarkdown(provider string, sessionID string, title string, rawFormat string, raw []byte, sanitized bool, source string) []byte {
	var out strings.Builder
	heading := strings.TrimSpace(title)
	if heading == "" {
		heading = providerDisplayName(provider) + " " + sessionID
	}
	conversation := extractSessionConversation(rawFormat, raw)
	fmt.Fprintf(&out, "# %s\n\n", heading)
	fmt.Fprintf(&out, "- Provider: `%s`\n", provider)
	fmt.Fprintf(&out, "- Session: `%s`\n", sessionID)
	fmt.Fprintf(&out, "- Attached: `%s`\n", nowUTC())
	fmt.Fprintf(&out, "- Sanitized: `%t`\n", sanitized)
	if source != "" {
		fmt.Fprintf(&out, "- Source: `%s`\n", source)
	}
	out.WriteString("\n## Analytics\n\n")
	fmt.Fprintf(&out, "- Events scanned: `%d`\n", conversation.EventCount)
	fmt.Fprintf(&out, "- Messages shown: `%d`\n", conversation.MessageCount)
	fmt.Fprintf(&out, "- User messages: `%d`\n", conversation.UserMessages)
	fmt.Fprintf(&out, "- Agent messages: `%d`\n", conversation.AgentMessages)
	fmt.Fprintf(&out, "- Tool calls: `%d`\n", conversation.ToolCalls)
	fmt.Fprintf(&out, "- Omitted system/tool/log events: `%d`\n", conversation.OmittedEvents)
	if conversation.FirstTimestamp != "" {
		fmt.Fprintf(&out, "- First event: `%s`\n", conversation.FirstTimestamp)
	}
	if conversation.LastTimestamp != "" {
		fmt.Fprintf(&out, "- Last event: `%s`\n", conversation.LastTimestamp)
	}
	out.WriteString("\n## Messages\n\n")
	if len(conversation.Messages) == 0 {
		out.WriteString("No user-facing messages could be extracted. The raw artifact is still stored separately for audit.\n")
		return []byte(out.String())
	}
	for _, message := range conversation.Messages {
		fmt.Fprintf(&out, "### %s", titleCase(message.Role))
		if message.Timestamp != "" {
			fmt.Fprintf(&out, " `%s`", message.Timestamp)
		}
		out.WriteString("\n\n")
		out.WriteString(message.Text)
		if !strings.HasSuffix(message.Text, "\n") {
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func extractSessionConversation(rawFormat string, raw []byte) sessionConversation {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(rawFormat), ".")) {
	case "jsonl":
		return extractJSONLConversation(raw)
	case "json":
		return extractJSONConversation(raw)
	case "md", "markdown":
		return extractMarkdownConversation(raw)
	default:
		return extractTextConversation(raw)
	}
}

func extractJSONLConversation(raw []byte) sessionConversation {
	conversation := sessionConversation{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 1024), 1024*1024*10)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		conversation.EventCount++
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			conversation.OmittedEvents++
			continue
		}
		conversation.addEvent(event)
	}
	return conversation
}

func extractJSONConversation(raw []byte) sessionConversation {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return extractTextConversation(raw)
	}
	conversation := sessionConversation{}
	conversation.addJSONValue(value)
	return conversation
}

func extractMarkdownConversation(raw []byte) sessionConversation {
	lines := strings.Split(string(raw), "\n")
	conversation := sessionConversation{}
	role := ""
	var body strings.Builder
	flush := func() {
		text := strings.TrimSpace(body.String())
		if role != "" && text != "" {
			conversation.addMessage(role, text, "")
		}
		body.Reset()
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(strings.TrimLeft(trimmed, "# "))
		if strings.HasPrefix(trimmed, "#") {
			headingRole := markdownHeadingRole(lower)
			if !isUserFacingRole(headingRole) {
				continue
			}
			flush()
			role = canonicalMessageRole(headingRole)
			conversation.EventCount++
			continue
		}
		if role != "" {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	flush()
	if conversation.EventCount == 0 && strings.TrimSpace(string(raw)) != "" {
		conversation.EventCount = len(lines)
	}
	conversation.OmittedEvents = max(conversation.EventCount-conversation.MessageCount, 0)
	return conversation
}

func markdownHeadingRole(heading string) string {
	heading = strings.TrimSpace(strings.TrimLeft(heading, "# "))
	if heading == "" {
		return ""
	}
	fields := strings.Fields(heading)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "`:[]()")
}

func extractTextConversation(raw []byte) sessionConversation {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return sessionConversation{}
	}
	return sessionConversation{
		EventCount:    len(strings.Split(text, "\n")),
		OmittedEvents: len(strings.Split(text, "\n")),
	}
}

func (conversation *sessionConversation) addJSONValue(value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			conversation.addJSONValue(item)
		}
	case map[string]any:
		if events, ok := firstArray(typed, "events", "messages", "transcript", "conversation", "items"); ok {
			for _, event := range events {
				conversation.addJSONValue(event)
			}
			return
		}
		conversation.EventCount++
		conversation.addEvent(typed)
	default:
		conversation.EventCount++
		conversation.OmittedEvents++
	}
}

func (conversation *sessionConversation) addEvent(event map[string]any) {
	timestamp := eventTimestamp(event)
	if timestamp != "" {
		if conversation.FirstTimestamp == "" {
			conversation.FirstTimestamp = timestamp
		}
		conversation.LastTimestamp = timestamp
	}
	if isToolEvent(event) {
		conversation.ToolCalls++
		conversation.OmittedEvents++
		return
	}
	conversation.ToolCalls += nestedToolCallCount(event)
	role := eventRole(event)
	text := extractText(event)
	if role == "" || text == "" || !isUserFacingRole(role) {
		conversation.OmittedEvents++
		return
	}
	if shouldOmitSessionMessage(role, text) {
		conversation.OmittedEvents++
		return
	}
	conversation.addMessage(canonicalMessageRole(role), text, timestamp)
}

func (conversation *sessionConversation) addMessage(role string, text string, timestamp string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	role = canonicalMessageRole(role)
	if len(conversation.Messages) > 0 {
		last := conversation.Messages[len(conversation.Messages)-1]
		if last.Role == role && last.Text == text {
			return
		}
	}
	conversation.Messages = append(conversation.Messages, sessionMessage{Role: role, Text: text, Timestamp: timestamp})
	conversation.MessageCount++
	switch role {
	case "user":
		conversation.UserMessages++
	case "assistant":
		conversation.AgentMessages++
	}
}

func eventTimestamp(event map[string]any) string {
	if timestamp := firstString(event, "timestamp", "created_at", "createdAt", "time"); timestamp != "" {
		return timestamp
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		if timestamp := firstString(payload, "timestamp", "created_at", "createdAt", "time"); timestamp != "" {
			return timestamp
		}
	}
	return ""
}

func eventRole(event map[string]any) string {
	role := firstString(event, "role", "speaker", "author")
	eventType := firstString(event, "type", "kind", "event")
	if role == "" {
		role = eventType
	}
	if message, ok := event["message"].(map[string]any); ok {
		if messageRole := firstString(message, "role", "speaker", "author"); messageRole != "" {
			role = messageRole
		}
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		payloadType := firstString(payload, "type", "kind", "event")
		if payloadRole := firstString(payload, "role", "speaker", "author"); payloadRole != "" {
			role = payloadRole
		} else if payloadType == "agent_message" {
			role = "assistant"
		} else if payloadType == "user_message" {
			role = "user"
		}
	}
	return strings.ToLower(strings.TrimSpace(role))
}

func isUserFacingRole(role string) bool {
	role = canonicalMessageRole(role)
	return role == "user" || role == "assistant"
}

func shouldOmitSessionMessage(role string, text string) bool {
	role = canonicalMessageRole(role)
	text = strings.TrimSpace(text)
	if role != "user" {
		return false
	}
	return strings.HasPrefix(text, "# AGENTS.md instructions") ||
		strings.Contains(text, "<environment_context>") ||
		strings.Contains(text, "<INSTRUCTIONS>")
}

func canonicalMessageRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(role, "_", " ")))
	switch role {
	case "assistant", "agent", "model", "ai", "assistant message", "agent message":
		return "assistant"
	case "user", "human", "customer", "user message":
		return "user"
	default:
		return role
	}
}

func isToolEvent(event map[string]any) bool {
	values := []string{
		firstString(event, "role", "type", "kind", "event", "name"),
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		values = append(values, firstString(payload, "role", "type", "kind", "event", "name"))
	}
	if message, ok := event["message"].(map[string]any); ok {
		values = append(values, firstString(message, "role", "type", "kind", "event", "name"))
	}
	for _, value := range values {
		value = strings.ToLower(value)
		if strings.Contains(value, "tool") || strings.Contains(value, "function_call") || value == "function call" || value == "function call output" {
			return true
		}
	}
	return false
}

func nestedToolCallCount(value any) int {
	switch typed := value.(type) {
	case []any:
		total := 0
		for _, item := range typed {
			total += nestedToolCallCount(item)
		}
		return total
	case map[string]any:
		if isToolEvent(typed) {
			return 1
		}
		total := 0
		for _, nested := range typed {
			total += nestedToolCallCount(nested)
		}
		return total
	default:
		return 0
	}
}

func sanitizeSession(raw []byte) []byte {
	redacted := string(raw)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`sk-[A-Za-z0-9_-]{16,}`),
		regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{20,}`),
		regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{20,}`),
		regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)["'\s:=]+[A-Za-z0-9_./+\-=]{8,}`),
	}
	for _, pattern := range patterns {
		redacted = pattern.ReplaceAllString(redacted, "$1[REDACTED]")
	}
	return []byte(redacted)
}

func detectRawFormat(source string, raw []byte) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(source)), ".")
	switch ext {
	case "jsonl", "json", "md", "markdown", "txt":
		if ext == "markdown" {
			return "md"
		}
		return ext
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return "json"
	}
	return "jsonl"
}

func resolveSnapshotTarget(evolution *eve.Evolution, repo string) (snapshotTarget, error) {
	if strings.TrimSpace(evolution.Implementation.Snapshot) == "" {
		return snapshotTarget{}, fmt.Errorf("Evolution %s has no resolvable code snapshot.", evolution.Metadata.ID)
	}
	repository := strings.TrimSpace(repo)
	if repository == "" {
		repository = currentRepositoryName()
		if len(evolution.Implementation.Repositories) == 1 {
			for name := range evolution.Implementation.Repositories {
				repository = name
			}
		}
	}
	if len(evolution.Implementation.Repositories) > 0 {
		if _, ok := evolution.Implementation.Repositories[repository]; !ok {
			return snapshotTarget{}, fmt.Errorf("Evolution %s does not reference repository %q.", evolution.Metadata.ID, repository)
		}
	}
	return snapshotTarget{Repository: repository, Commit: evolution.Implementation.Snapshot}, nil
}

func workingTreeDirty() (bool, error) {
	output, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func printSnapshot(w io.Writer, evolution *eve.Evolution, target snapshotTarget) {
	fmt.Fprintln(w, "Snapshot")
	fmt.Fprintf(w, "%s\n", evolution.Metadata.Title)
	if evolution.Outcome != "" {
		fmt.Fprintf(w, "%s\n", evolution.Outcome)
	}
	fmt.Fprintf(w, "Repository: %s\n", target.Repository)
	fmt.Fprintf(w, "Commit: %s\n", target.Commit)
	fmt.Fprintln(w, "Behavior")
	printClaims(w, "+", evolution.Behavior.Added)
	printClaims(w, "~", evolution.Behavior.Changed)
	printClaims(w, "-", evolution.Behavior.Removed)
	printClaims(w, "!", evolution.Behavior.Fixed)
	fmt.Fprintln(w, "Verification")
	for _, verification := range evolution.Verification {
		fmt.Fprintf(w, "- %s: %s\n", verification.Status, verification.Reference)
	}
}

func printEvolution(w io.Writer, store localStore, evolution *eve.Evolution) {
	fmt.Fprintf(w, "%s\n", fallback(evolution.Metadata.Title, evolution.Metadata.ID))
	fmt.Fprintf(w, "ID: %s\n", evolution.Metadata.ID)
	fmt.Fprintf(w, "Status: %s\n", humanStatus(evolution.Metadata.Status))
	if evolution.Metadata.Type != "" {
		fmt.Fprintf(w, "Type: %s\n", evolution.Metadata.Type)
	}
	if evolution.Intent != "" {
		fmt.Fprintf(w, "\nIntent\n%s\n", evolution.Intent)
	}
	if evolution.Outcome != "" {
		fmt.Fprintf(w, "\nOutcome\n%s\n", evolution.Outcome)
	}
	fmt.Fprintln(w, "\nBehavior")
	if behaviorCount(evolution) == 0 {
		fmt.Fprintln(w, "- none")
	} else {
		printClaims(w, "+", evolution.Behavior.Added)
		printClaims(w, "~", evolution.Behavior.Changed)
		printClaims(w, "-", evolution.Behavior.Removed)
		printClaims(w, "!", evolution.Behavior.Fixed)
	}
	fmt.Fprintln(w, "\nSessions")
	manifest := store.loadSessionManifest(evolution.Metadata.ID)
	if len(evolution.Sessions) == 0 && len(manifest.Sessions) == 0 {
		fmt.Fprintln(w, "- none")
	}
	for _, session := range evolution.Sessions {
		label := fallback(session.Provider, "unknown")
		if session.ID != "" {
			label += " " + session.ID
		}
		fmt.Fprintf(w, "- %s\n", label)
		for _, artifact := range manifest.Sessions {
			if artifact.Provider == session.Provider && artifact.ID == session.ID {
				fmt.Fprintf(w, "  transcript: %s\n", artifact.Transcript)
				fmt.Fprintf(w, "  raw: %s\n", artifact.Raw)
				if artifact.Sanitized {
					fmt.Fprintln(w, "  sanitized: true")
				}
			}
		}
	}
	fmt.Fprintln(w, "\nVerification")
	if len(evolution.Verification) == 0 {
		fmt.Fprintln(w, "- none")
	}
	for _, verification := range evolution.Verification {
		label := strings.TrimSpace(verification.Type + " " + verification.Status)
		if verification.Reference != "" {
			label += " (" + verification.Reference + ")"
		}
		fmt.Fprintf(w, "- %s\n", label)
	}
	fmt.Fprintln(w, "\nImplementation")
	fmt.Fprintf(w, "Repositories: %d\n", len(evolution.Implementation.Repositories))
	if len(evolution.Implementation.Repositories) > 0 {
		names := make([]string, 0, len(evolution.Implementation.Repositories))
		for name := range evolution.Implementation.Repositories {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(w, "  - %s %s\n", name, evolution.Implementation.Repositories[name].Status)
		}
	}
	fmt.Fprintln(w, "Commits:")
	if len(evolution.Implementation.Commits) == 0 {
		fmt.Fprintln(w, "  - none")
	}
	for _, commit := range evolution.Implementation.Commits {
		fmt.Fprintf(w, "  - %s\n", commit)
	}
	fmt.Fprintln(w, "Snapshot:")
	if evolution.Implementation.Snapshot == "" {
		fmt.Fprintln(w, "  - none")
	} else {
		fmt.Fprintf(w, "  - %s\n", evolution.Implementation.Snapshot)
	}
}

func printClaims(w io.Writer, marker string, claims []eve.BehaviorClaim) {
	for _, claim := range claims {
		fmt.Fprintf(w, "%s %s\n", marker, claim.Description)
	}
}

func printGraphNode(w io.Writer, byID map[string]*eve.Evolution, children map[string][]string, id string, prefix string, last bool, root bool, seen map[string]bool) {
	connector := ""
	nextPrefix := ""
	if !root {
		if last {
			connector = "`-- "
			nextPrefix = prefix + "    "
		} else {
			connector = "|-- "
			nextPrefix = prefix + "|   "
		}
	}
	if seen[id] {
		fmt.Fprintf(w, "%s%s%s (cycle)\n", prefix, connector, id)
		return
	}
	seen[id] = true
	evolution := byID[id]
	if evolution == nil {
		fmt.Fprintf(w, "%s%s%s\n", prefix, connector, id)
		return
	}
	fmt.Fprintf(w, "%s%s%s %s [%s]\n", prefix, connector, id, evolution.Metadata.Title, humanStatus(evolution.Metadata.Status))
	for i, child := range children[id] {
		printGraphNode(w, byID, children, child, nextPrefix, i == len(children[id])-1, false, seen)
	}
}

func printEvolutionTable(w io.Writer, evolutions []*eve.Evolution) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tSTATUS\tTYPE\tTITLE")
	for _, evolution := range evolutions {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", evolution.Metadata.ID, humanStatus(evolution.Metadata.Status), evolution.Metadata.Type, evolution.Metadata.Title)
	}
	table.Flush()
}

func evolutionMatches(evolution *eve.Evolution, query string) bool {
	for _, field := range searchableFields(evolution) {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func searchableFields(evolution *eve.Evolution) []string {
	fields := []string{evolution.Metadata.ID, evolution.Metadata.Title, evolution.Metadata.Type, evolution.Metadata.Status, evolution.Intent, evolution.Outcome}
	for _, claim := range evolution.Behavior.Added {
		fields = append(fields, claim.Description)
	}
	for _, claim := range evolution.Behavior.Changed {
		fields = append(fields, claim.Description)
	}
	for _, claim := range evolution.Behavior.Removed {
		fields = append(fields, claim.Description)
	}
	for _, claim := range evolution.Behavior.Fixed {
		fields = append(fields, claim.Description)
	}
	for _, session := range evolution.Sessions {
		fields = append(fields, session.Provider, session.ID, session.URI)
	}
	for _, verification := range evolution.Verification {
		fields = append(fields, verification.Type, verification.Status, verification.Reference)
	}
	fields = append(fields, evolution.Implementation.Commits...)
	fields = append(fields, evolution.Implementation.Snapshot)
	return fields
}

func upsertSession(sessions []eve.Session, session eve.Session) []eve.Session {
	for i, existing := range sessions {
		if existing.Provider == session.Provider && existing.ID == session.ID {
			sessions[i] = session
			return sessions
		}
	}
	return append(sessions, session)
}

func upsertSessionExtension(evolution *eve.Evolution, artifact sessionArtifact) error {
	if evolution.Extensions == nil {
		evolution.Extensions = map[string]json.RawMessage{}
	}
	var extension sessionExtension
	if raw, ok := evolution.Extensions["eve.sessions"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &extension); err != nil {
			return fmt.Errorf("parse eve.sessions extension: %w", err)
		}
	}
	replaced := false
	for i, existing := range extension.Sessions {
		if existing.Provider == artifact.Provider && existing.ID == artifact.ID {
			extension.Sessions[i] = artifact
			replaced = true
			break
		}
	}
	if !replaced {
		extension.Sessions = append(extension.Sessions, artifact)
	}
	sortSessionArtifacts(extension.Sessions)
	raw, err := json.Marshal(extension)
	if err != nil {
		return fmt.Errorf("marshal eve.sessions extension: %w", err)
	}
	evolution.Extensions["eve.sessions"] = raw
	return nil
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "claude-code" {
		return "claude"
	}
	return provider
}

func isSupportedSessionProvider(provider string) bool {
	switch provider {
	case "codex", "claude", "opencode", "pi":
		return true
	default:
		return false
	}
}

func normalizeSessionID(provider string, sessionID string, source string) string {
	if sessionID != "" && sessionID != "latest" {
		return sessionID
	}
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	base = strings.TrimPrefix(base, provider+"-")
	if base == "" || base == "." {
		return "latest"
	}
	return base
}

func safeArtifactName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func firstID(value any) string {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if id := firstID(item); id != "" {
				return id
			}
		}
	case map[string]any:
		if id := firstString(typed, "id", "sessionID", "session_id"); id != "" {
			return id
		}
		for _, nested := range typed {
			if id := firstID(nested); id != "" {
				return id
			}
		}
	}
	return ""
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func firstArray(values map[string]any, keys ...string) ([]any, bool) {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if items, ok := value.([]any); ok {
				return items, true
			}
		}
	}
	return nil, false
}

func extractText(event map[string]any) string {
	if text := firstString(event, "text", "message", "content", "summary"); text != "" {
		return text
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		if text := extractText(payload); text != "" {
			return text
		}
	}
	if message, ok := event["message"].(map[string]any); ok {
		if text := extractText(message); text != "" {
			return text
		}
	}
	if value, ok := firstArray(event, "content", "parts"); ok {
		var parts []string
		for _, item := range value {
			switch typed := item.(type) {
			case string:
				parts = append(parts, typed)
			case map[string]any:
				if isToolEvent(typed) {
					continue
				}
				if text := firstString(typed, "text", "content", "message"); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func addTimeline(evolution *eve.Evolution, event string, description string) {
	evolution.Timeline = append(evolution.Timeline, eve.TimelineEntry{
		Timestamp:   nowUTC(),
		Actor:       &eve.Actor{Type: "tool", Provider: "eve-cli"},
		Event:       event,
		Description: description,
	})
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func fallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func humanStatus(status string) string {
	switch status {
	case "active":
		return "In Progress"
	case "draft":
		return "Draft"
	case "completed":
		return "Completed"
	case "archived":
		return "Archived"
	default:
		return titleCase(status)
	}
}

func humanEvent(event string) string {
	event = strings.ReplaceAll(event, "_", " ")
	parts := strings.Fields(event)
	for i, part := range parts {
		parts[i] = titleCase(part)
	}
	return strings.Join(parts, " ")
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, addition := range additions {
		addition = strings.TrimSpace(addition)
		if addition == "" || seen[addition] {
			continue
		}
		values = append(values, addition)
		seen[addition] = true
	}
	return values
}

func uniqueStrings(values []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			continue
		}
		out = append(out, value)
		seen[value] = true
	}
	return out
}

func firstArgOrEmpty(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func currentRepositoryName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "repository"
	}
	name := filepath.Base(wd)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "repository"
	}
	return name
}
