import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { readFile, mkdir, chmod, rename, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repository = "nhestrompia/eve";
const packagePath = fileURLToPath(new URL("../package.json", import.meta.url));

export async function main(options = {}) {
  const argv = options.argv ?? process.argv.slice(2);
  const env = options.env ?? process.env;
  const stdout = options.stdout ?? process.stdout;
  const platform = options.platform ?? process.platform;
  const arch = options.arch ?? process.arch;
  const fetchImpl = options.fetchImpl ?? globalThis.fetch;
  const spawn = options.spawn ?? spawnSync;

  const parsed = parseInstallArgs(argv);
  if (parsed.help) {
    stdout.write(helpText());
    return 0;
  }

  const packageMetadata = JSON.parse(await readFile(packagePath, "utf8"));
  const version = normalizeVersion(env.EVE_VERSION || packageMetadata.version);
  const installDir = path.resolve(parsed.installDir || defaultInstallDir({ env, platform, home: os.homedir() }));
  const binaryName = platform === "win32" ? "eve.exe" : "eve";
  const destination = path.join(installDir, binaryName);

  stdout.write(`Installing EVE ${version} for ${platform}/${arch}...\n`);
  const asset = await downloadReleaseAsset({
    version,
    platform,
    arch,
    baseUrl: env.EVE_RELEASE_BASE_URL,
    fetchImpl,
  });
  const temporary = await stageBinary(destination, asset.bytes, platform);
  try {
    verifyBinary(temporary, version, spawn);
    await activateBinary(temporary, destination, platform);
  } finally {
    await rm(temporary, { force: true });
  }
  stdout.write(`Installed EVE at ${destination}\n`);

  if (!parsed.noMCP) {
    const args = ["install-mcp", "--eve-bin", destination];
    if (parsed.clients) {
      args.push("--clients", parsed.clients);
    }
    const result = spawn(destination, args, { encoding: "utf8", stdio: "inherit" });
    if (result.error) {
      throw new Error(`could not configure MCP clients: ${result.error.message}`);
    }
    if (result.status !== 0) {
      throw new Error(`eve install-mcp exited with status ${result.status}`);
    }
  }

  printPathGuidance(stdout, installDir, env, platform);
  stdout.write("\nNext: open a Git repository and run `eve init`.\n");
  return 0;
}

export function parseInstallArgs(argv) {
  const args = [...argv];
  if (args[0] === "install") {
    args.shift();
  } else if (args.length > 0 && args[0] !== "--help" && args[0] !== "-h") {
    throw new Error("expected `install`; run with --help for usage");
  }

  const result = { clients: "", help: false, installDir: "", noMCP: false };
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === "--help" || arg === "-h") {
      result.help = true;
      continue;
    }
    if (arg === "--no-mcp") {
      result.noMCP = true;
      continue;
    }
    if (arg === "--clients" || arg === "--install-dir") {
      const value = args[index + 1];
      if (!value || value.startsWith("--")) {
        throw new Error(`${arg} requires a value`);
      }
      if (arg === "--clients") {
        result.clients = value;
      } else {
        result.installDir = value;
      }
      index += 1;
      continue;
    }
    if (arg.startsWith("--clients=")) {
      result.clients = arg.slice("--clients=".length);
      continue;
    }
    if (arg.startsWith("--install-dir=")) {
      result.installDir = arg.slice("--install-dir=".length);
      continue;
    }
    throw new Error(`unknown option ${arg}`);
  }
  return result;
}

export function normalizeVersion(value) {
  const version = String(value).trim().replace(/^v/, "");
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version) || version.includes("development")) {
    throw new Error(`npm package version ${JSON.stringify(value)} is not tied to a published EVE release`);
  }
  return version;
}

export function releaseAssetName(platform, arch) {
  const platforms = { darwin: "darwin", linux: "linux", win32: "windows" };
  const architectures = { arm64: "arm64", x64: "amd64" };
  const releasePlatform = platforms[platform];
  const releaseArch = architectures[arch];
  if (!releasePlatform || !releaseArch) {
    throw new Error(`unsupported platform ${platform}/${arch}`);
  }
  const suffix = platform === "win32" ? ".exe" : "";
  return `eve-${releasePlatform}-${releaseArch}${suffix}`;
}

