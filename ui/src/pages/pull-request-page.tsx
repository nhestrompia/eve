import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "@tanstack/react-router";
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronRight,
  Circle,
  CircleDashed,
  Code2,
  ExternalLink,
  FileCode2,
  GitBranch,
  GitPullRequest,
  MessageSquareText,
  Paperclip,
  ShieldCheck,
  Sparkles,
  XCircle,
} from "lucide-react";
import { useMemo, useState } from "react";
import { api } from "../api";
import { relativeTime } from "../components/dashboard-chrome";
import { EmptyState } from "../components/empty-state";
import { ErrorState } from "../components/error-state";
import { EvolutionShell } from "../components/evolution-shell";
import { LoadingState } from "../components/loading-state";
import { PullRequestBranchRoute } from "../components/pull-request-branch-route";
import { SnapshotCodeBrowser } from "../components/snapshot-code-browser";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { humanDate, shortCommit, statusLabel } from "../format";
import type {
  Behavior,
  DetailResponse,
  PlanRevision,
  PullRequestSummary,
} from "../types";

type ReviewTab = "evidence" | "code";

export function PullRequestPage() {
  const { repo, number: rawNumber } = useParams({
    from: "/repositories/$repo/pull-requests/$number",
  });
  const number = Number(rawNumber);
  const evolutions = useQuery({
    queryKey: ["snapshots"],
    queryFn: () => api.snapshots(),
  });
  const pullRequest = useQuery({
    queryKey: ["pull-request", repo, number],
    queryFn: () => api.pullRequest(repo, number),
    enabled: Number.isInteger(number) && number > 0,
    retry: false,
  });
  const detail = useQuery({
    queryKey: ["snapshot-detail", pullRequest.data?.snapshotId, repo],
    queryFn: () =>
      api.snapshotDetail(pullRequest.data?.snapshotId ?? "", repo),
    enabled: Boolean(pullRequest.data?.snapshotId),
    retry: false,
  });

  return (
    <EvolutionShell
      evolutions={evolutions.data ?? []}
      selectedId={pullRequest.data?.snapshotId}
      showHistoryRail={false}
      contentClassName="p-0 sm:p-0 lg:p-0"
    >
      {pullRequest.isLoading ? (
        <LoadingState label={`Loading pull request #${rawNumber}`} />
      ) : null}
      {pullRequest.error ? <ErrorState error={pullRequest.error} /> : null}
      {pullRequest.data ? (
        <PullRequestReview
          repository={repo}
          pullRequest={pullRequest.data}
          detail={detail.data}
          detailLoading={detail.isLoading}
        />
      ) : null}
      {!pullRequest.isLoading &&
      !pullRequest.error &&
      !pullRequest.data ? (
        <EmptyState
          title="Pull request not found"
          detail={`PR #${rawNumber} is not available for ${repo}.`}
        />
      ) : null}
    </EvolutionShell>
  );
}

