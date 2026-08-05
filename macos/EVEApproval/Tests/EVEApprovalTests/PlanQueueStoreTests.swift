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

    @MainActor
    func testDismissingNonStalePlanDoesNothing() async throws {
        let pending = try makeRequest(id: "planreq_pending01", state: "pending_approval")
        let client = PlanQueueClientStub(reviewQueues: [[pending]])
        let store = PlanQueueStore(client: client)

        await store.refresh()
        await store.dismiss(pending)

        XCTAssertEqual(store.requests.map(\.id), [pending.id])
        XCTAssertEqual(client.dismissCallCount, 0)
        XCTAssertFalse(store.isDismissing(pending))
        XCTAssertEqual(store.state, .online)
    }

    @MainActor
    func testFailedDismissalRefreshesTheQueueAndKeepsThePlanVisible() async throws {
        let stale = try makeRequest(id: "planreq_stale002", state: "stale")
        let client = PlanQueueClientStub(
            reviewQueues: [[stale], [stale]],
            dismissError: StubError.dismissFailed
        )
        let store = PlanQueueStore(client: client)

        await store.refresh()
        await store.dismiss(stale)

        XCTAssertEqual(store.requests.map(\.id), [stale.id])
        XCTAssertEqual(client.dismissCallCount, 1)
        XCTAssertFalse(store.isDismissing(stale))
        XCTAssertEqual(store.state, .online)
    }

    @MainActor
    func testConcurrentDismissalsSendOnlyOneRequest() async throws {
        let stale = try makeRequest(id: "planreq_stale003", state: "stale")
        let gate = DismissGate()
        let client = PlanQueueClientStub(reviewQueues: [[stale], []], dismissGate: gate)
        let store = PlanQueueStore(client: client)

        await store.refresh()
        let dismissal = Task { @MainActor in
            await store.dismiss(stale)
        }

        await gate.waitUntilCalled()

        let callCount = await gate.callCount
        XCTAssertEqual(callCount, 1)
        XCTAssertTrue(store.isDismissing(stale))

        await store.dismiss(stale)
        let callCountAfterDuplicate = await gate.callCount
        XCTAssertEqual(callCountAfterDuplicate, 1)

        await gate.release()
        await dismissal.value

        XCTAssertTrue(store.requests.isEmpty)
        XCTAssertFalse(store.isDismissing(stale))
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

    private let dismissError: Error?
    private let dismissGate: DismissGate?
    private(set) var dismissCallCount = 0

    init(
        reviewQueues: [[PlanRequest]],
        dismissError: Error? = nil,
        dismissGate: DismissGate? = nil
    ) {
        self.reviewQueues = reviewQueues
        self.dismissError = dismissError
        self.dismissGate = dismissGate
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
        dismissCallCount += 1
        if let dismissGate {
            await dismissGate.wait()
        }
        if let dismissError {
            throw dismissError
        }
        request
    }
}

private actor DismissGate {
    private(set) var callCount = 0
    private var calledContinuation: CheckedContinuation<Void, Never>?
    private var continuation: CheckedContinuation<Void, Never>?

    func wait() async {
        callCount += 1
        calledContinuation?.resume()
        calledContinuation = nil
        await withCheckedContinuation { continuation in
            self.continuation = continuation
        }
    }

    func waitUntilCalled() async {
        guard callCount == 0 else { return }
        await withCheckedContinuation { continuation in
            calledContinuation = continuation
        }
    }

    func release() {
        continuation?.resume()
        continuation = nil
    }
}

private enum StubError: Error {
    case unimplemented
    case dismissFailed
}
