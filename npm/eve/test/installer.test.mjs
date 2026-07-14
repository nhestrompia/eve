import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtemp, readFile, stat } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import {
  defaultInstallDir,
  downloadReleaseAsset,
  main,
  normalizeVersion,
  parseChecksums,
  parseInstallArgs,
  releaseAssetName,
  verifyReleaseAsset,
} from "../lib/installer.mjs";

test("parses installer options", () => {
  assert.deepEqual(parseInstallArgs(["install", "--clients", "codex,claude", "--install-dir=/tmp/eve", "--no-mcp"]), {
    clients: "codex,claude",
    help: false,
    installDir: "/tmp/eve",
    noMCP: true,
  });
  assert.equal(parseInstallArgs(["--help"]).help, true);
  assert.throws(() => parseInstallArgs(["init"]), /expected `install`/);
  assert.throws(() => parseInstallArgs(["install", "--clients"]), /requires a value/);
});

test("maps Node platforms to release artifacts", () => {
  assert.equal(releaseAssetName("darwin", "arm64"), "eve-darwin-arm64");
  assert.equal(releaseAssetName("linux", "x64"), "eve-linux-amd64");
  assert.equal(releaseAssetName("win32", "x64"), "eve-windows-amd64.exe");
  assert.throws(() => releaseAssetName("freebsd", "x64"), /unsupported platform/);
  assert.throws(() => releaseAssetName("linux", "ia32"), /unsupported platform/);
});

test("chooses user-owned install directories", () => {
  assert.equal(defaultInstallDir({ env: {}, platform: "darwin", home: "/Users/eve" }), path.join("/Users/eve", ".local", "bin"));
  assert.equal(defaultInstallDir({ env: { LOCALAPPDATA: "C:\\Users\\eve\\AppData\\Local" }, platform: "win32", home: "C:\\Users\\eve" }), path.join("C:\\Users\\eve\\AppData\\Local", "EVE", "bin"));
  assert.equal(defaultInstallDir({ env: { EVE_INSTALL_DIR: "/custom/bin" }, platform: "linux", home: "/home/eve" }), "/custom/bin");
});

test("normalizes published release versions", () => {
  assert.equal(normalizeVersion("v0.3.0"), "0.3.0");
  assert.equal(normalizeVersion("1.0.0-beta.1"), "1.0.0-beta.1");
  assert.throws(() => normalizeVersion("0.0.0-development"), /not tied to a published/);
});

test("parses and verifies release checksums", () => {
  const bytes = Buffer.from("eve binary");
  const digest = createHash("sha256").update(bytes).digest("hex");
  const checksums = `${digest}  eve-darwin-arm64\n${"0".repeat(64)} *eve-linux-amd64\n`;
  assert.equal(parseChecksums(checksums).get("eve-darwin-arm64"), digest);
  assert.doesNotThrow(() => verifyReleaseAsset("eve-darwin-arm64", bytes, checksums));
  assert.throws(() => verifyReleaseAsset("eve-darwin-arm64", Buffer.from("tampered"), checksums), /checksum mismatch/);
  assert.throws(() => verifyReleaseAsset("eve-windows-amd64.exe", bytes, checksums), /missing from SHA256SUMS/);
});

test("downloads the binary and checksum manifest from one release", async () => {
  const bytes = Buffer.from("release binary");
  const digest = createHash("sha256").update(bytes).digest("hex");
  const responses = new Map([
    ["https://releases.example/v1/SHA256SUMS", Buffer.from(`${digest}  eve-linux-amd64\n`)],
    ["https://releases.example/v1/eve-linux-amd64", bytes],
  ]);
  const fetchImpl = async (url) => ({
    ok: responses.has(url),
    status: responses.has(url) ? 200 : 404,
    statusText: responses.has(url) ? "OK" : "Not Found",
    arrayBuffer: async () => responses.get(url),
  });
  const result = await downloadReleaseAsset({
    version: "1.0.0",
    platform: "linux",
    arch: "x64",
    baseUrl: "https://releases.example/v1/",
    fetchImpl,
  });
  assert.equal(result.assetName, "eve-linux-amd64");
  assert.deepEqual(result.bytes, bytes);
});

test("installs, verifies, and configures a downloaded release", async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), "eve-npx-test-"));
  const installDir = path.join(root, "bin");
  const bytes = Buffer.from("fake eve binary");
  const digest = createHash("sha256").update(bytes).digest("hex");
  const responses = new Map([
    ["https://releases.example/v1/SHA256SUMS", Buffer.from(`${digest}  eve-linux-amd64\n`)],
    ["https://releases.example/v1/eve-linux-amd64", bytes],
  ]);
  const fetchImpl = async (url) => ({
    ok: responses.has(url),
    status: responses.has(url) ? 200 : 404,
    statusText: responses.has(url) ? "OK" : "Not Found",
    arrayBuffer: async () => responses.get(url),
  });
  const calls = [];
  const spawn = (command, args) => {
    calls.push({ command, args });
    if (args[0] === "version") {
      return { status: 0, stdout: "eve 1.2.3 (snapshot schema 0.1.0)\n" };
    }
    return { status: 0, stdout: "" };
  };
  let output = "";
  const code = await main({
    argv: ["install", "--install-dir", installDir, "--clients", "codex"],
    env: {
      EVE_RELEASE_BASE_URL: "https://releases.example/v1",
      EVE_VERSION: "1.2.3",
      PATH: installDir,
    },
    platform: "linux",
    arch: "x64",
    fetchImpl,
    spawn,
    stdout: { write: (value) => { output += value; } },
  });

  const destination = path.join(installDir, "eve");
  assert.equal(code, 0);
  assert.deepEqual(await readFile(destination), bytes);
  assert.equal((await stat(destination)).mode & 0o777, 0o755);
  assert.equal(calls.length, 2);
  assert.equal(calls[0].args[0], "version");
  assert.deepEqual(calls[1], {
    command: destination,
    args: ["install-mcp", "--eve-bin", destination, "--clients", "codex"],
  });
  assert.match(output, /Installed EVE at/);
  assert.match(output, /Next: open a Git repository/);
});
