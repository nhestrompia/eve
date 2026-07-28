import { ArrowRight } from "lucide-react";

export function PullRequestBranchRoute({
  headBranch,
  baseBranch,
}: {
  headBranch: string;
  baseBranch: string;
}) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <span className="max-w-[28ch] truncate rounded-md bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700">
        {headBranch}
      </span>
      <ArrowRight className="size-3.5 shrink-0" />
      <span className="rounded-md bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700">
        {baseBranch}
      </span>
    </span>
  );
}
