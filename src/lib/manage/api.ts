import type { ApiError } from './types';
import { tokenEndpointHeaders, updateTokenEndpointToken } from '$lib/utils/tokenEndpoint';

type TokenRequest = {
	url: string;
	method: string;
};

type TokenResponse = {
	token: string;
	expires_at: number;
};

export async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
	const response = await fetch(url, {
		...init,
		headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) }
	});
	if (!response.ok) {
		let payload: ApiError | null = null;
		try {
			payload = await response.json();
		} catch {
			payload = null;
		}
		throw payload ?? { message: `HTTP ${response.status}` };
	}
	return response.json() as Promise<T>;
}

export async function requestTokens(requests: TokenRequest[]): Promise<TokenResponse[]> {
	const response = await fetch('/api/tokens', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', ...tokenEndpointHeaders() },
		body: JSON.stringify(requests)
	});

	if (!response.ok) {
		let payload: ApiError | null = null;
		try {
			payload = await response.json();
		} catch {
			payload = null;
		}
		throw payload ?? { message: `HTTP ${response.status}` };
	}

	updateTokenEndpointToken(response);
	return response.json() as Promise<TokenResponse[]>;
}

export async function requestToken(url: string, method: string): Promise<string> {
	const [response] = await requestTokens([{ url, method }]);
	return response.token;
}

export async function fetchProtectedJSON<T>(url: string): Promise<T> {
	const token = await requestToken(url, 'GET');
	return fetchJSON<T>(url, {
		headers: {
			'X-Resource-Token': token
		}
	});
}