function PullRequestReview({
  repository,
  pullRequest,
  detail,
  detailLoading,
}: {
  repository: string;
  pullRequest: PullRequestSummary;
  detail?: DetailResponse;
  detailLoading: boolean;
}) {
  const [activeTab, setActiveTab] = useState<ReviewTab>("evidence");
  const lockedRevision = useMemo(
    () => findLockedRevision(detail),
    [detail],
  );

  return (
    <main className="min-h-[calc(100dvh-76px)] bg-slate-50">
      <header className="border-b bg-white px-4 pt-5 sm:px-6 lg:px-9">
        <nav
          aria-label="Breadcrumb"
          className="flex min-w-0 flex-wrap items-center gap-1.5 text-sm text-muted-foreground"
        >
          <Link
            to="/repositories/$repo"
            params={{ repo: repository }}
            className="inline-flex items-center gap-2 font-medium hover:text-blue-700"
          >
            <ArrowLeft className="size-3.5" />
            {repository}
          </Link>
          <ChevronRight className="size-3.5" />
          <PullRequestsBreadcrumbLink repository={repository} />
          <ChevronRight className="size-3.5" />
          <span aria-current="page">#{pullRequest.number}</span>
          {pullRequest.snapshotId ? (
            <>
              <ChevronRight className="size-3.5" />
              <Link
                to="/snapshots/$id"
                params={{ id: pullRequest.snapshotId }}
                className="inline-flex min-w-0 items-center gap-1.5 font-medium text-blue-700 hover:text-blue-800"
              >
                <BookOpen className="size-3.5 shrink-0" />
                <span className="max-w-[34ch] truncate">
                  {pullRequest.snapshotTitle || pullRequest.snapshotId}
                </span>
              </Link>
            </>
          ) : null}
        </nav>

        <div className="flex flex-col gap-5 py-7 lg:flex-row lg:items-end lg:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <ReviewStateBadge pullRequest={pullRequest} />
              <span className="text-sm text-muted-foreground">
                PR #{pullRequest.number}
              </span>
            </div>
            <h1 className="mt-3 max-w-[28ch] text-[2rem] font-semibold leading-[1.1] tracking-[-0.015em] text-balance text-slate-950">
              {pullRequest.title}
            </h1>
            <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">
                {pullRequest.author || "Unknown author"}
              </span>
              <span>opened {relativeTime(pullRequest.createdAt)}</span>
              <PullRequestBranchRoute
                headBranch={pullRequest.headBranch}
                baseBranch={pullRequest.baseBranch}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {pullRequest.snapshotId ? (
              <Button asChild variant="outline">
                <Link
                  to="/snapshots/$id"
                  params={{ id: pullRequest.snapshotId }}
                >
                  <BookOpen className="size-4" />
                  View Snapshot
                </Link>
              </Button>
            ) : null}
            <Button asChild>
              <a href={pullRequest.url} target="_blank" rel="noreferrer">
                Open on GitHub
                <ExternalLink className="size-4" />
              </a>
            </Button>
          </div>
        </div>

        <div
          className="flex gap-7 overflow-x-auto text-sm font-medium text-muted-foreground"
          role="tablist"
          aria-label="Pull request review sections"
        >
          {(["evidence", "code"] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              role="tab"
              aria-selected={activeTab === tab}
              onClick={() => setActiveTab(tab)}
              className={`relative inline-flex min-h-12 shrink-0 items-center gap-2 px-2 capitalize after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:rounded-full ${
                activeTab === tab
                  ? "text-blue-700 after:bg-blue-600"
                  : "hover:text-slate-950 after:bg-transparent"
              }`}
            >
              {tab === "evidence" ? (
                <ShieldCheck className="size-4" />
              ) : (
                <Code2 className="size-4" />
              )}
              {tab}
              {tab === "code" ? (
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">
                  {pullRequest.changedFiles}
                </span>
              ) : null}
            </button>
          ))}
        </div>
      </header>

      <div className="px-4 py-6 sm:px-6 lg:px-9 lg:py-8">
        {activeTab === "evidence" ? (
          <EvidenceReview
            pullRequest={pullRequest}
            detail={detail}
            detailLoading={detailLoading}
            lockedRevision={lockedRevision}
          />
        ) : pullRequest.snapshotId ? (
          <SnapshotCodeBrowser
            snapshotId={pullRequest.snapshotId}
            repository={repository}
          />
        ) : (
          <UnlinkedCodeState />
        )}
      </div>
    </main>
  );
}

