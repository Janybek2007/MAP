import { STORAGE_KEY } from '../config/mapConfig';

export function loadSaved(): Record<string, any> {
	try {
		return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
	} catch {
		return {};
	}
}

export function savePatch(patch: Record<string, unknown>) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(Object.assign(loadSaved(), patch)));
}