export function defaultInstallDir({ env, platform, home }) {
  if (env.EVE_INSTALL_DIR) {
    return env.EVE_INSTALL_DIR;
  }
  if (platform === "win32") {
    const localAppData = env.LOCALAPPDATA || env.APPDATA || path.join(home, "AppData", "Local");
    return path.join(localAppData, "EVE", "bin");
  }
  return path.join(home, ".local", "bin");
}

export async function downloadReleaseAsset({ version, platform, arch, baseUrl, fetchImpl }) {
  const assetName = releaseAssetName(platform, arch);
  const releaseRoot = (baseUrl || `https://github.com/${repository}/releases/download/v${version}`).replace(/\/$/, "");
  const [checksums, bytes] = await Promise.all([
    fetchBytes(`${releaseRoot}/SHA256SUMS`, fetchImpl),
    fetchBytes(`${releaseRoot}/${assetName}`, fetchImpl),
  ]);
  verifyReleaseAsset(assetName, bytes, checksums.toString("utf8"));
  return { assetName, bytes };
}

export function verifyReleaseAsset(assetName, bytes, checksums) {
  const expected = parseChecksums(checksums).get(assetName);
  if (!expected) {
    throw new Error(`${assetName} is missing from SHA256SUMS`);
  }
  const actual = createHash("sha256").update(bytes).digest("hex");
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${assetName}`);
  }
}

export function parseChecksums(value) {
  const checksums = new Map();
  for (const line of String(value).split(/\r?\n/)) {
    const match = line.trim().match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (match) {
      checksums.set(path.basename(match[2]), match[1].toLowerCase());
    }
  }
  return checksums;
}

async function fetchBytes(url, fetchImpl) {
  if (typeof fetchImpl !== "function") {
    throw new Error("Node.js 20 or newer is required to download EVE");
  }
  const response = await fetchImpl(url, {
    headers: { "user-agent": "@nhestrompia/eve installer" },
    redirect: "follow",
  });
  if (!response.ok) {
    throw new Error(`download ${url} returned ${response.status} ${response.statusText}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

async function stageBinary(destination, bytes, platform) {
  await mkdir(path.dirname(destination), { recursive: true });
  const extension = platform === "win32" ? ".exe" : "";
  const base = path.basename(destination, extension);
  const temporary = path.join(path.dirname(destination), `.${base}.download-${process.pid}-${Date.now()}${extension}`);
  await writeFile(temporary, bytes, { mode: 0o755 });
  if (platform !== "win32") {
    await chmod(temporary, 0o755);
  }
  return temporary;
}

async function activateBinary(temporary, destination, platform) {
  if (platform === "win32") {
    await rm(destination, { force: true });
  }
  await rename(temporary, destination);
}

function verifyBinary(destination, version, spawn) {
  const result = spawn(destination, ["version"], { encoding: "utf8" });
  if (result.error) {
    throw new Error(`installed binary could not run: ${result.error.message}`);
  }
  if (result.status !== 0 || !String(result.stdout).includes(`eve ${version}`)) {
    throw new Error(`installed binary did not report EVE ${version}`);
  }
}

function printPathGuidance(stdout, installDir, env, platform) {
  const entries = String(env.PATH || "").split(path.delimiter).map((entry) => path.resolve(entry || "."));
  if (entries.includes(path.resolve(installDir))) {
    return;
  }
  stdout.write(`\n${installDir} is not currently on PATH.\n`);
  if (platform === "win32") {
    stdout.write("Add that directory to your user PATH, then open a new terminal.\n");
  } else {
    stdout.write(`Add this line to your shell profile:\n  export PATH="${installDir}:$PATH"\n`);
  }
}

function helpText() {
  return `Install EVE globally from its checksummed release artifacts.

Usage:
  npx @nhestrompia/eve@latest install [options]

Options:
  --clients <list>      MCP clients to configure (default: codex,claude,opencode)
  --install-dir <path>  Override the user installation directory
  --no-mcp              Install the CLI without configuring MCP clients
  -h, --help            Show this help
`;
}
