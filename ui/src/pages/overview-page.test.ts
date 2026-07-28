import { describe, expect, it } from "vitest";
import type { AgentActivity } from "../types";
import { buildAgentRows } from "./overview-page";

function agent(status: AgentActivity["status"], label: string): AgentActivity {
  return {
    agentId: `agent_${status}`,
    provider: "Codex",
    repository: "eve",
    planRequestId: `planreq_${status}`,
    label,
    status,
  };
}

describe("overview active agents", () => {
  it("keeps offline work visible without counting it as live", () => {
    expect(buildAgentRows([agent("offline", "Prepare eve 0.5.0 release")])[0]?.status).toBe("offline");
  });

  it("sorts live agents ahead of disconnected plans", () => {
    expect(buildAgentRows([
      agent("offline", "Old task"),
      agent("running", "Fix live dashboard updates"),
      agent("waiting", "Await approval"),
    ]).map((row) => row.status)).toEqual(["running", "waiting", "offline"]);
  });
});
