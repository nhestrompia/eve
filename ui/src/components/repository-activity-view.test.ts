import { describe, expect, it } from "vitest";
import { fittingRepositoryCount } from "./repository-activity-view";

describe("fittingRepositoryCount", () => {
  it("only returns the number of complete repository cards that fit", () => {
    expect(fittingRepositoryCount(1_180, 6)).toBe(4);
    expect(fittingRepositoryCount(1_320, 6)).toBe(5);
  });

  it("keeps one complete card on narrow screens and never exceeds the total", () => {
    expect(fittingRepositoryCount(220, 5)).toBe(1);
    expect(fittingRepositoryCount(2_000, 3)).toBe(3);
    expect(fittingRepositoryCount(2_000, 0)).toBe(0);
  });
});