function EvidenceReview({
  pullRequest,
  detail,
  detailLoading,
  lockedRevision,
}: {
  pullRequest: PullRequestSummary;
  detail?: DetailResponse;
  detailLoading: boolean;
  lockedRevision?: PlanRevision;
}) {
  if (detailLoading) {
    return <LoadingState label="Loading Snapshot evidence" />;
  }

  return (
    <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
      <div className="min-w-0 space-y-6">
        <MergeImpactSection
          pullRequest={pullRequest}
          detail={detail}
          lockedRevision={lockedRevision}
        />
        <ReviewSignals pullRequest={pullRequest} />
        {detail ? (
          <>
            <AcceptanceCriteria
              revision={lockedRevision}
              matched={pullRequest.planAligned}
            />
            <ImplementationEvidence
              pullRequest={pullRequest}
              detail={detail}
            />
          </>
        ) : (
          <UnlinkedEvidenceState pullRequest={pullRequest} />
        )}
      </div>
      <aside className="space-y-5 xl:sticky xl:top-6">
        <ReviewDecisionCard pullRequest={pullRequest} />
        <PullRequestFacts pullRequest={pullRequest} />
        {detail ? <SnapshotContext detail={detail} /> : null}
      </aside>
    </div>
  );
}

function MergeImpactSection({
  pullRequest,
  detail,
  lockedRevision,
}: {
  pullRequest: PullRequestSummary;
  detail?: DetailResponse;
  lockedRevision?: PlanRevision;
}) {
  const claims = detail ? behaviorClaims(detail.evolution.behavior) : [];
  const primaryChange =
    detail?.snapshot.userVisibleChange ||
    detail?.summary.outcome ||
    lockedRevision?.goal ||
    pullRequest.body;

  return (
    <section className="overflow-hidden rounded-lg border border-blue-200 bg-white shadow-[0_14px_38px_-34px_rgba(30,64,175,0.7)]">
      <div className="bg-blue-50/70 px-5 py-5 sm:px-7 sm:py-6">
        <div className="flex items-center gap-2 text-sm font-semibold text-blue-800">
          <Sparkles className="size-4" />
          If this pull request merges
        </div>
        <h2 className="mt-4 max-w-[24ch] text-2xl font-semibold leading-tight text-balance text-slate-950">
          {primaryChange || "The linked code will land without verified product context."}
        </h2>
        {detail?.summary.outcome &&
        detail.summary.outcome !== primaryChange ? (
          <p className="mt-3 max-w-[72ch] text-sm leading-6 text-slate-700">
            {detail.summary.outcome}
          </p>
        ) : null}
      </div>
      <div className="grid border-t sm:grid-cols-3">
        <ImpactMetric
          label="Product change"
          value={detail?.summary.type || "Not recorded"}
          icon={BookOpen}
        />
        <ImpactMetric
          label="Implementation"
          value={`${pullRequest.changedFiles} files${
            pullRequest.commitCount > 0
              ? ` · ${pullRequest.commitCount} commits`
              : ""
          }`}
          icon={FileCode2}
        />
        <ImpactMetric
          label="Target"
          value={pullRequest.baseBranch}
          icon={GitBranch}
        />
      </div>
      {claims.length ? (
        <div className="border-t px-5 py-5 sm:px-7">
          <h3 className="text-sm font-semibold">Observable changes</h3>
          <ul className="mt-3 grid gap-2">
            {claims.slice(0, 5).map((claim) => (
              <li
                key={`${claim.kind}-${claim.description}`}
                className="flex gap-3 text-sm leading-6 text-slate-700"
              >
                <Check className="mt-1 size-4 shrink-0 text-emerald-600" />
                <span>
                  <strong className="capitalize text-slate-950">
                    {claim.kind}:
                  </strong>{" "}
                  {claim.description}
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}

export function pullRequestsBreadcrumbTarget(repository: string) {
  return {
    to: "/repositories/$repo" as const,
    params: { repo: repository },
    hash: "pull-requests",
  };
}

export function PullRequestsBreadcrumbLink({
  repository,
}: {
  repository: string;
}) {
  return (
    <Link
      {...pullRequestsBreadcrumbTarget(repository)}
      className="font-medium hover:text-blue-700"
    >
      Pull requests
    </Link>
  );
}

function ImpactMetric({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: string;
  icon: typeof BookOpen;
}) {
  return (
    <div className="flex gap-3 border-b px-5 py-4 last:border-b-0 sm:border-b-0 sm:border-r sm:last:border-r-0">
      <Icon className="mt-0.5 size-4 shrink-0 text-slate-500" />
      <div className="min-w-0">
        <p className="text-xs font-medium text-muted-foreground">{label}</p>
        <p className="mt-1 truncate text-sm font-semibold capitalize">{value}</p>
      </div>
    </div>
  );
}

function ReviewSignals({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}) {
  const exactHeadSnapshot =
    Boolean(pullRequest.snapshotId) && pullRequest.snapshotHeadMatch;
  const signals = [
    {
      label: "Plan alignment",
      value: !exactHeadSnapshot
        ? "Not evaluated"
        : !pullRequest.planValid
          ? "Revision invalid"
        : pullRequest.planAligned
          ? "Matched"
          : "Needs evidence",
      tone: !exactHeadSnapshot
        ? "neutral"
        : pullRequest.planAligned && pullRequest.planValid
          ? "success"
          : "warning",
      detail: pullRequest.planRevision
        ? exactHeadSnapshot
          ? `Locked revision ${pullRequest.planRevision}`
          : "Requires current product evidence"
        : "No linked locked revision",
    },
    {
      label: "eve checks",
      value: !exactHeadSnapshot
        ? "Not evaluated"
        : pullRequest.eveChecksPassed
          ? "Passed"
          : "Incomplete",
      tone: !exactHeadSnapshot
        ? "neutral"
        : pullRequest.eveChecksPassed
          ? "success"
          : "warning",
      detail: exactHeadSnapshot
        ? "Required repository verification"
        : "Requires current product evidence",
    },
    {
      label: "Scope",
      value: !exactHeadSnapshot
        ? "Not evaluated"
        : !pullRequest.planValid
          ? "Plan invalid"
        : pullRequest.scopeDrift
          ? "Drift detected"
          : !pullRequest.planAligned
            ? "Not confirmed"
          : "Within plan",
      tone: !exactHeadSnapshot
        ? "neutral"
        : !pullRequest.planValid ||
            pullRequest.scopeDrift ||
            !pullRequest.planAligned
          ? "warning"
          : "success",
      detail: !exactHeadSnapshot
        ? "Requires current product evidence"
        : pullRequest.planValid && pullRequest.planAligned
          ? "Compared with the declared path scope"
          : "Plan conformance evidence is incomplete",
    },
    {
      label: "GitHub",
      value:
        pullRequest.mergeability === "conflicting"
          ? "Merge conflict"
          : pullRequest.githubReady
            ? "Permits merge"
            : "Blocked",
      tone: pullRequest.githubReady ? "success" : "warning",
      detail:
        pullRequest.mergeability === "conflicting"
          ? `Resolve conflicts with ${pullRequest.baseBranch}`
          : `${pullRequest.checksPassed}/${pullRequest.checksTotal} checks passed`,
    },
  ];

  return (
    <section
      aria-label="Review readiness"
      className="grid overflow-hidden rounded-lg border bg-white sm:grid-cols-2 xl:grid-cols-4"
    >
      {signals.map((signal) => (
        <div
          key={signal.label}
          className="border-b px-5 py-5 last:border-b-0 sm:border-r sm:[&:nth-child(2)]:border-r-0 xl:border-b-0 xl:[&:nth-child(2)]:border-r"
        >
          <div className="flex items-center gap-2">
            {signal.tone === "success" ? (
              <CheckCircle2 className="size-4 text-emerald-600" />
            ) : signal.tone === "warning" ? (
              <AlertTriangle className="size-4 text-amber-600" />
            ) : (
              <CircleDashed className="size-4 text-slate-500" />
            )}
            <p className="text-xs font-medium text-muted-foreground">
              {signal.label}
            </p>
          </div>
          <p className="mt-2 text-sm font-semibold">{signal.value}</p>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            {signal.detail}
          </p>
        </div>
      ))}
    </section>
  );
}

function AcceptanceCriteria({
  revision,
  matched,
}: {
  revision?: PlanRevision;
  matched: boolean;
}) {
  const criteria = parseAcceptanceCriteria(revision?.acceptanceCriteria);
  return (
    <section className="rounded-lg border bg-white p-5 sm:p-7">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium text-muted-foreground">
            Locked plan
          </p>
          <h2 className="mt-1 text-lg font-semibold">Acceptance criteria</h2>
        </div>
        {revision ? (
          <Badge variant={matched ? "success" : "warning"}>
            Revision {revision.revision}
          </Badge>
        ) : null}
      </div>
      {revision?.goal ? (
        <p className="mt-5 max-w-[72ch] text-sm leading-6 text-slate-700">
          <strong className="text-slate-950">Goal:</strong> {revision.goal}
        </p>
      ) : null}
      {criteria.length ? (
        <ul className="mt-5 divide-y rounded-lg border">
          {criteria.map((criterion) => (
            <li key={criterion} className="flex gap-3 px-4 py-3.5 text-sm">
              {matched ? (
                <Check className="mt-0.5 size-4 shrink-0 text-emerald-600" />
              ) : (
                <Circle className="mt-0.5 size-4 shrink-0 text-slate-400" />
              )}
              <span className="leading-5">{criterion}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-4 text-sm text-muted-foreground">
          No structured acceptance criteria are attached to this Snapshot.
        </p>
      )}
    </section>
  );
}

function ImplementationEvidence({
  pullRequest,
  detail,
}: {
  pullRequest: PullRequestSummary;
  detail: DetailResponse;
}) {
  const decisions = records(detail.snapshot.decisions);
  const risks = records(detail.snapshot.risks);
  const validations = detail.snapshot.validation;
  const artifacts = detail.snapshot.artifacts;

  return (
    <section className="rounded-lg border bg-white p-5 sm:p-7">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs font-medium text-muted-foreground">
            Snapshot evidence
          </p>
          <h2 className="mt-1 text-lg font-semibold">Implementation record</h2>
        </div>
        <span className="font-mono text-xs text-muted-foreground">
          {shortCommit(pullRequest.headSha)}
        </span>
      </div>
      <dl className="mt-6 grid gap-px overflow-hidden rounded-lg border bg-slate-200 sm:grid-cols-3">
        <EvidenceMetric
          label="Changed files"
          value={String(pullRequest.changedFiles)}
        />
        <EvidenceMetric
          label="Commits"
          value={String(pullRequest.commitCount)}
        />
        <EvidenceMetric
          label="Validation claims"
          value={String(validations.length)}
        />
        <EvidenceMetric
          label="Decisions"
          value={String(decisions.length)}
        />
        <EvidenceMetric label="Risks" value={String(risks.length)} />
        <EvidenceMetric
          label="Snapshot coverage"
          value={pullRequest.snapshotHeadMatch ? "Current" : "Stale"}
        />
      </dl>

      {validations.length ? (
        <div className="mt-7">
          <h3 className="text-sm font-semibold">Checks</h3>
          <div className="mt-3 divide-y rounded-lg border">
            {validations.map((validation, index) => {
              const passed = validation.status === "passed";
              return (
                <div
                  key={`${validation.command}-${index}`}
                  className="flex items-start justify-between gap-4 px-4 py-3.5"
                >
                  <div className="flex min-w-0 gap-3">
                    {passed ? (
                      <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
                    ) : (
                      <XCircle className="mt-0.5 size-4 shrink-0 text-red-600" />
                    )}
                    <code className="break-all font-mono text-xs leading-5">
                      {validation.command}
                    </code>
                  </div>
                  <span className="shrink-0 text-xs font-medium capitalize text-muted-foreground">
                    {validation.status}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      ) : null}

      {(decisions.length || risks.length) ? (
        <div className="mt-7 grid gap-6 lg:grid-cols-2">
          <RecordList
            title="Decisions"
            icon={MessageSquareText}
            records={decisions}
            empty="No decisions recorded."
          />
          <RecordList
            title="Risks"
            icon={AlertTriangle}
            records={risks}
            empty="No risks recorded."
          />
        </div>
      ) : null}

      {artifacts.length ? (
        <div className="mt-7">
          <h3 className="flex items-center gap-2 text-sm font-semibold">
            <Paperclip className="size-4 text-slate-500" />
            Evidence and artifacts
          </h3>
          <div className="mt-3 divide-y rounded-lg border">
            {artifacts.map((artifact, index) => {
              const href = artifactHref(detail.repository, artifact);
              const content = (
                <>
                  <span className="min-w-0">
                    <span className="block text-sm font-medium">
                      {artifact.description ||
                        artifact.path ||
                        artifact.url ||
                        artifact.uri ||
                        `Artifact ${index + 1}`}
                    </span>
                    <span className="mt-1 block truncate text-xs text-muted-foreground">
                      {artifact.type}
                      {artifact.mimeType ? ` · ${artifact.mimeType}` : ""}
                    </span>
                  </span>
                  {href ? (
                    <ExternalLink className="size-4 shrink-0 text-slate-400" />
                  ) : null}
                </>
              );
              return href ? (
                <a
                  key={`${href}-${index}`}
                  href={href}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center justify-between gap-4 px-4 py-3.5 transition-colors hover:bg-slate-50"
                >
                  {content}
                </a>
              ) : (
                <div
                  key={`${artifact.type}-${index}`}
                  className="flex items-center justify-between gap-4 px-4 py-3.5"
                >
                  {content}
                </div>
              );
            })}
          </div>
        </div>
      ) : null}
    </section>
  );
}

function EvidenceMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white px-4 py-4">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-sm font-semibold">{value}</dd>
    </div>
  );
}

function RecordList({
  title,
  icon: Icon,
  records: values,
  empty,
}: {
  title: string;
  icon: typeof AlertTriangle;
  records: Array<{ title: string; detail?: string }>;
  empty: string;
}) {
  return (
    <div>
      <h3 className="flex items-center gap-2 text-sm font-semibold">
        <Icon className="size-4 text-slate-500" />
        {title}
      </h3>
      {values.length ? (
        <ul className="mt-3 space-y-3">
          {values.map((record, index) => (
            <li key={`${record.title}-${index}`} className="text-sm">
              <p className="font-medium">{record.title}</p>
              {record.detail ? (
                <p className="mt-1 leading-5 text-muted-foreground">
                  {record.detail}
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-muted-foreground">{empty}</p>
      )}
    </div>
  );
}

function ReviewDecisionCard({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}) {
  const readiness = pullRequestReadiness(pullRequest);
  const ready = readiness.ready;
  return (
    <section
      className={`rounded-lg border p-5 ${
        ready
          ? "border-emerald-200 bg-emerald-50"
          : "border-amber-200 bg-amber-50"
      }`}
    >
      <div className="flex items-start gap-3">
        {ready ? (
          <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-emerald-700" />
        ) : (
          <AlertTriangle className="mt-0.5 size-5 shrink-0 text-amber-700" />
        )}
        <div>
          <h2 className="font-semibold">
            {readiness.title}
          </h2>
          <p className="mt-2 text-sm leading-6 opacity-75">
            {readiness.detail}
          </p>
        </div>
      </div>
      <Button
        asChild
        className={`mt-5 w-full ${
          ready ? "bg-emerald-700 hover:bg-emerald-800" : ""
        }`}
        variant={ready ? "default" : "outline"}
      >
        <a href={pullRequest.url} target="_blank" rel="noreferrer">
          {ready ? "Merge on GitHub" : "Resolve on GitHub"}
          <ExternalLink className="size-4" />
        </a>
      </Button>
      <p className="mt-3 text-xs leading-5 opacity-65">
        GitHub remains authoritative for repository permissions and merge
        execution.
      </p>
    </section>
  );
}

function PullRequestFacts({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}) {
  return (
    <section className="rounded-lg border bg-white p-5">
      <h2 className="font-semibold">Pull request</h2>
      <dl className="mt-5 space-y-4 text-sm">
        <Fact label="State" value={pullRequest.draft ? "Draft" : pullRequest.state} />
        <Fact
          label="Branches"
          value={`${pullRequest.headBranch} → ${pullRequest.baseBranch}`}
          mono
        />
        <Fact label="Head" value={shortCommit(pullRequest.headSha)} mono />
        <Fact label="Updated" value={humanDate(pullRequest.updatedAt)} />
        <Fact
          label="Reviews"
          value={statusLabel(pullRequest.reviewDecision || "not required")}
        />
        <Fact
          label="GitHub checks"
          value={`${pullRequest.checksPassed} passed · ${pullRequest.checksFailed} failed · ${pullRequest.checksPending} pending`}
        />
        <Fact
          label="Diff"
          value={`+${pullRequest.additions} −${pullRequest.deletions}`}
        />
      </dl>
    </section>
  );
}

function SnapshotContext({ detail }: { detail: DetailResponse }) {
  return (
    <section className="rounded-lg border bg-white p-5">
      <div className="flex items-center gap-2">
        <BookOpen className="size-4 text-blue-600" />
        <h2 className="font-semibold">Linked Snapshot</h2>
      </div>
      <p className="mt-4 text-sm font-semibold">{detail.summary.title}</p>
      <p className="mt-2 line-clamp-3 text-sm leading-6 text-muted-foreground">
        {detail.summary.outcome}
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <Badge variant="secondary">{detail.summary.type}</Badge>
        <Badge variant="outline">{detail.summary.id}</Badge>
      </div>
      <Button asChild variant="outline" className="mt-5 w-full">
        <Link to="/snapshots/$id" params={{ id: detail.summary.id }}>
          Open Snapshot
          <ArrowRight className="size-4" />
        </Link>
      </Button>
    </section>
  );
}

function Fact({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd
        className={`mt-1 break-words font-medium capitalize ${
          mono ? "font-mono text-xs normal-case" : ""
        }`}
      >
        {value}
      </dd>
    </div>
  );
}

function ReviewStateBadge({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}) {
  const readiness = pullRequestReadiness(pullRequest);
  return <Badge variant={readiness.variant}>{readiness.badge}</Badge>;
}

function UnlinkedEvidenceState({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}) {
  return (
    <section className="rounded-lg border border-amber-200 bg-amber-50 p-6">
      <AlertTriangle className="size-5 text-amber-700" />
      <h2 className="mt-4 text-lg font-semibold">No verified Snapshot is linked</h2>
      <p className="mt-2 max-w-[68ch] text-sm leading-6 text-amber-950/75">
        The pull request head{" "}
        <code className="font-mono">{shortCommit(pullRequest.headSha)}</code>{" "}
        does not match a completed eve Snapshot. Product evidence, plan
        alignment, and code curation cannot be trusted until a current Snapshot
        exists.
      </p>
    </section>
  );
}

function UnlinkedCodeState() {
  return (
    <section className="rounded-lg border bg-white p-8">
      <Code2 className="size-5 text-slate-500" />
      <h2 className="mt-4 text-lg font-semibold">Code review needs a Snapshot</h2>
      <p className="mt-2 max-w-[64ch] text-sm leading-6 text-muted-foreground">
        eve’s code viewer is scoped to the exact code recorded by a Snapshot.
        Create a fresh Snapshot at this pull request head to inspect its curated
        files and full diff.
      </p>
    </section>
  );
}

export function pullRequestReadiness(pullRequest: PullRequestSummary): {
  ready: boolean;
  badge: string;
  title: string;
  detail: string;
  variant: "success" | "warning" | "destructive" | "secondary";
} {
  if (pullRequest.readyToMerge) {
    return {
      ready: true,
      badge: "Ready to merge",
      title: "Ready to merge",
      detail:
        "eve trusts the linked Snapshot and GitHub currently permits the merge.",
      variant: "success",
    };
  }
  if (!pullRequest.snapshotId) {
    return {
      ready: false,
      badge: "Snapshot required",
      title: "Snapshot required",
      detail:
        "Create a Snapshot for the current pull request code before relying on eve evidence.",
      variant: "warning",
    };
  }
  if (!pullRequest.snapshotHeadMatch) {
    return {
      ready: false,
      badge: "Review stale",
      title: "Review is stale",
      detail: `The Snapshot no longer matches ${shortCommit(pullRequest.headSha)}. Create an updated Snapshot.`,
      variant: "destructive",
    };
  }
  if (pullRequest.mergeability === "conflicting") {
    return {
      ready: false,
      badge: "Merge conflict",
      title: "Resolve the merge conflict",
      detail: `GitHub reports conflicts with ${pullRequest.baseBranch}. Update the branch before merging.`,
      variant: "destructive",
    };
  }
  if (!pullRequest.planValid) {
    return {
      ready: false,
      badge: "Plan invalid",
      title: "Plan revision is no longer valid",
      detail:
        "The Snapshot does not fulfill the current locked Plan record for this branch.",
      variant: "destructive",
    };
  }
  if (pullRequest.scopeDrift) {
    return {
      ready: false,
      badge: "Scope drift",
      title: "Scope drift blocks merge",
      detail:
        "The implementation includes paths outside the locked plan scope.",
      variant: "destructive",
    };
  }
  if (!pullRequest.planAligned || !pullRequest.eveChecksPassed) {
    return {
      ready: false,
      badge: "Evidence incomplete",
      title: pullRequest.planAligned
        ? "eve checks incomplete"
        : "Plan alignment incomplete",
      detail: "The linked Snapshot has unresolved eve evidence.",
      variant: "warning",
    };
  }
  if (pullRequest.draft) {
    return {
      ready: false,
      badge: "Draft",
      title: "Pull request is a draft",
      detail: "Mark the pull request ready for review before merging.",
      variant: "secondary",
    };
  }
  return {
    ready: false,
    badge: "Merge blocked",
    title: "GitHub blocks merge",
    detail:
      "GitHub checks, reviews, branch protection, or mergeability still block this pull request.",
    variant: "warning",
  };
}

function findLockedRevision(detail?: DetailResponse) {
  if (!detail?.planRecord) return undefined;
  return detail.planRecord.revisions.find(
    (revision) => revision.revision === detail.planRecord?.lockedRevision,
  );
}

export function parseAcceptanceCriteria(value?: string) {
  if (!value?.trim()) return [];
  return value
    .split(/\r?\n/)
    .map((line) =>
      line
        .trim()
        .replace(/^[-*]\s+/, "")
        .replace(/^\[[ xX]\]\s*/, "")
        .replace(/^\d+[.)]\s+/, "")
        .trim(),
    )
    .filter(Boolean);
}

function behaviorClaims(behavior: Behavior) {
  return (["added", "changed", "fixed", "removed"] as const).flatMap(
    (kind) =>
      (behavior[kind] ?? []).map((claim) => ({
        kind,
        description: claim.description,
      })),
  );
}

function records(values: unknown[]) {
  return values.flatMap((value) => {
    if (!value || typeof value !== "object") return [];
    const record = value as Record<string, unknown>;
    const title =
      typeof record.title === "string"
        ? record.title
        : typeof record.description === "string"
          ? record.description
          : "";
    if (!title) return [];
    const detail =
      typeof record.rationale === "string"
        ? record.rationale
        : typeof record.mitigation === "string"
          ? record.mitigation
          : undefined;
    return [{ title, detail }];
  });
}

function artifactHref(
  repository: string,
  artifact: { url?: string; uri?: string; path?: string },
) {
  const direct = artifact.url || artifact.uri;
  if (direct) return direct;
  const path = artifact.path?.replace(/^\/+/, "");
  const prefix = ".eve/artifacts/";
  if (!path?.startsWith(prefix)) return undefined;
  return `/api/repos/${encodeURIComponent(repository)}/artifacts/${path
    .slice(prefix.length)
    .split("/")
    .map(encodeURIComponent)
    .join("/")}`;
}
