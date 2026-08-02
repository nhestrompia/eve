import { Badge } from "../../components/ui/badge";
import type { PullRequestSummary } from "../../types";

export function PullRequestStateBadge({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}): React.JSX.Element {
  if (pullRequest.draft) return <Badge variant="secondary">Draft</Badge>;
  return <Badge variant="success">Open</Badge>;
}
