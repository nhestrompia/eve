import Foundation

@MainActor
final class PlanQueueStore: ObservableObject {
    @Published private(set) var requests: [PlanRequest] = []
    @Published private(set) var state: QueueState = .loading
    @Published var selectedID: PlanRequest.ID?
    @Published private(set) var notice: QueueNotice?
    @Published private(set) var dismissingIDs: Set<PlanRequest.ID> = []

    var onNewPendingRequests: (([PlanRequest]) -> Void)?
    var onPendingQueueDrained: (() -> Void)?

    private let client: any PlanQueueClient
    private var refreshTask: Task<Void, Never>?
    private var seenPendingIDs: Set<PlanRequest.ID> = []

    init(client: any PlanQueueClient = EVEClient()) {
        self.client = client
    }

    var selected: PlanRequest? {
        requests.first { $0.id == selectedID } ?? requests.first
    }

    var pendingCount: Int {
        requests.filter(\.canApprove).count
    }

    func start() {
        refreshTask?.cancel()
        refreshTask = Task {
            do {
                try await DaemonLauncher.startIfNeeded(client: client)
                while !Task.isCancelled {
                    await refresh()
                    try? await Task.sleep(nanoseconds: 2_000_000_000)
                }
            } catch {
                state = .offline(error.localizedDescription)
            }
        }
    }

    func refresh() async {
        await refresh(previousRequestCount: requests.count)
    }

    func isDismissing(_ request: PlanRequest) -> Bool {
        dismissingIDs.contains(request.id)
    }

    private func refresh(previousRequestCount: Int) async {
        do {
            let refreshed = orderedPlanRequests(collapsedDuplicatePlanRequests(try await client.reviewQueue()))
            let newIDs = newPendingPlanIDs(previous: seenPendingIDs, requests: refreshed)
            requests = refreshed
            seenPendingIDs = Set(
                refreshed
                    .filter(\.canApprove)
                    .map(\.id)
            )
            selectedID = preferredPlanSelection(currentID: selectedID, requests: requests)
            state = .online
            if !newIDs.isEmpty {
                let newRequests = refreshed.filter { newIDs.contains($0.id) }
                onNewPendingRequests?(newRequests)
            }
            if didReviewQueueEmptyAfterDisplayedRequests(previousRequestCount: previousRequestCount, requests: refreshed) {
                onPendingQueueDrained?()
            }
        } catch {
            state = .offline(error.localizedDescription)
        }
    }

    func approve(_ request: PlanRequest, proposal: PlanProposal?) async {
        do {
            let updated = try await client.approve(request, proposal: proposal)
            notice = QueueNotice(
                requestID: request.id,
                state: updated.state,
                message: noticeMessage(afterApproving: updated)
            )
            await refresh()
        } catch {
            await recoverTerminalState(for: request, fallback: error)
        }
    }

    func reject(_ request: PlanRequest, feedback: String) async {
        do {
            _ = try await client.reject(request, feedback: feedback)
            notice = QueueNotice(
                requestID: request.id,
                state: "rejected",
                message: "Plan rejected with feedback."
            )
            await refresh()
        } catch {
            await recoverTerminalState(for: request, fallback: error)
        }
    }

    func dismiss(_ request: PlanRequest) async {
        guard request.canDismissFromQueue, dismissingIDs.insert(request.id).inserted else { return }
        let previousRequestCount = requests.count
        defer { dismissingIDs.remove(request.id) }
        do {
            _ = try await client.dismiss(request)
            requests.removeAll { $0.id == request.id }
            selectedID = preferredPlanSelection(currentID: selectedID, requests: requests)
            await refresh(previousRequestCount: previousRequestCount)
        } catch {
            await refresh()
        }
    }

    func noticeMessage(for request: PlanRequest) -> String? {
        guard request.canApprove else {
            return nil
        }
        return notice?.requestID == request.id && notice?.state == request.state ? notice?.message : nil
    }

    private func recoverTerminalState(for request: PlanRequest, fallback error: Error) async {
        if let refreshed = try? await client.planRequest(id: request.planRequestId),
           refreshed.state == "stale" {
            requests.removeAll { $0.id == refreshed.id }
            requests.insert(refreshed, at: 0)
            selectedID = refreshed.id
            notice = QueueNotice(
                requestID: refreshed.id,
                state: refreshed.state,
                message: "Repository context changed. Review the exact stale reasons and declare a fresh plan."
            )
            state = .online
            return
        }
        state = .offline(error.localizedDescription)
    }

    private func noticeMessage(afterApproving request: PlanRequest) -> String {
        switch request.state {
        case "locked":
            return "Plan approved and locked."
        case "stale":
            return "Repository context changed. Review the exact stale reasons and declare a fresh plan."
        default:
            return "Plan state changed to \(request.statusTitle.lowercased())."
        }
    }
}
