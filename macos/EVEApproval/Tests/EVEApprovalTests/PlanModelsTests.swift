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

    func testPlanStatusCopyIsMutuallyExclusive() throws {
        let stale = try decodeRequest(id: "planreq_stale001", state: "stale")
        let locked = try decodeRequest(id: "planreq_locked01", state: "locked")
        let pending = try decodeRequest(id: "planreq_pending01", state: "pending_approval")

        XCTAssertEqual(stale.statusTitle, "Stale")
        XCTAssertEqual(stale.terminalActionMessage, "Approval is disabled because the repository changed. Ask the agent to declare a fresh plan.")
        XCTAssertTrue(stale.canRemoveFromQueue)
        XCTAssertEqual(locked.statusTitle, "Locked")
        XCTAssertEqual(locked.terminalActionMessage, "This plan is locked. The agent can continue with implementation.")
        XCTAssertFalse(locked.canRemoveFromQueue)
        XCTAssertEqual(pending.statusTitle, "Awaiting approval")
        XCTAssertNil(pending.terminalActionMessage)
        XCTAssertFalse(pending.canRemoveFromQueue)
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

    func testDuplicatePlanRequestsCollapseToNewestEquivalentRequest() throws {
        let older = try decodeRequest(
            id: "planreq_publish_a",
            state: "stale",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:07:34Z"
        )
        let newer = try decodeRequest(
            id: "planreq_publish_b",
            state: "stale",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:09:32Z"
        )

        XCTAssertEqual(collapsedDuplicatePlanRequests([older, newer]).map(\.id), [newer.id])
    }

    func testDuplicatePlanRequestsPreferPendingRequestOverStaleRequest() throws {
        let stale = try decodeRequest(
            id: "planreq_publish_stale",
            state: "stale",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:09:32Z"
        )
        let pending = try decodeRequest(
            id: "planreq_publish_pending",
            state: "pending_approval",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:08:00Z"
        )

        XCTAssertEqual(collapsedDuplicatePlanRequests([stale, pending]).map(\.id), [pending.id])
    }

    func testDuplicatePlanCollapsePreservesVisibleGroupOrder() throws {
        let unrelated = try decodeRequest(
            id: "planreq_other",
            state: "pending_approval",
            goal: "Improve the approval copy."
        )
        let olderDuplicate = try decodeRequest(
            id: "planreq_publish_a",
            state: "stale",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:07:34Z"
        )
        let newerDuplicate = try decodeRequest(
            id: "planreq_publish_b",
            state: "stale",
            goal: "Publish the macOS eve app release asset used by the npm installer.",
            updatedAt: "2026-07-25T20:09:32Z"
        )

        XCTAssertEqual(
            collapsedDuplicatePlanRequests([unrelated, olderDuplicate, newerDuplicate]).map(\.id),
            [unrelated.id, newerDuplicate.id]
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

    func testReviewQueueEmptiesOnlyWhenLastDisplayedRequestLeaves() throws {
        let pending = try decodeRequest(id: "planreq_pending01", state: "pending_approval")
        let stale = try decodeRequest(id: "planreq_stale001", state: "stale")

        XCTAssertTrue(didReviewQueueEmptyAfterPendingRequests(previousPendingCount: 1, requests: []))
        XCTAssertFalse(didReviewQueueEmptyAfterPendingRequests(previousPendingCount: 1, requests: [stale]))
        XCTAssertFalse(didReviewQueueEmptyAfterPendingRequests(previousPendingCount: 2, requests: [pending]))
        XCTAssertFalse(didReviewQueueEmptyAfterPendingRequests(previousPendingCount: 0, requests: []))
    }

    func testMenuBarApprovalWindowUsesCompactEmptySize() {
        XCTAssertEqual(approvalWindowSize(hasRequests: false), CGSize(width: 420, height: 260))
        XCTAssertEqual(approvalWindowSize(hasRequests: true), CGSize(width: 760, height: 680))
    }

    func testMenuBarIconChangesWhenPlansAreWaiting() {
        XCTAssertEqual(menuBarSystemImageName(pendingCount: 0), "checkmark.shield")
        XCTAssertEqual(menuBarSystemImageName(pendingCount: 2), "checkmark.shield.fill")
    }

    private func decodeRequest(
        id: String,
        state: String,
        repository: String = "eve",
        goal: String = "Add a gate",
        updatedAt: String = "2026-07-25T20:00:00Z"
    ) throws -> PlanRequest {
        let data = Data("""
        {
          "planRequestId":"\(id)",
          "repository":"\(repository)",
          "repositoryRoot":"/tmp/eve",
          "branch":"main",
          "state":"\(state)",
          "currentRevision":1,
          "createdAt":"2026-07-25T19:59:00Z",
          "updatedAt":"\(updatedAt)",
          "revisions":[{
            "revision":1,
            "source":"agent",
            "goal":"\(goal)",
            "acceptanceCriteria":"- Criteria",
            "allowedPathGlobs":[".github/workflows/release.yml"],
            "milestones":[],
            "resolvedCheckIds":[],
            "branch":"main"
          }]
        }
        """.utf8)
        return try JSONDecoder().decode(PlanRequest.self, from: data)
    }
}
