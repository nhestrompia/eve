import XCTest
import Speech
@testable import EVEApproval

@MainActor
final class VoiceFeedbackTranscriberTests: XCTestCase {
    func testDictationComposerPreservesExistingFeedbackAcrossPartialResults() {
        let composer = FeedbackDictationComposer(existingText: "Please")

        XCTAssertEqual(composer.text(appending: "narrow"), "Please narrow")
        XCTAssertEqual(composer.text(appending: "narrow the scope."), "Please narrow the scope.")
    }

    func testDictationComposerRespectsExistingWhitespaceAndEmptyTranscripts() {
        XCTAssertEqual(
            FeedbackDictationComposer(existingText: "Keep this:\n").text(appending: "Add a test."),
            "Keep this:\nAdd a test."
        )
        XCTAssertEqual(
            FeedbackDictationComposer(existingText: "Typed feedback").text(appending: ""),
            "Typed feedback"
        )
        XCTAssertEqual(
            FeedbackDictationComposer(existingText: "").text(appending: "Voice only."),
            "Voice only."
        )
    }

    func testVoiceTranscriptionStatesExplainOnDeviceProgress() {
        XCTAssertEqual(VoiceTranscriptionState.idle.statusMessage, "Speech is transcribed on this Mac.")
        XCTAssertEqual(VoiceTranscriptionState.recording.actionTitle, "Stop dictation")
        XCTAssertEqual(VoiceTranscriptionState.transcribing.actionTitle, "Cancel dictation")
        XCTAssertTrue(VoiceTranscriptionState.recording.blocksFeedbackEditing)
        XCTAssertTrue(VoiceTranscriptionState.transcribing.isBusy)
        XCTAssertFalse(VoiceTranscriptionState.failed("Try again.").isBusy)
    }

    func testRecognitionRequestAlwaysRequiresOnDevicePartialDictation() {
        let request = makeOnDeviceRecognitionRequest()

        XCTAssertTrue(request.requiresOnDeviceRecognition)
        XCTAssertTrue(request.shouldReportPartialResults)
        XCTAssertEqual(request.taskHint, .dictation)
    }

    func testPermissionsAreLazyAndRequestedInOrder() async {
        let permissions = RecordingPermissionAuthorizer(
            speechResult: true,
            microphoneResult: true
        )
        let transcriber = VoiceFeedbackTranscriber(
            locale: Locale(identifier: "en-US"),
            recognizerFactory: { _ in
                StubSpeechRecognizer(
                    supportsOnDeviceRecognition: true,
                    isAvailable: false
                )
            },
            permissionAuthorizer: permissions
        )

        XCTAssertTrue(permissions.calls.isEmpty)
        await transcriber.start(existingText: "") { _ in }

        XCTAssertEqual(permissions.calls, [.speech, .microphone])
        XCTAssertEqual(
            transcriber.state,
            .unavailable("The on-device recognizer is temporarily unavailable. Try again in a moment.")
        )
    }

    func testDeniedSpeechPermissionDoesNotRequestMicrophone() async {
        let permissions = RecordingPermissionAuthorizer(
            speechResult: false,
            microphoneResult: true
        )
        let transcriber = VoiceFeedbackTranscriber(
            locale: Locale(identifier: "en-US"),
            recognizerFactory: { _ in
                StubSpeechRecognizer(
                    supportsOnDeviceRecognition: true,
                    isAvailable: true
                )
            },
            permissionAuthorizer: permissions
        )

        await transcriber.start(existingText: "") { _ in }

        XCTAssertEqual(permissions.calls, [.speech])
    }

    func testCancelDuringPermissionRequestPreventsMicrophoneAndRecording() async {
        let permissions = SuspendedSpeechPermissionAuthorizer()
        let transcriber = VoiceFeedbackTranscriber(
            locale: Locale(identifier: "en-US"),
            recognizerFactory: { _ in
                StubSpeechRecognizer(
                    supportsOnDeviceRecognition: true,
                    isAvailable: true
                )
            },
            permissionAuthorizer: permissions
        )

        let startTask = Task {
            await transcriber.start(existingText: "Keep this") { _ in }
        }
        await permissions.waitUntilSpeechPermissionWasRequested()
        XCTAssertEqual(transcriber.state, .requestingPermission)

        transcriber.cancel()
        permissions.resolveSpeechPermission(granted: true)
        await startTask.value

        XCTAssertEqual(transcriber.state, .idle)
        XCTAssertEqual(permissions.microphoneRequestCount, 0)
    }
}

@MainActor
private final class RecordingPermissionAuthorizer: VoicePermissionAuthorizing {
    enum PermissionRequest: Equatable {
        case speech
        case microphone
    }

    private let speechResult: Bool
    private let microphoneResult: Bool
    private(set) var calls: [PermissionRequest] = []

    init(speechResult: Bool, microphoneResult: Bool) {
        self.speechResult = speechResult
        self.microphoneResult = microphoneResult
    }

    func speechRecognitionIsAuthorized() async -> Bool {
        calls.append(.speech)
        return speechResult
    }

    func microphoneIsAuthorized() async -> Bool {
        calls.append(.microphone)
        return microphoneResult
    }
}

@MainActor
private final class SuspendedSpeechPermissionAuthorizer: VoicePermissionAuthorizing {
    private var speechContinuation: CheckedContinuation<Bool, Never>?
    private var requestWaiters: [CheckedContinuation<Void, Never>] = []
    private(set) var microphoneRequestCount = 0

    func speechRecognitionIsAuthorized() async -> Bool {
        requestWaiters.forEach { $0.resume() }
        requestWaiters.removeAll()
        return await withCheckedContinuation { continuation in
            speechContinuation = continuation
        }
    }

    func microphoneIsAuthorized() async -> Bool {
        microphoneRequestCount += 1
        return true
    }

    func waitUntilSpeechPermissionWasRequested() async {
        if speechContinuation != nil { return }
        await withCheckedContinuation { continuation in
            requestWaiters.append(continuation)
        }
    }

    func resolveSpeechPermission(granted: Bool) {
        speechContinuation?.resume(returning: granted)
        speechContinuation = nil
    }
}

private final class StubSpeechRecognizer: VoiceSpeechRecognizing {
    let supportsOnDeviceRecognition: Bool
    let isAvailable: Bool

    init(supportsOnDeviceRecognition: Bool, isAvailable: Bool) {
        self.supportsOnDeviceRecognition = supportsOnDeviceRecognition
        self.isAvailable = isAvailable
    }

    func recognitionTask(
        with request: SFSpeechRecognitionRequest,
        resultHandler: @escaping (SFSpeechRecognitionResult?, Error?) -> Void
    ) -> SFSpeechRecognitionTask {
        fatalError("Recognition must not start in permission and availability tests.")
    }
}
