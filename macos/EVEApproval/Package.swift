// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "EVEApproval",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "EVEApproval", targets: ["EVEApproval"])
    ],
    targets: [
        .executableTarget(name: "EVEApproval"),
        .testTarget(name: "EVEApprovalTests", dependencies: ["EVEApproval"])
    ]
)
