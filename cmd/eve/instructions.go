package main

import (
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
)

const currentInstructionVersion = 2

const canonicalInstructionTemplateV1 = `<!-- eve:instructions:start version="1" -->

## EVE Product History

This repository uses EVE to record completed product work.

When you complete a coherent unit of product work, call the EVE
` + "`complete_snapshot`" + ` tool before ending the task.

Create a Snapshot for work such as:

- A feature or user-visible improvement
- A bug fix
- A meaningful refactor
- A migration
- An experiment
- A release-related change

Do not create a Snapshot for trivial work such as:

- Formatting-only changes
- A variable rename with no behavior change
- Lint-only fixes
- Temporary debugging changes
- Work that was started but not completed

When no Snapshot is warranted, call ` + "`skip_snapshot`" + ` and include a short reason.

The Snapshot should reflect the completed task and include the relevant
behavior changes, validation, commits, screenshots, decisions, risks,
relationships, and session references when available.

<!-- eve:instructions:end -->`

const canonicalInstructionTemplateV2 = `<!-- eve:instructions:start version="2" -->

## EVE Product History

This repository uses EVE to approve implementation plans and record completed
product work.

Before making non-trivial, Snapshot-worthy code changes:

1. Call the EVE ` + "`declare_plan`" + ` tool with a caller-stable
   ` + "`planRequestId`" + `, goal, acceptance criteria, allowed path globs,
   milestones, and verification suite when applicable.
2. Do not modify code until EVE returns a locked Plan ID and revision.
3. If the call times out, is cancelled, or the agent restarts, recover with
   ` + "`get_plan_request`" + ` or call ` + "`declare_plan`" + ` again using
   the same request ID. Never replace a pending request with a new ID just to
   avoid waiting.
4. Pass the locked Plan ID and revision to ` + "`complete_snapshot`" + ` after
   implementation and verification.

When you complete a coherent unit of product work, call the EVE
` + "`complete_snapshot`" + ` tool before ending the task.

Create a Snapshot for work such as:

- A feature or user-visible improvement
- A bug fix
- A meaningful refactor
- A migration
- An experiment
- A release-related change

Do not create a Snapshot for trivial work such as:

- Formatting-only changes
- A variable rename with no behavior change
- Lint-only fixes
- Temporary debugging changes
- Work that was started but not completed

When no Snapshot is warranted, call ` + "`skip_snapshot`" + ` and include a short reason.

The Snapshot should reflect the completed task and include the relevant
behavior changes, validation, commits, screenshots, decisions, risks,
relationships, and session references when available.

<!-- eve:instructions:end -->`

var instructionTemplates = map[int]string{
	1: canonicalInstructionTemplateV1,
	2: canonicalInstructionTemplateV2,
}

type instructionState string

const (
	instructionMissing   instructionState = "missing"
	instructionCurrent   instructionState = "current"
	instructionStale     instructionState = "stale"
	instructionModified  instructionState = "modified"
	instructionMalformed instructionState = "malformed"
)

type instructionTarget struct {
	Name     string
	Filename string
}

var instructionTargets = []instructionTarget{
	{Name: "agents", Filename: "AGENTS.md"},
	{Name: "claude", Filename: "CLAUDE.md"},
}

type instructionInspection struct {
	Target     instructionTarget
	Path       string
	Exists     bool
	State      instructionState
	Version    int
	Data       []byte
	Block      []byte
	BlockStart int
	BlockEnd   int
	Mode       os.FileMode
}

type instructionInstallResult struct {
	Inspection instructionInspection
	Action     string
	Tracked    bool
	Err        error
}

var instructionStartRE = regexp.MustCompile(`(?m)^<!-- eve:instructions:start version="([0-9]+)" -->[ \t]*\r?$`)
var instructionEndRE = regexp.MustCompile(`(?m)^<!-- eve:instructions:end -->[ \t]*\r?$`)

func runInstructions(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "eve instructions requires install, status, or diff")
		return 2
	}
	switch args[0] {
	case "install":
		return runInstructionsInstall(args[1:], stdout, stderr)
	case "status":
		return runInstructionsStatus(args[1:], stdout, stderr)
	case "diff":
		return runInstructionsDiff(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown instructions command %q\n", args[0])
		return 2
	}
}

