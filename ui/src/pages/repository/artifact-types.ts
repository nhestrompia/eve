import type { ArtifactKind } from "../../lib/artifacts";

export type ArtifactCardRow = {
  id: string;
  type: string;
  description: string;
  href?: string;
  imageSrc?: string;
  source?: string;
  kind: ArtifactKind;
};
