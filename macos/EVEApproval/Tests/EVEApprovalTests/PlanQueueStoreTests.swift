import XCTest
@testable import EVEApproval

final class PlanQueueStoreTests: XCTestCase {
    @MainActor
    func testDismissingLastStalePlanDrainsTheApprovalQueue() async throws {
        let stale = try makeRequest(id: "planreq_stale001", state: "stale")
        let client = PlanQueueClientStub(reviewQueues: [[stale], []])
        let store = PlanQueueStore(client: client)
        var queueDrained = false
        store.onPendingQueueDrained = {
            queueDrained = true
        }

        await store.refresh()
        await store.dismiss(stale)

        XCTAssertTrue(store.requests.isEmpty)
        XCTAssertTrue(queueDrained)
        XCTAssertEqual(store.state, .online)
    }

    private func makeRequest(id: String, state: String) throws -> PlanRequest {
        let data = Data("""
        {
          "planRequestId":"\(id)",
          "repository":"eve",
          "repositoryRoot":"/tmp/eve",
          "branch":"main",
          "state":"\(state)",
          "currentRevision":1,
          "staleReasons":["repository HEAD changed"],
          "revisions":[{
            "revision":1,
            "source":"agent",
            "goal":"Add a gate",
            "acceptanceCriteria":"- Criteria",
            "allowedPathGlobs":["cmd/**"],
            "milestones":[],
            "resolvedCheckIds":[],
            "branch":"main"
          }]
        }
        """.utf8)
        return try JSONDecoder().decode(PlanRequest.self, from: data)
    }
}

private final class PlanQueueClientStub: PlanQueueClient {
    private var reviewQueues: [[PlanRequest]]

    init(reviewQueues: [[PlanRequest]]) {
        self.reviewQueues = reviewQueues
    }

    func health() async -> Bool {
        true
    }

    func reviewQueue() async throws -> [PlanRequest] {
        reviewQueues.isEmpty ? [] : reviewQueues.removeFirst()
    }

    func planRequest(id: String) async throws -> PlanRequest {
        throw StubError.unimplemented
    }

    func approve(_ request: PlanRequest, proposal: PlanProposal?) async throws -> PlanRequest {
        throw StubError.unimplemented
    }

    func reject(_ request: PlanRequest, feedback: String) async throws -> PlanRequest {
        throw StubError.unimplemented
    }

    func dismiss(_ request: PlanRequest) async throws -> PlanRequest {
        request
    }
}

private enum StubError: Error {
    case unimplemented
}