func runInstructionsInstall(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("instructions install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetValue := fs.String("target", "", "instruction target: agents or claude")
	force := fs.Bool("force", false, "replace a modified managed block")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve instructions install takes no positional arguments")
		return 2
	}
	targets, err := selectInstructionTargets(*targetValue)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	repo, err := resolveRepo(repoRequest{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	results, failed := installInstructionTargets(repo, targets, *force, false)
	printInstructionInstallResults(stdout, stderr, results)
	if failed {
		return 1
	}
	return 0
}

func runInstructionsStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("instructions status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve instructions status takes no arguments")
		return 2
	}
	repo, err := resolveRepo(repoRequest{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Agent instructions")
	allCurrent := true
	for _, target := range instructionTargets {
		inspection, err := inspectInstructionTarget(repo, target)
		if err != nil {
			allCurrent = false
			fmt.Fprintf(stdout, "\n%s\n  ✗ %v\n", target.Filename, err)
			continue
		}
		fmt.Fprintf(stdout, "\n%s\n", target.Filename)
		printInstructionStatus(stdout, inspection)
		if inspection.State != instructionCurrent {
			allCurrent = false
		}
	}
	if !allCurrent {
		return 1
	}
	return 0
}

func runInstructionsDiff(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("instructions diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve instructions diff takes no arguments")
		return 2
	}
	repo, err := resolveRepo(repoRequest{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	failed := false
	for i, target := range instructionTargets {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		inspection, err := inspectInstructionTarget(repo, target)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", target.Filename, err)
			failed = true
			continue
		}
		fmt.Fprintln(stdout, target.Filename)
		if inspection.State == instructionMalformed {
			fmt.Fprintln(stderr, "  contains malformed EVE instruction markers")
			failed = true
			continue
		}
		if inspection.State == instructionCurrent {
			fmt.Fprintln(stdout, "  No differences.")
			continue
		}
		fmt.Fprintf(stdout, "--- installed/%s\n+++ canonical/%s\n", target.Filename, target.Filename)
		current := string(inspection.Block)
		if inspection.State == instructionMissing {
			current = ""
		}
		printLineDiff(stdout, current, instructionTemplates[currentInstructionVersion])
	}
	if failed {
		return 1
	}
	return 0
}

func selectInstructionTargets(value string) ([]instructionTarget, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return append([]instructionTarget(nil), instructionTargets...), nil
	}
	for _, target := range instructionTargets {
		if target.Name == value {
			return []instructionTarget{target}, nil
		}
	}
	return nil, fmt.Errorf("unsupported instruction target %q; supported targets: agents, claude", value)
}

func inspectInstructionTarget(repo repository, target instructionTarget) (instructionInspection, error) {
	path := filepath.Join(repo.Root, target.Filename)
	inspection := instructionInspection{Target: target, Path: path, State: instructionMissing}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return inspection, nil
	}
	if err != nil {
		return inspection, fmt.Errorf("inspect %s: %w", target.Filename, err)
	}
	if !info.Mode().IsRegular() {
		return inspection, fmt.Errorf("%s is not a regular file", target.Filename)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return inspection, fmt.Errorf("read %s: %w", target.Filename, err)
	}
	inspection, err = classifyInstructionData(target, path, data, info.Mode(), instructionTemplates, currentInstructionVersion)
	if err != nil {
		return instructionInspection{}, err
	}
	inspection.Exists = true
	return inspection, nil
}

