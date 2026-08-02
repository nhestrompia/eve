import { shortCommit } from "../../format";
import type { DetailResponse, GitCommit } from "../../types";

export function commitsForDetail(detail: DetailResponse): GitCommit[] {
  const snapshotHash =
    detail.snapshot.implementation.gitState || detail.summary.snapshot;
  if (!snapshotHash) return [];

  const matchedCommit = detail.commits.find(
    (commit) =>
      commit.hash === snapshotHash ||
      commit.shortHash === snapshotHash ||
      shortCommit(commit.hash) === shortCommit(snapshotHash),
  );
  if (matchedCommit) return [matchedCommit];

  return [
    {
      hash: snapshotHash,
      shortHash: shortCommit(snapshotHash),
      subject: "Snapshot commit",
      authorName: "",
      authoredAt: "",
      committedAt: "",
    },
  ];
}

export function githubCommitUrl(
  remoteUrl: string | undefined,
  hash: string,
): string | undefined {
  if (!remoteUrl || !hash) return undefined;
  const trimmed = remoteUrl.trim().replace(/\.git$/, "");
  const sshMatch = trimmed.match(/^git@github\.com:(.+)$/);
  const baseUrl = sshMatch ? `https://github.com/${sshMatch[1]}` : trimmed;
  if (!/^https:\/\/github\.com\//i.test(baseUrl)) return undefined;
  return `${baseUrl}/commit/${hash}`;
}
