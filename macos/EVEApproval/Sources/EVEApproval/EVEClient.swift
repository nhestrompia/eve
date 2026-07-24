import Foundation

struct EVEClient {
    var baseURL = URL(string: "http://127.0.0.1:4317")!
    var session: URLSession = .shared

    func health() async -> Bool {
        guard let url = URL(string: "/api/health", relativeTo: baseURL) else { return false }
        do {
            let (_, response) = try await session.data(from: url)
            return (response as? HTTPURLResponse)?.statusCode == 200
        } catch {
            return false
        }
    }

    func reviewQueue() async throws -> [PlanRequest] {
        let requests: [PlanRequest] = try await request(path: "/api/plan-requests")
        return requests.filter { $0.state == "pending_approval" || $0.state == "stale" }
    }

    func planRequest(id: String) async throws -> PlanRequest {
        try await request(path: "/api/plan-requests/\(id)")
    }

    func approve(_ request: PlanRequest, proposal: PlanProposal?) async throws -> PlanRequest {
        try await self.request(
            path: "/api/plan-requests/\(request.planRequestId)/approve",
            method: "POST",
            body: ApprovalBody(expectedRevision: request.currentRevision, proposal: proposal)
        )
    }

    func reject(_ request: PlanRequest, feedback: String) async throws -> PlanRequest {
        try await self.request(
            path: "/api/plan-requests/\(request.planRequestId)/reject",
            method: "POST",
            body: RejectionBody(expectedRevision: request.currentRevision, feedback: feedback)
        )
    }

    private func request<Response: Decodable>(path: String) async throws -> Response {
        guard let url = URL(string: path, relativeTo: baseURL) else { throw ClientError.invalidURL }
        let (data, response) = try await session.data(from: url)
        return try decode(data: data, response: response)
    }

    private func request<Body: Encodable, Response: Decodable>(
        path: String,
        method: String,
        body: Body
    ) async throws -> Response {
        guard let url = URL(string: path, relativeTo: baseURL) else { throw ClientError.invalidURL }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await session.data(for: request)
        return try decode(data: data, response: response)
    }

    private func decode<Response: Decodable>(data: Data, response: URLResponse) throws -> Response {
        guard let http = response as? HTTPURLResponse else { throw ClientError.invalidResponse }
        guard (200..<300).contains(http.statusCode) else {
            let message = (try? JSONDecoder().decode(APIErrorPayload.self, from: data).error) ?? "EVE returned HTTP \(http.statusCode)."
            throw ClientError.server(message)
        }
        return try JSONDecoder().decode(Response.self, from: data)
    }
}

enum ClientError: LocalizedError {
    case invalidURL
    case invalidResponse
    case server(String)

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "The EVE server address is invalid."
        case .invalidResponse: return "EVE returned an invalid response."
        case let .server(message): return message
        }
    }
}

enum DaemonLauncher {
    static func startIfNeeded(client: EVEClient) async throws {
        if await client.health() { return }
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/env")
        process.arguments = ["eve", "daemon", "--addr", "127.0.0.1:4317"]
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice
        do {
            try process.run()
        } catch {
            throw ClientError.server("EVE could not be started. Install the eve binary and ensure port 4317 is available.")
        }
        for _ in 0..<20 {
            try await Task.sleep(nanoseconds: 150_000_000)
            if await client.health() { return }
        }
        if process.isRunning { process.terminate() }
        throw ClientError.server("EVE did not become healthy on 127.0.0.1:4317. Check for a port conflict or run `eve daemon` in Terminal.")
    }
}