func classifyInstructionData(target instructionTarget, path string, data []byte, mode os.FileMode, templates map[int]string, currentVersion int) (instructionInspection, error) {
	inspection := instructionInspection{Target: target, Path: path, Exists: true, State: instructionMissing, Data: data, Mode: mode}
	text := string(data)
	startPrefixCount := strings.Count(text, "<!-- eve:instructions:start")
	endPrefixCount := strings.Count(text, "<!-- eve:instructions:end")
	if startPrefixCount == 0 && endPrefixCount == 0 {
		return inspection, nil
	}
	if startPrefixCount != 1 || endPrefixCount != 1 {
		inspection.State = instructionMalformed
		return inspection, nil
	}
	start := instructionStartRE.FindStringSubmatchIndex(text)
	end := instructionEndRE.FindStringIndex(text)
	if start == nil || end == nil || start[0] >= end[0] {
		inspection.State = instructionMalformed
		return inspection, nil
	}
	version, err := strconv.Atoi(text[start[2]:start[3]])
	if err != nil {
		inspection.State = instructionMalformed
		return inspection, nil
	}
	inspection.Version = version
	inspection.BlockStart = start[0]
	inspection.BlockEnd = end[1]
	if inspection.BlockEnd > inspection.BlockStart && inspection.BlockEnd < len(data) && data[inspection.BlockEnd-1] == '\r' && data[inspection.BlockEnd] == '\n' {
		inspection.BlockEnd--
	}
	inspection.Block = append([]byte(nil), data[inspection.BlockStart:inspection.BlockEnd]...)
	expected, known := templates[version]
	if known && normalizeInstructionText(string(inspection.Block)) == normalizeInstructionText(expected) {
		if version == currentVersion {
			inspection.State = instructionCurrent
		} else if version < currentVersion {
			inspection.State = instructionStale
		} else {
			inspection.State = instructionModified
		}
		return inspection, nil
	}
	inspection.State = instructionModified
	return inspection, nil
}

