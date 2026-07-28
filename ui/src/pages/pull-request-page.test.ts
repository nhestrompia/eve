import { describe, expect, it } from "vitest";
import { parseAcceptanceCriteria } from "./pull-request-page";

describe("parseAcceptanceCriteria", () => {
  it("normalizes markdown bullets and task-list markers", () => {
    expect(
      parseAcceptanceCriteria(
        "- [x] Show a pull request count\n- [ ] Link exact-head Snapshots\n3. Keep code secondary",
      ),
    ).toEqual([
      "Show a pull request count",
      "Link exact-head Snapshots",
      "Keep code secondary",
    ]);
  });

  it("omits empty lines", () => {
    expect(parseAcceptanceCriteria("\n- First\n\n* Second\n")).toEqual([
      "First",
      "Second",
    ]);
  });
});
