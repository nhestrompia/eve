import AppKit
import SwiftUI

@main
struct EVEApprovalApp: App {
    @StateObject private var store = PlanQueueStore()

    init() {
        NSApplication.shared.setActivationPolicy(.accessory)
    }

    var body: some Scene {
        MenuBarExtra {
            PlanQueueView(store: store)
                .frame(width: 460, height: 620)
                .task { store.start() }
        } label: {
            let pendingCount = store.requests.filter { $0.state == "pending_approval" }.count
            Label("EVE Plans", systemImage: pendingCount == 0 ? "checkmark.shield" : "checkmark.shield.fill")
                .accessibilityLabel("\(pendingCount) EVE plans pending approval")
        }
        .menuBarExtraStyle(.window)
    }
}

struct PlanQueueView: View {
    @ObservedObject var store: PlanQueueStore

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            content
        }
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private var header: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Plan approvals").font(.headline)
                Text("\(store.requests.filter { $0.state == "pending_approval" }.count) pending").font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            Button {
                Task { await store.refresh() }
            } label: {
                Image(systemName: "arrow.clockwise")
            }
            .buttonStyle(.borderless)
            .help("Refresh plans")
            .accessibilityLabel("Refresh pending plans")
        }
        .padding()
    }

    @ViewBuilder
    private var content: some View {
        switch store.state {
        case .loading:
            ProgressView("Connecting to EVE…").frame(maxWidth: .infinity, maxHeight: .infinity)
        case let .offline(message):
            QueueEmptyState(title: "EVE is offline", systemImage: "bolt.horizontal.circle", message: message)
        case .online:
            if let request = store.selected {
                PlanReviewView(request: request, store: store)
            } else {
                QueueEmptyState(title: "No plans waiting", systemImage: "checkmark.circle", message: "New declarations will appear here automatically.")
            }
        }
    }
}

private struct QueueEmptyState: View {
    let title: String
    let systemImage: String
    let message: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage).font(.system(size: 32)).foregroundStyle(.secondary)
            Text(title).font(.headline)
            Text(message).multilineTextAlignment(.center).foregroundStyle(.secondary)
        }
        .padding(32)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

struct PlanReviewView: View {
    let request: PlanRequest
    @ObservedObject var store: PlanQueueStore
    @State private var editing = false
    @State private var rejecting = false
    @State private var proposal: PlanProposal
    @State private var feedback = ""

    init(request: PlanRequest, store: PlanQueueStore) {
        self.request = request
        self.store = store
        _proposal = State(initialValue: request.current.map(PlanProposal.init) ?? PlanProposal(goal: "", acceptanceCriteria: "", allowedPathGlobs: [], milestones: [], requiredSuite: nil))
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    repositoryHeader
                    if let notice = store.notice {
                        MessagePanel(title: "Status", lines: [notice], color: .accentColor)
                    }
                    if let stale = request.staleReasons, !stale.isEmpty {
                        MessagePanel(title: "Approval disabled", lines: stale, color: .orange)
                    }
                    if editing {
                        editForm
                    } else {
                        planReadout
                    }
                    if rejecting {
                        TextField("Required rejection feedback", text: $feedback, axis: .vertical)
                            .lineLimit(3...6)
                            .textFieldStyle(.roundedBorder)
                            .accessibilityLabel("Rejection feedback")
                    }
                }
                .padding()
            }
            Divider()
            actionBar
        }
    }

    private var repositoryHeader: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(request.repository).font(.headline)
            Text(request.branch).font(.caption.monospaced()).foregroundStyle(.secondary)
        }
    }

    @ViewBuilder
    private var planReadout: some View {
        if let revision = request.current {
            Field(title: "Goal", text: revision.goal)
            Field(title: "Acceptance criteria", text: revision.acceptanceCriteria)
            ListField(title: "Declared scope", values: revision.allowedPathGlobs)
            ListField(title: "Milestones", values: revision.milestones.map { milestone in
                milestone.goal.map { "\(milestone.title) — \($0)" } ?? milestone.title
            })
            ListField(title: "Resolved checks", values: revision.resolvedCheckIds.isEmpty ? ["No configured checks"] : revision.resolvedCheckIds)
        }
    }

    private var editForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            TextField("Goal", text: $proposal.goal, axis: .vertical).lineLimit(2...5)
            TextField("Acceptance criteria", text: $proposal.acceptanceCriteria, axis: .vertical).lineLimit(4...10)
            TextField("Allowed path globs (one per line)", text: Binding(
                get: { proposal.allowedPathGlobs.joined(separator: "\n") },
                set: { proposal.allowedPathGlobs = $0.split(separator: "\n", omittingEmptySubsequences: false).map(String.init) }
            ), axis: .vertical).lineLimit(3...8)
            Picker("Verification suite", selection: Binding(
                get: { proposal.requiredSuite ?? "" },
                set: { proposal.requiredSuite = $0.isEmpty ? nil : $0 }
            )) {
                Text("Use branch default").tag("")
                ForEach(request.suiteOptions, id: \.self) { suite in
                    Text(suite).tag(suite)
                }
            }
            .pickerStyle(.menu)
            .accessibilityLabel("Verification suite")
            if let message = proposal.validationMessage {
                Text(message).font(.caption).foregroundStyle(.red)
            }
        }
        .textFieldStyle(.roundedBorder)
    }

    private var actionBar: some View {
        HStack {
            if rejecting {
                Button("Cancel") { rejecting = false; feedback = "" }
                Spacer()
                Button("Reject") {
                    Task { await store.reject(request, feedback: feedback) }
                }
                .buttonStyle(.borderedProminent)
                .tint(.red)
                .disabled(rejectionValidationMessage(feedback) != nil)
            } else {
                Button(editing ? "Cancel edits" : "Edit") {
                    editing.toggle()
                    if !editing, let revision = request.current { proposal = PlanProposal(revision: revision) }
                }
                Button("Reject") { rejecting = true }
                    .keyboardShortcut(.delete, modifiers: [.command])
                Spacer()
                Button(editing ? "Approve edited revision" : "Approve") {
                    Task { await store.approve(request, proposal: editing ? proposal : nil) }
                }
                .keyboardShortcut(.return, modifiers: [.command])
                .buttonStyle(.borderedProminent)
                .disabled(request.state != "pending_approval" || (editing && proposal.validationMessage != nil))
            }
        }
        .controlSize(.large)
        .padding()
    }
}

private struct Field: View {
    let title: String
    let text: String
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            Text(text).textSelection(.enabled)
        }
    }
}

private struct ListField: View {
    let title: String
    let values: [String]
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            ForEach(values, id: \.self) { Text($0).font(.callout.monospaced()) }
        }
    }
}

private struct MessagePanel: View {
    let title: String
    let lines: [String]
    let color: Color
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(title).font(.headline)
            ForEach(lines, id: \.self) { Text($0).font(.caption) }
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(color.opacity(0.12), in: RoundedRectangle(cornerRadius: 8))
    }
}
