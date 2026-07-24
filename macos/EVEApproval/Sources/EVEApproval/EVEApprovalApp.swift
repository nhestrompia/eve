import AppKit
import Combine
import SwiftUI

@main
struct EVEApprovalApp: App {
    @StateObject private var coordinator = ApprovalAppCoordinator()

    init() {
        NSApplication.shared.setActivationPolicy(.accessory)
    }

    var body: some Scene {
        MenuBarExtra {
            PlanQueueView(store: coordinator.store)
                .frame(width: 760, height: 680)
        } label: {
            Label(
                "EVE Plans",
                systemImage: coordinator.store.pendingCount == 0
                    ? "checkmark.shield"
                    : "checkmark.shield.fill"
            )
            .accessibilityLabel("\(coordinator.store.pendingCount) EVE plans pending approval")
        }
        .menuBarExtraStyle(.window)
    }
}

@MainActor
private final class ApprovalAppCoordinator: ObservableObject {
    let store: PlanQueueStore
    private let attentionController = PlanAttentionController()
    private var storeObservation: AnyCancellable?

    init() {
        let store = PlanQueueStore()
        self.store = store
        storeObservation = store.objectWillChange.sink { [weak self] in
            self?.objectWillChange.send()
        }
        store.onNewPendingRequests = { [weak self, weak store] requests in
            guard let self, let store else { return }
            self.attentionController.show(store: store, newRequests: requests)
        }
        store.start()
    }
}

@MainActor
private final class PlanAttentionController {
    private var panel: NSPanel?

    func show(store: PlanQueueStore, newRequests: [PlanRequest]) {
        if panel == nil {
            let panel = NSPanel(
                contentRect: NSRect(x: 0, y: 0, width: 820, height: 720),
                styleMask: [.titled, .closable, .resizable, .fullSizeContentView],
                backing: .buffered,
                defer: false
            )
            panel.title = "EVE Plan Approvals"
            panel.titlebarAppearsTransparent = true
            panel.isFloatingPanel = true
            panel.hidesOnDeactivate = false
            panel.isReleasedWhenClosed = false
            panel.minSize = NSSize(width: 680, height: 560)
            panel.contentViewController = NSHostingController(rootView: PlanQueueView(store: store))
            self.panel = panel
        }

        panel?.title = newRequests.count == 1
            ? "Plan approval · \(newRequests[0].repository)"
            : "\(newRequests.count) new plans awaiting approval"
        NSApplication.shared.activate(ignoringOtherApps: true)
        NSApplication.shared.requestUserAttention(.informationalRequest)
        panel?.center()
        panel?.makeKeyAndOrderFront(nil)
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
        HStack(spacing: 12) {
            Image(systemName: "checkmark.shield.fill")
                .font(.title2)
                .foregroundStyle(.tint)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 2) {
                Text("Plan approvals").font(.headline)
                Text(queueSummary).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            if store.pendingCount > 0 {
                Text("\(store.pendingCount) waiting")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color.accentColor)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(Color.accentColor.opacity(0.1), in: Capsule())
                    .accessibilityLabel("\(store.pendingCount) plans waiting for approval")
            }
            Button {
                Task { await store.refresh() }
            } label: {
                Image(systemName: "arrow.clockwise")
            }
            .buttonStyle(.borderless)
            .help("Refresh plans")
            .accessibilityLabel("Refresh pending plans")
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 14)
    }

    private var queueSummary: String {
        if store.requests.isEmpty { return "Watching every known repository" }
        return "\(store.requests.count) \(store.requests.count == 1 ? "request" : "requests") across your repositories"
    }

    @ViewBuilder
    private var content: some View {
        switch store.state {
        case .loading:
            ProgressView("Connecting to EVE…")
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case let .offline(message):
            QueueEmptyState(
                title: "EVE is offline",
                systemImage: "bolt.horizontal.circle",
                message: message
            )
        case .online:
            if store.requests.isEmpty {
                QueueEmptyState(
                    title: "No plans waiting",
                    systemImage: "checkmark.circle",
                    message: "EVE is watching in the background. A review window will open when an agent declares a plan."
                )
            } else {
                HStack(spacing: 0) {
                    PlanQueueSidebar(store: store)
                        .frame(width: 228)
                    Divider()
                    if let request = store.selected {
                        PlanReviewView(request: request, store: store)
                            .id(request.id)
                    }
                }
            }
        }
    }
}

