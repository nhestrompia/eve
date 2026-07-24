import AVFoundation
import Combine
import Foundation
import Speech

protocol VoiceSpeechRecognizing: AnyObject {
    var supportsOnDeviceRecognition: Bool { get }
    var isAvailable: Bool { get }

    func recognitionTask(
        with request: SFSpeechRecognitionRequest,
        resultHandler: @escaping (SFSpeechRecognitionResult?, Error?) -> Void
    ) -> SFSpeechRecognitionTask
}

extension SFSpeechRecognizer: VoiceSpeechRecognizing {}

@MainActor
protocol VoicePermissionAuthorizing {
    func speechRecognitionIsAuthorized() async -> Bool
    func microphoneIsAuthorized() async -> Bool
}

struct SystemVoicePermissionAuthorizer: VoicePermissionAuthorizing {
    func speechRecognitionIsAuthorized() async -> Bool {
        let current = SFSpeechRecognizer.authorizationStatus()
        if current != .notDetermined {
            return current == .authorized
        }
        return await withCheckedContinuation { continuation in
            SFSpeechRecognizer.requestAuthorization { status in
                continuation.resume(returning: status == .authorized)
            }
        }
    }

    func microphoneIsAuthorized() async -> Bool {
        switch AVCaptureDevice.authorizationStatus(for: .audio) {
        case .authorized:
            return true
        case .notDetermined:
            return await withCheckedContinuation { continuation in
                AVCaptureDevice.requestAccess(for: .audio) { granted in
                    continuation.resume(returning: granted)
                }
            }
        case .denied, .restricted:
            return false
        @unknown default:
            return false
        }
    }
}

enum VoiceTranscriptionState: Equatable {
    case idle
    case requestingPermission
    case recording
    case transcribing
    case unavailable(String)
    case failed(String)

    var presentation: VoiceTranscriptionPresentation {
        switch self {
        case .idle:
            return VoiceTranscriptionPresentation(
                statusMessage: "Speech is transcribed on this Mac.",
                actionTitle: "Dictate feedback",
                isBusy: false,
                isRecording: false,
                tone: .secondary
            )
        case .requestingPermission:
            return VoiceTranscriptionPresentation(
                statusMessage: "Waiting for Speech and microphone access…",
                actionTitle: "Cancel dictation",
                isBusy: true,
                isRecording: false,
                tone: .secondary
            )
        case .recording:
            return VoiceTranscriptionPresentation(
                statusMessage: "Listening on this Mac…",
                actionTitle: "Stop dictation",
                isBusy: true,
                isRecording: true,
                tone: .primary
            )
        case .transcribing:
            return VoiceTranscriptionPresentation(
                statusMessage: "Finishing the on-device transcription…",
                actionTitle: "Cancel dictation",
                isBusy: true,
                isRecording: false,
                tone: .secondary
            )
        case let .unavailable(message), let .failed(message):
            return VoiceTranscriptionPresentation(
                statusMessage: message,
                actionTitle: "Dictate feedback",
                isBusy: false,
                isRecording: false,
                tone: .error
            )
        }
    }

    var isRecording: Bool {
        presentation.isRecording
    }

    var isBusy: Bool {
        presentation.isBusy
    }

    var blocksFeedbackEditing: Bool {
        presentation.isBusy
    }

    var statusMessage: String {
        presentation.statusMessage
    }

    var actionTitle: String {
        presentation.actionTitle
    }

    var actionSymbol: String {
        presentation.actionSymbol
    }
}

struct VoiceTranscriptionPresentation: Equatable {
    enum Tone: Equatable {
        case secondary
        case primary
        case error
    }

    let statusMessage: String
    let actionTitle: String
    let isBusy: Bool
    let isRecording: Bool
    let tone: Tone

    var actionSymbol: String {
        isRecording ? "stop.fill" : "mic.fill"
    }
}

struct FeedbackDictationComposer {
    private let existingText: String

    init(existingText: String) {
        self.existingText = existingText
    }

    func text(appending transcript: String) -> String {
        guard !transcript.isEmpty else { return existingText }
        guard !existingText.isEmpty else { return transcript }
        let separator = existingText.last?.isWhitespace == true ? "" : " "
        return existingText + separator + transcript
    }
}

@MainActor
final class VoiceFeedbackTranscriber: ObservableObject {
    @Published private(set) var state: VoiceTranscriptionState = .idle

    private let locale: Locale
    private let audioEngine: AVAudioEngine
    private let recognizerFactory: (Locale) -> VoiceSpeechRecognizing?
    private let permissionAuthorizer: VoicePermissionAuthorizing
    private var speechRecognizer: VoiceSpeechRecognizing?
    private var recognitionRequest: SFSpeechAudioBufferRecognitionRequest?
    private var recognitionTask: SFSpeechRecognitionTask?
    private var composer: FeedbackDictationComposer?
    private var onTranscript: ((String) -> Void)?
    private var inputTapInstalled = false
    private var activeSessionID: UUID?

    init(
        locale: Locale = .current,
        audioEngine: AVAudioEngine = AVAudioEngine(),
        recognizerFactory: @escaping (Locale) -> VoiceSpeechRecognizing? = {
            SFSpeechRecognizer(locale: $0)
        },
        permissionAuthorizer: VoicePermissionAuthorizing? = nil
    ) {
        self.locale = locale
        self.audioEngine = audioEngine
        self.recognizerFactory = recognizerFactory
        self.permissionAuthorizer = permissionAuthorizer ?? SystemVoicePermissionAuthorizer()
    }

