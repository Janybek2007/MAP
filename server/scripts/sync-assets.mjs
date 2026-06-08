import { cp, mkdir, rm } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const serverDir = resolve(scriptDir, '..');
const rootDir = resolve(serverDir, '..');

const pairs = [
	[resolve(rootDir, 'build'), resolve(serverDir, 'build')]
];

await rm(resolve(serverDir, 'data'), { recursive: true, force: true });

for (const [source, target] of pairs) {
	await rm(target, { recursive: true, force: true });
	await mkdir(target, { recursive: true });
	await cp(source, target, { recursive: true, force: true });
}
