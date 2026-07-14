#!/usr/bin/env node

import { main } from "../lib/installer.mjs";

try {
  process.exitCode = await main();
} catch (error) {
  console.error(`EVE installation failed: ${error instanceof Error ? error.message : error}`);
  process.exitCode = 1;
}
