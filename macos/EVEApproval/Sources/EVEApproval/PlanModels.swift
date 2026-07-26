import Foundation

struct PlanMilestone: Codable, Hashable {
    var title: String
    var goal: String?
}

struct PlanRevision: Codable, Hashable {
    var revision: Int
    var source: String
    var goal: String
    var acceptanceCriteria: String
    var allowedPathGlobs: [String]
    var milestones: [PlanMilestone]
    var configuredSuite: String?
    var resolvedSuite: String?
    var resolvedCheckIds: [String]
    var branch: String
}

struct PlanRequest: Codable, Identifiable, Hashable {
    var planRequestId: String
    var planId: String?
    var repository: String
    var repositoryRoot: String
    var branch: String
    var state: String
    var currentRevision: Int
    var lockedRevision: Int?
    var revisions: [PlanRevision]
    var availableSuites: [String]?
    var rejectionFeedback: String?
    var staleReasons: [String]?
    var supersededBy: String?
    var fulfilledSnapshotId: String?
    var createdAt: String?
    var updatedAt: String?

    var id: String { planRequestId }
    var current: PlanRevision? {
        revisions.first { $0.revision == currentRevision }
    }

    var suiteOptions: [String] {
        availableSuites ?? []
    }

    var isPendingApproval: Bool {
        state == "pending_approval"
    }

    var isStale: Bool {
        state == "stale" || !(staleReasons ?? []).isEmpty
    }

    var canApprove: Bool {
        isPendingApproval && !isStale
    }

    var canDismissFromQueue: Bool {
        state == "stale"
    }

    var statusTitle: String {
        if isStale {
            return "Stale"
        }
        switch state {
        case "pending_approval":
            return "Awaiting approval"
        case "locked":
            return "Locked"
        case "stale":
            return "Stale"
        case "rejected":
            return "Rejected"
        case "superseded":
            return "Superseded"
        case "fulfilled":
            return "Fulfilled"
        default:
            return state.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }

    var terminalActionMessage: String? {
        if isStale {
            return "Approval is disabled because the repository changed. Remove this stale plan or ask the agent to declare a fresh one."
        }
        switch state {
        case "pending_approval":
            return nil
        case "stale":
            return "Approval is disabled because the repository changed. Remove this stale plan or ask the agent to declare a fresh one."
        case "locked":
            return "This plan is locked. The agent can continue with implementation."
        case "rejected":
            return "This plan was rejected. The agent needs to declare a revised plan."
        case "superseded":
            return "This plan was superseded by a newer request."
        case "fulfilled":
            return "This plan has already been fulfilled by a snapshot."
        default:
            return "This plan is no longer waiting for approval."
        }
    }
}

struct PlanProposal: Codable, Equatable {
    var goal: String
    var acceptanceCriteria: String
    var allowedPathGlobs: [String]
    var milestones: [PlanMilestone]
    var requiredSuite: String?

    init(goal: String, acceptanceCriteria: String, allowedPathGlobs: [String], milestones: [PlanMilestone], requiredSuite: String?) {
        self.goal = goal
        self.acceptanceCriteria = acceptanceCriteria
        self.allowedPathGlobs = allowedPathGlobs
        self.milestones = milestones
        self.requiredSuite = requiredSuite
    }

    init(revision: PlanRevision) {
        goal = revision.goal
        acceptanceCriteria = revision.acceptanceCriteria
        allowedPathGlobs = revision.allowedPathGlobs
        milestones = revision.milestones
        requiredSuite = revision.configuredSuite
    }

    var validationMessage: String? {
        if goal.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return "Goal is required." }
        if acceptanceCriteria.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return "Acceptance criteria are required." }
        if allowedPathGlobs.isEmpty || allowedPathGlobs.contains(where: { $0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }) {
            return "At least one allowed path glob is required."
        }
        return nil
    }
}

struct ApprovalBody: Encodable {
    var expectedRevision: Int
    var proposal: PlanProposal?
}

struct RejectionBody: Encodable {
    var expectedRevision: Int
    var feedback: String
}

struct DismissalBody: Encodable {
    var expectedRevision: Int
}

struct APIErrorPayload: Decodable {
    var error: String
}

enum QueueState: Equatable {
    case loading
    case online
    case offline(String)
}

struct QueueNotice: Equatable {
    var requestID: PlanRequest.ID
    var state: String
    var message: String
}

func rejectionValidationMessage(_ feedback: String) -> String? {
    feedback.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Rejection feedback is required." : nil
}

func preferredPlanSelection(currentID: PlanRequest.ID?, requests: [PlanRequest]) -> PlanRequest.ID? {
    if let currentID,
       let current = requests.first(where: { $0.id == currentID }),
       current.canApprove {
        return currentID
    }
    return requests.first(where: \.canApprove)?.id ?? requests.first?.id
}

func orderedPlanRequests(_ requests: [PlanRequest]) -> [PlanRequest] {
    requests.sorted { left, right in
        let leftRank = left.canApprove ? 0 : 1
        let rightRank = right.canApprove ? 0 : 1
        if leftRank != rightRank { return leftRank < rightRank }
        if left.repository != right.repository { return left.repository < right.repository }
        if left.branch != right.branch { return left.branch < right.branch }
        return left.id < right.id
    }
}

func collapsedDuplicatePlanRequests(_ requests: [PlanRequest]) -> [PlanRequest] {
    var collapsedByKey: [String: PlanRequest] = [:]
    var keyOrder: [String] = []
    var result: [PlanRequest] = []

    for request in requests {
        guard let key = duplicatePlanKey(for: request) else {
            result.append(request)
            continue
        }
        if let existing = collapsedByKey[key] {
            collapsedByKey[key] = preferredDuplicatePlanRequest(existing, request)
        } else {
            collapsedByKey[key] = request
            keyOrder.append(key)
        }
    }

    result.append(contentsOf: keyOrder.compactMap { collapsedByKey[$0] })
    return result
}

private func duplicatePlanKey(for request: PlanRequest) -> String? {
    guard let revision = request.current else {
        return nil
    }
    let milestones = revision.milestones
        .map { "\($0.title)\u{1F}\($0.goal ?? "")" }
        .joined(separator: "\u{1E}")
    return [
        request.repositoryRoot,
        request.branch,
        revision.goal,
        revision.acceptanceCriteria,
        revision.allowedPathGlobs.joined(separator: "\u{1E}"),
        milestones,
        revision.configuredSuite ?? ""
    ].joined(separator: "\u{1D}")
}

private func preferredDuplicatePlanRequest(_ left: PlanRequest, _ right: PlanRequest) -> PlanRequest {
    if left.canApprove != right.canApprove {
        return left.canApprove ? left : right
    }
    let leftRecency = left.updatedAt ?? left.createdAt ?? ""
    let rightRecency = right.updatedAt ?? right.createdAt ?? ""
    if leftRecency != rightRecency {
        return leftRecency > rightRecency ? left : right
    }
    return left.id > right.id ? left : right
}

func newPendingPlanIDs(
    previous: Set<PlanRequest.ID>,
    requests: [PlanRequest]
) -> Set<PlanRequest.ID> {
    Set(
        requests
            .filter { $0.canApprove && !previous.contains($0.id) }
            .map(\.id)
    )
}

func didReviewQueueEmptyAfterPendingRequests(previousPendingCount: Int, requests: [PlanRequest]) -> Bool {
    previousPendingCount > 0 && requests.isEmpty
}

func menuBarSystemImageName(pendingCount: Int) -> String {
    pendingCount > 0 ? "checkmark.shield.fill" : "checkmark.shield"
}
