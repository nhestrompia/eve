import { describe, expect, it } from "vitest";
import type { SnapshotArtifact } from "../types";
import {
  artifactHref,
  artifactKind,
  localArtifactHref,
} from "./artifacts";

describe("artifactHref", () => {
  it("routes repository-relative Playwright screenshots through the artifact API", () => {
    const artifact: SnapshotArtifact = {
      type: "screenshot",
      path: "output/playwright/evolution-hero-desktop.png",
      mimeType: "image/png",
    };

    expect(artifactHref("eve", artifact)).toBe(
      "/api/repos/eve/files/output/playwright/evolution-hero-desktop.png",
    );
  });

  it("routes legacy local artifact URIs through the artifact API", () => {
    const artifact: SnapshotArtifact = {
      type: "note",
      uri: "output/playwright/evolution-hero-mobile.png",
    };

    expect(artifactHref("eve", artifact)).toBe(
      "/api/repos/eve/files/output/playwright/evolution-hero-mobile.png",
    );
  });

  it("preserves existing EVE artifact and external URLs", () => {
    expect(
      localArtifactHref(
        "eve",
        ".eve/artifacts/snap_123/screenshot with spaces.png",
      ),
    ).toBe(
      "/api/repos/eve/artifacts/snap_123/screenshot%20with%20spaces.png",
    );
    expect(
      artifactHref("eve", {
        type: "url",
        url: "https://example.com/evidence",
      }),
    ).toBe("https://example.com/evidence");
    expect(
      artifactHref("eve", {
        type: "log",
        path: "/private/tmp/eve-high-cpu.sample.txt",
      }),
    ).toBe(
      "/api/repos/eve/files?path=%2Fprivate%2Ftmp%2Feve-high-cpu.sample.txt",
    );
  });

  it("does not turn prose notes or escaping paths into file URLs", () => {
    expect(
      artifactHref("eve", {
        type: "note",
        uri: "The validation was reviewed manually.",
      }),
    ).toBeUndefined();
    expect(
      artifactHref("eve", {
        type: "note",
        uri: "Unrelated change in ui/src/components/sidebar.tsx.",
      }),
    ).toBeUndefined();
    expect(
      localArtifactHref("eve", "../outside/repository.log"),
    ).toBeUndefined();
  });
});

describe("artifactKind", () => {
  it("classifies images, logs, and generic files separately", () => {
    expect(
      artifactKind({
        type: "screenshot",
        path: "output/playwright/page.png",
      }),
    ).toBe("image");
    expect(
      artifactKind({
        type: "log",
        path: "output/diagnostics/sample.txt",
      }),
    ).toBe("log");
    expect(
      artifactKind({
        type: "note",
        path: "output/report.json",
      }),
    ).toBe("file");
  });
});
