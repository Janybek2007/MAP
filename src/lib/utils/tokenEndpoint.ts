type TokenEndpointValue =
	| string
	| {
			token?: string;
			expires_at?: number;
	  };

declare global {
	interface Window {
		TOKEN?: TokenEndpointValue;
	}
}

export function tokenEndpointHeaders(): Record<string, string> {
	if (typeof window === 'undefined') return {};

	const value = window.TOKEN;
	const token = typeof value === 'string' ? value : value?.token;
	if (!token) return {};

	return { 'X-Token': token };
}

export function updateTokenEndpointToken(response: Response) {
	if (typeof window === 'undefined') return;

	const token = response.headers.get('X-Next-Token');
	if (!token) return;

	const expiresAt = Number(response.headers.get('X-Next-Token-Expires-At') ?? 0);
	window.TOKEN =
		Number.isFinite(expiresAt) && expiresAt > 0 ? { token, expires_at: expiresAt } : token;
}
