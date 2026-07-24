import XCTest
@testable import EVEApproval

final class PlanModelsTests: XCTestCase {
    func testDecodesPendingQueueItem() throws {
        let data = Data(#"""
        {
          "planRequestId":"planreq_12345678",
          "repository":"eve",
          "repositoryRoot":"/tmp/eve",
          "branch":"main",
          "state":"pending_approval",
          "currentRevision":1,
          "availableSuites":["change","extended"],
          "revisions":[{
            "revision":1,
            "source":"agent",
            "goal":"Add a gate",
            "acceptanceCriteria":"- It resumes",
            "allowedPathGlobs":["cmd/**"],
            "milestones":[],
            "resolvedCheckIds":["go-test"],
            "branch":"main"
          }]
        }
        """#.utf8)
        let request = try JSONDecoder().decode(PlanRequest.self, from: data)
        XCTAssertEqual(request.id, "planreq_12345678")
        XCTAssertEqual(request.current?.allowedPathGlobs, ["cmd/**"])
        XCTAssertEqual(request.suiteOptions, ["change", "extended"])
    }

    func testEditedProposalRequiresGoalCriteriaAndScope() {
        let revision = PlanRevision(
            revision: 1, source: "agent", goal: "", acceptanceCriteria: "",
            allowedPathGlobs: [], milestones: [], configuredSuite: nil, resolvedSuite: nil,
            resolvedCheckIds: [], branch: "main"
        )
        XCTAssertNotNil(PlanProposal(revision: revision).validationMessage)
    }

    func testRejectionRequiresFeedback() {
        XCTAssertEqual(rejectionValidationMessage(" \n"), "Rejection feedback is required.")
        XCTAssertNil(rejectionValidationMessage("Please narrow the scope."))
    }

    func testEditedProposalPreservesConfiguredSuiteChoice() {
        let revision = PlanRevision(
            revision: 2, source: "human", goal: "Ship safely", acceptanceCriteria: "- Checks pass",
            allowedPathGlobs: ["cmd/**"], milestones: [], configuredSuite: "extended", resolvedSuite: "extended",
            resolvedCheckIds: ["unit", "integration"], branch: "main"
        )
        XCTAssertEqual(PlanProposal(revision: revision).requiredSuite, "extended")
    }

    func testStaleAndOfflineStatesRemainActionable() throws {
        let data = Data(#"""
        {
          "planRequestId":"planreq_stale001",
          "repository":"eve",
          "repositoryRoot":"/tmp/eve",
          "branch":"main",
          "state":"stale",
          "currentRevision":1,
          "staleReasons":["repository HEAD changed"],
          "revisions":[]
        }
        """#.utf8)
        let request = try JSONDecoder().decode(PlanRequest.self, from: data)
        XCTAssertEqual(request.staleReasons, ["repository HEAD changed"])
        XCTAssertEqual(QueueState.offline("Port 4317 is busy"), .offline("Port 4317 is busy"))
    }

    func testFreshPendingRequestReplacesStaleSelection() throws {
        let stale = try decodeRequest(id: "planreq_stale001", state: "stale")
        let pending = try decodeRequest(id: "planreq_pending01", state: "pending_approval")
        XCTAssertEqual(
            preferredPlanSelection(currentID: stale.id, requests: [stale, pending]),
            pending.id
        )
    }

    func testNewPendingRequestDoesNotStealTheActivePendingSelection() throws {
        let active = try decodeRequest(id: "planreq_pending01", state: "pending_approval")
        let incoming = try decodeRequest(id: "planreq_pending02", state: "pending_approval")

        XCTAssertEqual(
            preferredPlanSelection(currentID: active.id, requests: [incoming, active]),
            active.id
        )
    }

    func testMultiplePendingRequestsAreKeptAheadOfStaleRequests() throws {
        let stale = try decodeRequest(id: "planreq_stale001", state: "stale", repository: "alpha")
        let second = try decodeRequest(id: "planreq_pending02", state: "pending_approval", repository: "zeta")
        let first = try decodeRequest(id: "planreq_pending01", state: "pending_approval", repository: "alpha")

        XCTAssertEqual(
            orderedPlanRequests([stale, second, first]).map(\.id),
            [first.id, second.id, stale.id]
        )
    }

    func testAttentionOnlyIncludesNewPendingRequests() throws {
        let existing = try decodeRequest(id: "planreq_pending01", state: "pending_approval")
        let fresh = try decodeRequest(id: "planreq_pending02", state: "pending_approval")
        let stale = try decodeRequest(id: "planreq_stale001", state: "stale")

        XCTAssertEqual(
            newPendingPlanIDs(previous: [existing.id], requests: [existing, fresh, stale]),
            [fresh.id]
        )
        XCTAssertTrue(
            newPendingPlanIDs(previous: [existing.id, fresh.id], requests: [existing, fresh]).isEmpty
        )
    }

    private func decodeRequest(id: String, state: String, repository: String = "eve") throws -> PlanRequest {
        let data = Data("""
        {
          "planRequestId":"\(id)",
          "repository":"\(repository)",
          "repositoryRoot":"/tmp/eve",
          "branch":"main",
          "state":"\(state)",
          "currentRevision":1,
          "revisions":[]
        }
        """.utf8)
        return try JSONDecoder().decode(PlanRequest.self, from: data)
    }
}