    deinit {
        recognitionTask?.cancel()
        if inputTapInstalled {
            audioEngine.inputNode.removeTap(onBus: 0)
        }
        audioEngine.stop()
    }

    func start(existingText: String, onTranscript: @escaping (String) -> Void) async {
        guard !state.isBusy else { return }
        cancel()
        let sessionID = UUID()
        activeSessionID = sessionID

        guard let recognizer = recognizerFactory(locale),
              recognizer.supportsOnDeviceRecognition else {
            becomeUnavailable(
                "On-device transcription isn’t available for \(locale.localizedString(forIdentifier: locale.identifier) ?? locale.identifier). Add a supported Dictation language in System Settings."
            )
            return
        }

        state = .requestingPermission

        let speechAuthorized = await permissionAuthorizer.speechRecognitionIsAuthorized()
        guard activeSessionID == sessionID else { return }
        guard speechAuthorized else {
            becomeUnavailable(
                "Speech Recognition access is off. Enable EVE in System Settings › Privacy & Security › Speech Recognition."
            )
            return
        }

        let microphoneAuthorized = await permissionAuthorizer.microphoneIsAuthorized()
        guard activeSessionID == sessionID else { return }
        guard microphoneAuthorized else {
            becomeUnavailable(
                "Microphone access is off. Enable EVE in System Settings › Privacy & Security › Microphone."
            )
            return
        }

        guard recognizer.isAvailable else {
            becomeUnavailable(
                "The on-device recognizer is temporarily unavailable. Try again in a moment."
            )
            return
        }

        do {
            try beginRecognition(
                with: recognizer,
                sessionID: sessionID,
                existingText: existingText,
                onTranscript: onTranscript
            )
        } catch {
            fail("EVE couldn’t start the microphone: \(error.localizedDescription)")
        }
    }

    func stop() {
        guard state == .recording else { return }
        stopAudioCapture()
        state = .transcribing
        recognitionRequest?.endAudio()
    }

    func cancel() {
        transition(to: .idle, cancellingTask: true)
    }

    private func beginRecognition(
        with recognizer: VoiceSpeechRecognizing,
        sessionID: UUID,
        existingText: String,
        onTranscript: @escaping (String) -> Void
    ) throws {
        speechRecognizer = recognizer
        composer = FeedbackDictationComposer(existingText: existingText)
        self.onTranscript = onTranscript

        let request = makeOnDeviceRecognitionRequest()
        recognitionRequest = request

        recognitionTask = recognizer.recognitionTask(with: request) { [weak self] result, error in
            Task { @MainActor in
                self?.handleRecognition(
                    result: result,
                    error: error,
                    sessionID: sessionID
                )
            }
        }

        let inputNode = audioEngine.inputNode
        let format = inputNode.outputFormat(forBus: 0)
        guard format.sampleRate > 0, format.channelCount > 0 else {
            throw VoiceTranscriptionError.noAudioInput
        }
        inputNode.installTap(
            onBus: 0,
            bufferSize: 1_024,
            format: format
        ) { [weak request] buffer, _ in
            request?.append(buffer)
        }
        inputTapInstalled = true

        audioEngine.prepare()
        try audioEngine.start()
        state = .recording
    }

    private func handleRecognition(
        result: SFSpeechRecognitionResult?,
        error: Error?,
        sessionID: UUID
    ) {
        guard activeSessionID == sessionID else { return }

        if let result, let composer {
            onTranscript?(composer.text(appending: result.bestTranscription.formattedString))
            if result.isFinal {
                finish()
                return
            }
        }

        if let error {
            fail("On-device transcription stopped: \(error.localizedDescription)")
        }
    }

    private func finish() {
        transition(to: .idle, cancellingTask: false)
    }

    private func fail(_ message: String) {
        transition(to: .failed(message), cancellingTask: true)
    }

    private func becomeUnavailable(_ message: String) {
        transition(to: .unavailable(message), cancellingTask: true)
    }

    private func transition(
        to nextState: VoiceTranscriptionState,
        cancellingTask: Bool
    ) {
        activeSessionID = nil
        if cancellingTask {
            recognitionTask?.cancel()
        }
        stopAudioCapture()
        clearRecognition()
        state = nextState
    }

    private func stopAudioCapture() {
        if audioEngine.isRunning {
            audioEngine.stop()
        }
        if inputTapInstalled {
            audioEngine.inputNode.removeTap(onBus: 0)
            inputTapInstalled = false
        }
    }

    private func clearRecognition() {
        speechRecognizer = nil
        recognitionTask = nil
        recognitionRequest = nil
        composer = nil
        onTranscript = nil
    }
}

func makeOnDeviceRecognitionRequest() -> SFSpeechAudioBufferRecognitionRequest {
    let request = SFSpeechAudioBufferRecognitionRequest()
    request.requiresOnDeviceRecognition = true
    request.shouldReportPartialResults = true
    request.addsPunctuation = true
    request.taskHint = .dictation
    request.contextualStrings = [
        "approve", "reject", "repository", "scope", "milestone", "verification"
    ]
    return request
}

private enum VoiceTranscriptionError: LocalizedError {
    case noAudioInput

    var errorDescription: String? {
        "No microphone input is available."
    }
}
