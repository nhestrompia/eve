import { cp, rm, mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const uiRoot = resolve(here, '..');
const repoRoot = resolve(uiRoot, '..');
const source = resolve(uiRoot, 'dist');
const target = resolve(repoRoot, 'cmd/eve/ui_dist');

await rm(target, { recursive: true, force: true });
await mkdir(target, { recursive: true });
await cp(source, target, { recursive: true });
