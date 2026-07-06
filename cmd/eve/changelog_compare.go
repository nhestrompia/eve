package main

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nhestrompia/eve"
)

type snapshotRangeOptions struct {
	Since string
	From  string
	To    string
}

type changelogGroup struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

type comparisonResponse struct {
	Repository string               `json:"repository"`
	From       snapshotSummary      `json:"from"`
	To         snapshotSummary      `json:"to"`
	Range      []snapshotSummary    `json:"range"`
	Added      []comparisonChange   `json:"added"`
	Changed    []comparisonChange   `json:"changed"`
	Fixed      []comparisonChange   `json:"fixed"`
	Decisions  []comparisonDecision `json:"decisions"`
	Risks      []comparisonRisk     `json:"risks"`
	Validation []comparisonCheck    `json:"validation"`
	Timeline   []comparisonTimeline `json:"timeline"`
}

type comparisonChange struct {
	SnapshotID    string `json:"snapshotId"`
	SnapshotTitle string `json:"snapshotTitle"`
	Text          string `json:"text"`
	Type          string `json:"type"`
	CreatedAt     string `json:"createdAt"`
}

type comparisonDecision struct {
	SnapshotID    string `json:"snapshotId"`
	SnapshotTitle string `json:"snapshotTitle"`
	Title         string `json:"title"`
	Rationale     string `json:"rationale,omitempty"`
}

type comparisonRisk struct {
	SnapshotID    string `json:"snapshotId"`
	SnapshotTitle string `json:"snapshotTitle"`
	Title         string `json:"title"`
	Severity      string `json:"severity"`
	Mitigation    string `json:"mitigation,omitempty"`
}

