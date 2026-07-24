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

    var id: String { planRequestId }
    var current: PlanRevision? {
        revisions.first { $0.revision == currentRevision }
    }

    var suiteOptions: [String] {
        availableSuites ?? []
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

struct APIErrorPayload: Decodable {
    var error: String
}

enum QueueState: Equatable {
    case loading
    case online
    case offline(String)
}

func rejectionValidationMessage(_ feedback: String) -> String? {
    feedback.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Rejection feedback is required." : nil
}

func preferredPlanSelection(currentID: PlanRequest.ID?, requests: [PlanRequest]) -> PlanRequest.ID? {
    if let currentID,
       let current = requests.first(where: { $0.id == currentID }),
       current.state == "pending_approval" {
        return currentID
    }
    return requests.first(where: { $0.state == "pending_approval" })?.id ?? requests.first?.id
}