private struct PlanQueueSidebar: View {
    @ObservedObject var store: PlanQueueStore

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("WAITING QUEUE")
                .font(.caption2.weight(.semibold))
                .tracking(0.8)
                .foregroundStyle(.secondary)
                .padding(.horizontal, 14)
                .padding(.vertical, 12)
            ScrollView {
                LazyVStack(spacing: 6) {
                    ForEach(store.requests) { request in
                        Button {
                            store.selectedID = request.id
                        } label: {
                            QueueRow(request: request, selected: store.selectedID == request.id)
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("\(request.repository), \(request.current?.goal ?? "Plan request")")
                        .accessibilityValue(request.isPendingApproval ? "Awaiting approval" : request.state)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.bottom, 12)
            }
        }
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.55))
    }
}

private struct QueueRow: View {
    let request: PlanRequest
    let selected: Bool

    var body: some View {
        HStack(alignment: .top, spacing: 9) {
            Circle()
                .fill(request.isPendingApproval ? Color.accentColor : Color.orange)
                .frame(width: 7, height: 7)
                .padding(.top, 5)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 4) {
                Text(request.repository)
                    .font(.callout.weight(.semibold))
                    .lineLimit(1)
                Text(request.current?.goal ?? "Plan request")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
                Text(request.branch)
                    .font(.caption2.monospaced())
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
            }
            Spacer(minLength: 0)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            selected ? Color.accentColor.opacity(0.12) : Color.clear,
            in: RoundedRectangle(cornerRadius: 9)
        )
        .contentShape(RoundedRectangle(cornerRadius: 9))
    }
}

