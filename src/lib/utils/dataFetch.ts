import { tokenEndpointHeaders, updateTokenEndpointToken } from './tokenEndpoint';

type BatchEntry = {
	method: string;
	path: string;
	resolve: (token: string) => void;
	reject: (err: Error) => void;
};

const batches = new Map<string, BatchEntry[]>();
const scheduled = new Set<string>();

async function flush(apiBase: string) {
	const batch = batches.get(apiBase) ?? [];
	batches.delete(apiBase);
	scheduled.delete(apiBase);
	if (!batch.length) return;

	try {
		const res = await fetch(`${apiBase}/api/tokens`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', ...tokenEndpointHeaders() },
			body: JSON.stringify(batch.map((e) => ({ method: e.method, url: e.path })))
		});
		if (!res.ok) throw new Error(`Token request failed: ${res.status}`);
		updateTokenEndpointToken(res);
		const tokens = (await res.json()) as Array<{ token: string }>;
		batch.forEach((entry, i) => entry.resolve(tokens[i].token));
	} catch (err) {
		const e = err instanceof Error ? err : new Error(String(err));
		batch.forEach((entry) => entry.reject(e));
	}
}

function queueToken(apiBase: string, path: string, method = 'GET'): Promise<string> {
	if (!batches.has(apiBase)) batches.set(apiBase, []);
	return new Promise((resolve, reject) => {
		batches.get(apiBase)!.push({ method, path, resolve, reject });
		if (!scheduled.has(apiBase)) {
			scheduled.add(apiBase);
			Promise.resolve().then(() => flush(apiBase));
		}
	});
}

export async function fetchWithToken(apiBase: string, path: string): Promise<Response> {
	const token = await queueToken(apiBase, path);
	return fetch(`${apiBase}${path}`, {
		headers: { 'X-Resource-Token': token }
	});
}
