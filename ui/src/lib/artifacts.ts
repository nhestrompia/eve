import type { SnapshotArtifact } from "../types";

export type ArtifactKind = "image" | "log" | "file";

export function artifactSource(artifact: SnapshotArtifact) {
  return artifact.path || artifact.url || artifact.uri;
}

export function artifactKind(artifact: SnapshotArtifact): ArtifactKind {
  const type = artifact.type.toLowerCase();
  const source = (artifactSource(artifact) || "").toLowerCase();
  if (
    artifact.mimeType?.startsWith("image/") ||
    type.includes("screenshot") ||
    type.includes("image") ||
    /\.(png|jpe?g|gif|webp|avif|svg)$/i.test(source)
  ) {
    return "image";
  }
  if (
    type === "log" ||
    type.endsWith("_log") ||
    artifact.mimeType?.startsWith("text/") ||
    /\.(log|out|trace|txt)$/i.test(source)
  ) {
    return "log";
  }
  return "file";
}

export function artifactHref(repo: string, artifact: SnapshotArtifact) {
  const external = [artifact.url, artifact.uri, artifact.path].find(
    (value) => value && /^https?:\/\//i.test(value),
  );
  if (external) return external;

  if (artifact.path) return localArtifactHref(repo, artifact.path);
  if (artifact.uri && looksLikeLocalArtifactPath(artifact.uri)) {
    return localArtifactHref(repo, artifact.uri);
  }
  return undefined;
}

export function localArtifactHref(repo: string, artifactPath?: string) {
  if (!artifactPath) return undefined;
  if (/^https?:\/\//i.test(artifactPath)) return artifactPath;

  const normalized = artifactPath
    .trim()
    .replaceAll("\\", "/")
    .replace(/^\.\/+/, "");
  const parts = normalized.split("/");
  if (!normalized || parts.some((part) => part === "..")) {
    return undefined;
  }

  const encodedRepo = encodeURIComponent(repo);
  if (normalized.startsWith("/")) {
    return `/api/repos/${encodedRepo}/files?path=${encodeURIComponent(normalized)}`;
  }
  const evePrefix = ".eve/artifacts/";
  if (normalized.startsWith(evePrefix)) {
    const relative = normalized.slice(evePrefix.length);
    if (!relative) return undefined;
    return `/api/repos/${encodedRepo}/artifacts/${encodePath(relative)}`;
  }
  return `/api/repos/${encodedRepo}/files/${encodePath(normalized)}`;
}

function looksLikeLocalArtifactPath(value: string) {
  const trimmed = value.trim();
  if (/\s/.test(trimmed)) return false;
  return (
    trimmed.startsWith("./") ||
    trimmed.startsWith(".eve/") ||
    trimmed.includes("/") ||
    /\.(png|jpe?g|gif|webp|avif|svg|log|out|trace|txt|json|md)$/i.test(trimmed)
  );
}

function encodePath(value: string) {
  return value.split("/").map(encodeURIComponent).join("/");
}