private struct QueueEmptyState: View {
    let title: String
    let systemImage: String
    let message: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 32))
                .foregroundStyle(.secondary)
            Text(title).font(.headline)
            Text(message)
                .multilineTextAlignment(.center)
                .foregroundStyle(.secondary)
                .frame(maxWidth: 360)
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
        _proposal = State(
            initialValue: request.current.map(PlanProposal.init)
                ?? PlanProposal(
                    goal: "",
                    acceptanceCriteria: "",
                    allowedPathGlobs: [],
                    milestones: [],
                    requiredSuite: nil
                )
        )
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
                    planContent
                    if rejecting {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Why are you rejecting this plan?")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(.secondary)
                            TextField("Required feedback for the agent", text: $feedback, axis: .vertical)
                                .lineLimit(3...6)
                                .textFieldStyle(.roundedBorder)
                                .accessibilityLabel("Rejection feedback")
                            if let message = rejectionValidationMessage(feedback) {
                                Text(message).font(.caption).foregroundStyle(.red)
                            }
                        }
                    }
                }
                .padding(20)
            }
            Divider()
            actionBar
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var repositoryHeader: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 5) {
                Text(request.repository)
                    .font(.title2.weight(.semibold))
                HStack(spacing: 7) {
                    Label(request.branch, systemImage: "arrow.triangle.branch")
                    Text("·")
                    Text("Revision \(request.currentRevision)")
                }
                .font(.caption.monospaced())
                .foregroundStyle(.secondary)
            }
            Spacer()
            Text(request.isPendingApproval ? "Awaiting approval" : request.state.replacingOccurrences(of: "_", with: " ").capitalized)
                .font(.caption.weight(.semibold))
                .foregroundStyle(request.isPendingApproval ? Color.accentColor : Color.orange)
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .background(
                    (request.isPendingApproval ? Color.accentColor : Color.orange).opacity(0.1),
                    in: Capsule()
                )
        }
    }

    @ViewBuilder
    private var planContent: some View {
        if let revision = request.current {
            VStack(alignment: .leading, spacing: 18) {
                InlinePlanField(title: "Goal", editing: editing) {
                    if editing {
                        TextField("Goal", text: $proposal.goal, axis: .vertical)
                            .lineLimit(2...5)
                            .textFieldStyle(.roundedBorder)
                    } else {
                        Text(revision.goal).textSelection(.enabled)
                    }
                }

                InlinePlanField(title: "Acceptance criteria", editing: editing) {
                    if editing {
                        TextField("Acceptance criteria", text: $proposal.acceptanceCriteria, axis: .vertical)
                            .lineLimit(4...10)
                            .textFieldStyle(.roundedBorder)
                    } else {
                        Text(revision.acceptanceCriteria)
                            .textSelection(.enabled)
                    }
                }

                InlinePlanField(title: "Declared scope", editing: editing) {
                    if editing {
                        TextField(
                            "Allowed path globs (one per line)",
                            text: Binding(
                                get: { proposal.allowedPathGlobs.joined(separator: "\n") },
                                set: {
                                    proposal.allowedPathGlobs = $0
                                        .split(separator: "\n", omittingEmptySubsequences: false)
                                        .map(String.init)
                                }
                            ),
                            axis: .vertical
                        )
                        .lineLimit(3...8)
                        .font(.callout.monospaced())
                        .textFieldStyle(.roundedBorder)
                    } else {
                        ValueStack(values: revision.allowedPathGlobs, monospaced: true)
                    }
                }

                InlinePlanField(title: "Verification suite", editing: editing) {
                    if editing {
                        Picker(
                            "Verification suite",
                            selection: Binding(
                                get: { proposal.requiredSuite ?? "" },
                                set: { proposal.requiredSuite = $0.isEmpty ? nil : $0 }
                            )
                        ) {
                            Text("Use branch default").tag("")
                            ForEach(request.suiteOptions, id: \.self) { suite in
                                Text(suite).tag(suite)
                            }
                        }
                        .pickerStyle(.menu)
                        .labelsHidden()
                        .accessibilityLabel("Verification suite")
                    } else {
                        Text(revision.configuredSuite ?? "Branch default")
                    }
                }

                InlinePlanField(title: "Milestones", editing: false) {
                    ValueStack(
                        values: revision.milestones.isEmpty
                            ? ["No milestones"]
                            : revision.milestones.map { milestone in
                                milestone.goal.map { "\(milestone.title) — \($0)" } ?? milestone.title
                            }
                    )
                }

                InlinePlanField(title: "Resolved checks", editing: false) {
                    ValueStack(
                        values: revision.resolvedCheckIds.isEmpty
                            ? ["No configured checks"]
                            : revision.resolvedCheckIds,
                        monospaced: true
                    )
                }

                if editing, let message = proposal.validationMessage {
                    Label(message, systemImage: "exclamationmark.circle.fill")
                        .font(.caption)
                        .foregroundStyle(.red)
                }
            }
        }
    }

    private var actionBar: some View {
        HStack(spacing: 10) {
            if rejecting {
                Button("Cancel") {
                    rejecting = false
                    feedback = ""
                }
                Spacer()
                Button("Send rejection") {
                    Task { await store.reject(request, feedback: feedback) }
                }
                .buttonStyle(.borderedProminent)
                .tint(.red)
                .disabled(rejectionValidationMessage(feedback) != nil)
            } else {
                Button(editing ? "Cancel edits" : "Edit inline") {
                    editing.toggle()
                    if !editing, let revision = request.current {
                        proposal = PlanProposal(revision: revision)
                    }
                }
                Button("Reject") { rejecting = true }
                    .keyboardShortcut(.delete, modifiers: [.command])
                Spacer()
                Button(editing ? "Approve edited revision" : "Approve plan") {
                    Task { await store.approve(request, proposal: editing ? proposal : nil) }
                }
                .keyboardShortcut(.return, modifiers: [.command])
                .buttonStyle(.borderedProminent)
                .disabled(
                    !request.isPendingApproval
                        || (editing && proposal.validationMessage != nil)
                )
            }
        }
        .controlSize(.large)
        .padding(.horizontal, 20)
        .padding(.vertical, 14)
        .background(.bar)
    }
}

private struct InlinePlanField<Content: View>: View {
    let title: String
    let editing: Bool
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack(spacing: 6) {
                Text(title)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                if editing {
                    Text("EDITING")
                        .font(.system(size: 9, weight: .bold))
                        .tracking(0.6)
                        .foregroundStyle(.tint)
                }
            }
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct ValueStack: View {
    let values: [String]
    var monospaced = false

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            ForEach(Array(values.enumerated()), id: \.offset) { _, value in
                Text(value)
                    .font(monospaced ? .callout.monospaced() : .callout)
                    .foregroundStyle(value.hasPrefix("No ") ? .secondary : .primary)
                    .textSelection(.enabled)
            }
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
        .background(color.opacity(0.1), in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .stroke(color.opacity(0.2), lineWidth: 1)
        }
    }
}