func normalizeInstructionText(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func installInstructionTargets(repo repository, targets []instructionTarget, force bool, initMode bool) ([]instructionInstallResult, bool) {
	results := make([]instructionInstallResult, 0, len(targets))
	failed := false
	for _, target := range targets {
		result := installInstructionTarget(repo, target, force)
		if result.Err != nil {
			if initMode && result.Inspection.State == instructionModified {
				result.Action = "skipped"
				result.Err = nil
			} else {
				failed = true
			}
		}
		results = append(results, result)
	}
	return results, failed
}

func installInstructionTarget(repo repository, target instructionTarget, force bool) instructionInstallResult {
	inspection, err := inspectInstructionTarget(repo, target)
	result := instructionInstallResult{Inspection: inspection}
	if err != nil {
		result.Err = err
		return result
	}
	var updated []byte
	switch inspection.State {
	case instructionCurrent:
		result.Action = "current"
		return result
	case instructionMissing:
		updated = appendInstructionBlock(inspection.Data, instructionTemplateForData(inspection.Data))
		if inspection.Exists {
			result.Action = "updated"
		} else {
			result.Action = "created"
		}
	case instructionStale:
		updated = replaceInstructionBlock(inspection, instructionTemplateForData(inspection.Data))
		result.Action = "updated"
	case instructionModified:
		if !force {
			result.Err = fmt.Errorf("%s contains a modified EVE instruction block; run `eve instructions diff` or reinstall with --force", target.Filename)
			return result
		}
		updated = replaceInstructionBlock(inspection, instructionTemplateForData(inspection.Data))
		result.Action = "updated"
	case instructionMalformed:
		result.Err = fmt.Errorf("%s contains malformed EVE instruction markers; resolve them manually before reinstalling", target.Filename)
		return result
	}
	tracked := isTrackedFile(repo.Root, target.Filename)
	if err := writeInstructionFile(inspection, updated); err != nil {
		result.Err = fmt.Errorf("update %s: %w", target.Filename, err)
		return result
	}
	result.Tracked = tracked
	return result
}

func instructionTemplateForData(data []byte) string {
	template := instructionTemplates[currentInstructionVersion]
	if strings.Contains(string(data), "\r\n") {
		return strings.ReplaceAll(template, "\n", "\r\n")
	}
	return template
}

func appendInstructionBlock(data []byte, block string) []byte {
	newline := "\n"
	if strings.Contains(block, "\r\n") {
		newline = "\r\n"
	}
	if len(data) == 0 {
		return []byte(block + newline)
	}
	updated := append([]byte(nil), data...)
	if !strings.HasSuffix(string(updated), newline) {
		updated = append(updated, []byte(newline)...)
	}
	if !strings.HasSuffix(string(updated), newline+newline) {
		updated = append(updated, []byte(newline)...)
	}
	updated = append(updated, []byte(block)...)
	return append(updated, []byte(newline)...)
}

func replaceInstructionBlock(inspection instructionInspection, block string) []byte {
	updated := make([]byte, 0, len(inspection.Data)-len(inspection.Block)+len(block))
	updated = append(updated, inspection.Data[:inspection.BlockStart]...)
	updated = append(updated, []byte(block)...)
	updated = append(updated, inspection.Data[inspection.BlockEnd:]...)
	return updated
}

func writeInstructionFile(inspection instructionInspection, data []byte) error {
	dir := filepath.Dir(inspection.Path)
	temp, err := os.CreateTemp(dir, ".eve-instructions-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	mode := os.FileMode(0o644)
	if inspection.Exists {
		mode = inspection.Mode.Perm()
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, inspection.Path)
}

func isTrackedFile(root string, filename string) bool {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", "--", filename)
	cmd.Dir = root
	return cmd.Run() == nil
}

func printInstructionInstallResults(stdout io.Writer, stderr io.Writer, results []instructionInstallResult) {
	fmt.Fprintln(stdout, "Agent instructions")
	var tracked []string
	for _, result := range results {
		filename := result.Inspection.Target.Filename
		if result.Err != nil {
			fmt.Fprintf(stderr, "  ✗ %v\n", result.Err)
			continue
		}
		switch result.Action {
		case "created":
			fmt.Fprintf(stdout, "  ✓ %s created\n", filename)
		case "updated":
			fmt.Fprintf(stdout, "  ✓ %s updated\n", filename)
		case "current":
			fmt.Fprintf(stdout, "  ✓ %s is current\n", filename)
		case "skipped":
			fmt.Fprintf(stdout, "  ⚠ %s contains a modified EVE instruction block; preserved existing content\n", filename)
			fmt.Fprintln(stdout, "    Run `eve instructions diff` to inspect it or `eve instructions install --force` to replace it.")
		}
		if result.Tracked {
			tracked = append(tracked, filename)
		}
	}
	if len(tracked) > 0 {
		sort.Strings(tracked)
		fmt.Fprintln(stdout, "\nModified tracked files:")
		for _, filename := range tracked {
			fmt.Fprintf(stdout, "  %s\n", filename)
		}
		fmt.Fprintln(stdout, "\nReview these changes before committing.")
	}
}

func printInstructionStatus(w io.Writer, inspection instructionInspection) {
	switch inspection.State {
	case instructionCurrent:
		fmt.Fprintln(w, "  ✓ Installed")
		fmt.Fprintf(w, "  ✓ Version %d\n", inspection.Version)
		fmt.Fprintln(w, "  ✓ Unmodified")
	case instructionMissing:
		fmt.Fprintln(w, "  ⚠ Missing")
		fmt.Fprintf(w, "  Run `eve instructions install --target %s`.\n", inspection.Target.Name)
	case instructionStale:
		fmt.Fprintf(w, "  ⚠ Version %d is stale; current version is %d\n", inspection.Version, currentInstructionVersion)
		fmt.Fprintln(w, "  Run `eve instructions install` to update it.")
	case instructionModified:
		if inspection.Version > 0 {
			fmt.Fprintf(w, "  ⚠ Version %d is modified or unsupported\n", inspection.Version)
		} else {
			fmt.Fprintln(w, "  ⚠ Modified")
		}
		fmt.Fprintln(w, "  Run `eve instructions diff` to inspect changes.")
		fmt.Fprintln(w, "  Run `eve instructions install --force` to replace it.")
	case instructionMalformed:
		fmt.Fprintln(w, "  ✗ Malformed EVE instruction markers")
		fmt.Fprintln(w, "  Resolve the markers manually, then run `eve instructions status`.")
	}
}

func printLineDiff(w io.Writer, current string, canonical string) {
	currentLines := splitDiffLines(normalizeInstructionText(current))
	canonicalLines := splitDiffLines(normalizeInstructionText(canonical))
	common := 0
	for common < len(currentLines) && common < len(canonicalLines) && currentLines[common] == canonicalLines[common] {
		fmt.Fprintf(w, " %s\n", currentLines[common])
		common++
	}
	for _, line := range currentLines[common:] {
		fmt.Fprintf(w, "-%s\n", line)
	}
	for _, line := range canonicalLines[common:] {
		fmt.Fprintf(w, "+%s\n", line)
	}
}

func splitDiffLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(value, "\n"), "\n")
}
