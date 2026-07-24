import Foundation

@MainActor
final class PlanQueueStore: ObservableObject {
    @Published private(set) var requests: [PlanRequest] = []
    @Published private(set) var state: QueueState = .loading
    @Published var selectedID: PlanRequest.ID?
    @Published var notice: String?

    private let client: EVEClient
    private var refreshTask: Task<Void, Never>?

    init(client: EVEClient = EVEClient()) {
        self.client = client
    }

    var selected: PlanRequest? {
        requests.first { $0.id == selectedID } ?? requests.first
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
        do {
            requests = try await client.reviewQueue()
            selectedID = preferredPlanSelection(currentID: selectedID, requests: requests)
            state = .online
        } catch {
            state = .offline(error.localizedDescription)
        }
    }

    func approve(_ request: PlanRequest, proposal: PlanProposal?) async {
        do {
            _ = try await client.approve(request, proposal: proposal)
            notice = "Plan approved and locked."
            await refresh()
        } catch {
            await recoverTerminalState(for: request, fallback: error)
        }
    }

    func reject(_ request: PlanRequest, feedback: String) async {
        do {
            _ = try await client.reject(request, feedback: feedback)
            notice = "Plan rejected with feedback."
            await refresh()
        } catch {
            await recoverTerminalState(for: request, fallback: error)
        }
    }

    private func recoverTerminalState(for request: PlanRequest, fallback error: Error) async {
        if let refreshed = try? await client.planRequest(id: request.planRequestId),
           refreshed.state == "stale" {
            requests.removeAll { $0.id == refreshed.id }
            requests.insert(refreshed, at: 0)
            selectedID = refreshed.id
            notice = "Repository context changed. Review the exact stale reasons and declare a fresh plan."
            state = .online
            return
        }
        state = .offline(error.localizedDescription)
    }
}