type comparisonCheck struct {
	SnapshotID    string `json:"snapshotId"`
	SnapshotTitle string `json:"snapshotTitle"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Output        string `json:"output,omitempty"`
}

type comparisonTimeline struct {
	SnapshotID    string `json:"snapshotId"`
	SnapshotTitle string `json:"snapshotTitle"`
	Phase         string `json:"phase"`
	Title         string `json:"title"`
	Summary       string `json:"summary,omitempty"`
	OccurredAt    string `json:"occurredAt"`
}

func runChangelog(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flagSet("changelog", stderr)
	cwd := fs.String("cwd", "", "repository working directory")
	repoID := fs.String("repo-id", "", "repository id")
	since := fs.String("since", "", "snapshot id or YYYY-MM-DD date to generate changes since")
	from := fs.String("from", "", "starting snapshot id")
	to := fs.String("to", "", "ending snapshot id")
	markdown := fs.Bool("markdown", false, "write Markdown output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve changelog takes no positional arguments")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: *cwd, RepoID: *repoID})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	snapshots, err := selectSnapshotRange(repo, snapshotRangeOptions{Since: *since, From: *from, To: *to})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	printChangelog(stdout, buildChangelogGroups(snapshots), *markdown)
	return 0
}

func runCompare(args []string, stdout io.Writer, stderr io.Writer) int {
	cwd, repoID, markdown, snapshotIDs, err := parseCompareArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if len(snapshotIDs) != 2 {
		fmt.Fprintln(stderr, "eve compare requires from and to snapshot ids")
		return 2
	}
	repo, err := resolveRepo(repoRequest{CWD: cwd, RepoID: repoID})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	comparison, err := compareSnapshots(repo, snapshotIDs[0], snapshotIDs[1])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	printComparison(stdout, comparison, markdown)
	return 0
}

func parseCompareArgs(args []string) (string, string, bool, []string, error) {
	var cwd, repoID string
	markdown := false
	var snapshotIDs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--markdown":
			markdown = true
		case strings.HasPrefix(arg, "--markdown="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "--markdown="))
			if err != nil {
				return "", "", false, nil, fmt.Errorf("--markdown requires a boolean value")
			}
			markdown = value
		case arg == "--cwd":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return "", "", false, nil, fmt.Errorf("--cwd requires a value")
			}
			cwd = args[i]
		case strings.HasPrefix(arg, "--cwd="):
			cwd = strings.TrimPrefix(arg, "--cwd=")
			if strings.TrimSpace(cwd) == "" {
				return "", "", false, nil, fmt.Errorf("--cwd requires a value")
			}
		case arg == "--repo-id":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return "", "", false, nil, fmt.Errorf("--repo-id requires a value")
			}
			repoID = args[i]
		case strings.HasPrefix(arg, "--repo-id="):
			repoID = strings.TrimPrefix(arg, "--repo-id=")
			if strings.TrimSpace(repoID) == "" {
				return "", "", false, nil, fmt.Errorf("--repo-id requires a value")
			}
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", false, nil, fmt.Errorf("unknown flag %s", arg)
			}
			snapshotIDs = append(snapshotIDs, arg)
		}
	}
	return cwd, repoID, markdown, snapshotIDs, nil
}

func flagSet(name string, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	return fs
}

func selectSnapshotRange(repo repository, opts snapshotRangeOptions) ([]*eve.Snapshot, error) {
	opts.Since = strings.TrimSpace(opts.Since)
	opts.From = strings.TrimSpace(opts.From)
	opts.To = strings.TrimSpace(opts.To)
	if opts.Since != "" && (opts.From != "" || opts.To != "") {
		return nil, fmt.Errorf("--since cannot be combined with --from or --to")
	}
	if (opts.From == "") != (opts.To == "") {
		return nil, fmt.Errorf("--from and --to must be provided together")
	}
	snapshots, err := repo.listSnapshots("")
	if err != nil {
		return nil, err
	}
	sortSnapshotsChronological(snapshots)
	if opts.Since == "" && opts.From == "" && opts.To == "" {
		return snapshots, nil
	}
	if opts.Since != "" {
		if sinceDate, ok := parseSnapshotDate(opts.Since); ok {
			var selected []*eve.Snapshot
			for _, snapshot := range snapshots {
				createdAt, err := parseSnapshotCreatedAt(snapshot)
				if err != nil {
					return nil, err
				}
				if !createdAt.Before(sinceDate) {
					selected = append(selected, snapshot)
				}
			}
			return selected, nil
		}
		index, ok := snapshotIndexByID(snapshots, opts.Since)
		if !ok {
			return nil, fmt.Errorf("snapshot %s not found", opts.Since)
		}
		return append([]*eve.Snapshot(nil), snapshots[index+1:]...), nil
	}
	fromIndex, ok := snapshotIndexByID(snapshots, opts.From)
	if !ok {
		return nil, fmt.Errorf("snapshot %s not found", opts.From)
	}
	toIndex, ok := snapshotIndexByID(snapshots, opts.To)
	if !ok {
		return nil, fmt.Errorf("snapshot %s not found", opts.To)
	}
	if fromIndex >= toIndex {
		return nil, fmt.Errorf("snapshot range is reversed: %s must be before %s", opts.From, opts.To)
	}
	return append([]*eve.Snapshot(nil), snapshots[fromIndex+1:toIndex+1]...), nil
}

func compareSnapshots(repo repository, fromID string, toID string) (comparisonResponse, error) {
	fromSnapshot, err := repo.loadSnapshot(fromID)
	if err != nil {
		return comparisonResponse{}, fmt.Errorf("snapshot %s not found", fromID)
	}
	toSnapshot, err := repo.loadSnapshot(toID)
	if err != nil {
		return comparisonResponse{}, fmt.Errorf("snapshot %s not found", toID)
	}
	rangeSnapshots, err := selectSnapshotRange(repo, snapshotRangeOptions{From: fromID, To: toID})
	if err != nil {
		return comparisonResponse{}, err
	}
	comparison := comparisonResponse{
		Repository: repo.ID,
		From:       summarizeSnapshotForRepo(repo, fromSnapshot),
		To:         summarizeSnapshotForRepo(repo, toSnapshot),
		Range:      make([]snapshotSummary, 0, len(rangeSnapshots)),
		Added:      []comparisonChange{},
		Changed:    []comparisonChange{},
		Fixed:      []comparisonChange{},
		Decisions:  []comparisonDecision{},
		Risks:      []comparisonRisk{},
		Validation: []comparisonCheck{},
		Timeline:   []comparisonTimeline{},
	}
	for _, snapshot := range rangeSnapshots {
		comparison.Range = append(comparison.Range, summarizeSnapshotForRepo(repo, snapshot))
		change := comparisonChange{
			SnapshotID:    snapshot.ID,
			SnapshotTitle: snapshot.Title,
			Text:          snapshotChangeText(snapshot),
			Type:          snapshot.Type,
			CreatedAt:     snapshot.CreatedAt,
		}
		switch snapshot.Type {
		case "feature":
			comparison.Added = append(comparison.Added, change)
		case "bugfix":
			comparison.Fixed = append(comparison.Fixed, change)
		default:
			comparison.Changed = append(comparison.Changed, change)
		}
		for _, decision := range snapshot.Decisions {
			comparison.Decisions = append(comparison.Decisions, comparisonDecision{
				SnapshotID:    snapshot.ID,
				SnapshotTitle: snapshot.Title,
				Title:         decision.Title,
				Rationale:     decision.Rationale,
			})
		}
		for _, risk := range snapshot.Risks {
			comparison.Risks = append(comparison.Risks, comparisonRisk{
				SnapshotID:    snapshot.ID,
				SnapshotTitle: snapshot.Title,
				Title:         risk.Title,
				Severity:      risk.Severity,
				Mitigation:    risk.Mitigation,
			})
		}
		for _, validation := range snapshot.Validation {
			comparison.Validation = append(comparison.Validation, comparisonCheck{
				SnapshotID:    snapshot.ID,
				SnapshotTitle: snapshot.Title,
				Command:       validation.Command,
				Status:        validation.Status,
				Output:        validation.Output,
			})
		}
		if len(snapshot.Timeline) == 0 {
			comparison.Timeline = append(comparison.Timeline, comparisonTimeline{
				SnapshotID:    snapshot.ID,
				SnapshotTitle: snapshot.Title,
				Phase:         snapshot.Type,
				Title:         snapshot.Title,
				Summary:       snapshot.Summary,
				OccurredAt:    snapshot.CreatedAt,
			})
			continue
		}
		for _, entry := range snapshot.Timeline {
			comparison.Timeline = append(comparison.Timeline, comparisonTimeline{
				SnapshotID:    snapshot.ID,
				SnapshotTitle: snapshot.Title,
				Phase:         entry.Phase,
				Title:         entry.Title,
				Summary:       entry.Summary,
				OccurredAt:    firstNonEmpty(entry.OccurredAt, snapshot.CreatedAt),
			})
		}
	}
	return comparison, nil
}

func buildChangelogGroups(snapshots []*eve.Snapshot) []changelogGroup {
	order := []string{"Features", "Improvements", "Fixes", "Other"}
	groups := map[string][]string{}
	for _, snapshot := range snapshots {
		group := changelogGroupTitle(snapshot.Type)
		groups[group] = append(groups[group], snapshotChangeText(snapshot))
	}
	var result []changelogGroup
	for _, title := range order {
		if len(groups[title]) > 0 {
			result = append(result, changelogGroup{Title: title, Items: groups[title]})
		}
	}
	return result
}

func changelogGroupTitle(snapshotType string) string {
	switch snapshotType {
	case "feature":
		return "Features"
	case "bugfix":
		return "Fixes"
	case "refactor", "experiment":
		return "Improvements"
	default:
		return "Other"
	}
}

func snapshotChangeText(snapshot *eve.Snapshot) string {
	if text := strings.TrimSpace(snapshot.UserVisibleChange); text != "" {
		return text
	}
	return strings.TrimSpace(snapshot.Title)
}

func printChangelog(w io.Writer, groups []changelogGroup, markdown bool) {
	if len(groups) == 0 {
		fmt.Fprintln(w, "No snapshot changes found.")
		return
	}
	if markdown {
		fmt.Fprintln(w, "# Release Notes")
		for _, group := range groups {
			fmt.Fprintf(w, "## %s\n", group.Title)
			for _, item := range group.Items {
				fmt.Fprintf(w, "- %s\n", item)
			}
		}
		return
	}
	fmt.Fprintln(w, "Release Notes")
	for _, group := range groups {
		fmt.Fprintln(w)
		fmt.Fprintln(w, group.Title)
		for _, item := range group.Items {
			fmt.Fprintf(w, "- %s\n", item)
		}
	}
}

func printComparison(w io.Writer, comparison comparisonResponse, markdown bool) {
	if markdown {
		fmt.Fprintf(w, "# Snapshot Comparison\n\n")
		fmt.Fprintf(w, "**From:** %s  \n**To:** %s\n", comparison.From.ID, comparison.To.ID)
		printComparisonChangeSection(w, "Added", comparison.Added, true)
		printComparisonChangeSection(w, "Changed", comparison.Changed, true)
		printComparisonChangeSection(w, "Fixed", comparison.Fixed, true)
		printComparisonDecisionSection(w, comparison.Decisions, true)
		printComparisonRiskSection(w, comparison.Risks, true)
		printComparisonValidationSection(w, comparison.Validation, true)
		printComparisonTimelineSection(w, comparison.Timeline, true)
		return
	}
	fmt.Fprintln(w, "Snapshot Comparison")
	fmt.Fprintf(w, "From: %s\nTo: %s\n", comparison.From.ID, comparison.To.ID)
	printComparisonChangeSection(w, "Added", comparison.Added, false)
	printComparisonChangeSection(w, "Changed", comparison.Changed, false)
	printComparisonChangeSection(w, "Fixed", comparison.Fixed, false)
	printComparisonDecisionSection(w, comparison.Decisions, false)
	printComparisonRiskSection(w, comparison.Risks, false)
	printComparisonValidationSection(w, comparison.Validation, false)
	printComparisonTimelineSection(w, comparison.Timeline, false)
}

func printComparisonChangeSection(w io.Writer, title string, items []comparisonChange, markdown bool) {
	if len(items) == 0 {
		return
	}
	printSectionTitle(w, title, markdown)
	for _, item := range items {
		fmt.Fprintf(w, "- %s (%s)\n", item.Text, item.SnapshotID)
	}
}

func printComparisonDecisionSection(w io.Writer, items []comparisonDecision, markdown bool) {
	if len(items) == 0 {
		return
	}
	printSectionTitle(w, "Decisions", markdown)
	for _, item := range items {
		if item.Rationale != "" {
			fmt.Fprintf(w, "- %s: %s (%s)\n", item.Title, item.Rationale, item.SnapshotID)
		} else {
			fmt.Fprintf(w, "- %s (%s)\n", item.Title, item.SnapshotID)
		}
	}
}

func printComparisonRiskSection(w io.Writer, items []comparisonRisk, markdown bool) {
	if len(items) == 0 {
		return
	}
	printSectionTitle(w, "Risks", markdown)
	for _, item := range items {
		if item.Mitigation != "" {
			fmt.Fprintf(w, "- %s: %s; mitigation: %s (%s)\n", item.Severity, item.Title, item.Mitigation, item.SnapshotID)
		} else {
			fmt.Fprintf(w, "- %s: %s (%s)\n", item.Severity, item.Title, item.SnapshotID)
		}
	}
}

func printComparisonValidationSection(w io.Writer, items []comparisonCheck, markdown bool) {
	if len(items) == 0 {
		return
	}
	printSectionTitle(w, "Validation", markdown)
	for _, item := range items {
		fmt.Fprintf(w, "- %s: %s (%s)\n", item.Status, item.Command, item.SnapshotID)
	}
}

func printComparisonTimelineSection(w io.Writer, items []comparisonTimeline, markdown bool) {
	if len(items) == 0 {
		return
	}
	printSectionTitle(w, "Timeline", markdown)
	for _, item := range items {
		fmt.Fprintf(w, "- %s: %s (%s)\n", item.SnapshotID, item.Title, item.OccurredAt)
	}
}

func printSectionTitle(w io.Writer, title string, markdown bool) {
	if markdown {
		fmt.Fprintf(w, "\n## %s\n", title)
		return
	}
	fmt.Fprintf(w, "\n%s\n", title)
}

func sortSnapshotsChronological(snapshots []*eve.Snapshot) {
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt == snapshots[j].CreatedAt {
			return snapshots[i].ID < snapshots[j].ID
		}
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})
}

func snapshotIndexByID(snapshots []*eve.Snapshot, id string) (int, bool) {
	for index, snapshot := range snapshots {
		if snapshot.ID == id {
			return index, true
		}
	}
	return -1, false
}

func parseSnapshotDate(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func parseSnapshotCreatedAt(snapshot *eve.Snapshot) (time.Time, error) {
	createdAt, err := time.Parse(time.RFC3339, snapshot.CreatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("snapshot %s has invalid createdAt %q", snapshot.ID, snapshot.CreatedAt)
	}
	return createdAt.UTC(), nil
}
