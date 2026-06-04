import { fetchWithToken } from '$lib/utils/dataFetch';

export type Query<T> = {
	readonly data: T | null;
	readonly loading: boolean;
	readonly error: string | null;
	refetch: () => void;
};

export function createQuery<T>(path: string, getApiBase: () => string = () => ''): Query<T> {
	let data = $state<T | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let tick = $state(0);

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		tick;
		const apiBase = getApiBase();
		let stale = false;

		loading = true;
		error = null;

		fetchWithToken(apiBase, path)
			.then((res) => {
				if (stale) return undefined;
				if (!res.ok) throw new Error(`HTTP ${res.status}`);
				return res.json() as Promise<T>;
			})
			.then((result) => {
				if (stale || result == null) return;
				data = result;
			})
			.catch((err) => {
				if (stale) return;
				error = err instanceof Error ? err.message : String(err);
			})
			.finally(() => {
				if (stale) return;
				loading = false;
			});

		return () => {
			stale = true;
		};
	});

	return {
		get data() {
			return data;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		refetch() {
			data = null;
			tick++;
		}
	};
}
