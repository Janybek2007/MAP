import type { CityItem, RegionItem } from '../../types/map';
import { normalizeRings } from './coords';

export async function ensureRegionLayer(params: {
	DG: any;
	map: any;
	apiBase: string;
	hid: string;
	regions: RegionItem[];
	cache: Map<string, { polygon: any; rings: number[][][] }>;
	promiseCache: Map<string, Promise<number[][][]>>;
}): Promise<{ polygon: any; rings: number[][][] } | null> {
	const cached = params.cache.get(params.hid);
	if (cached) return cached;

	const existingPromise = params.promiseCache.get(params.hid);
	let rings: number[][][] = [];
	if (existingPromise) {
		rings = await existingPromise;
	} else {
		const promise = fetch(`${params.apiBase}/data/regions/${params.hid}/coords`)
			.then((res) => (res.ok ? res.json() : []))
			.then((raw) => normalizeRings(raw))
			.catch(() => []);
		params.promiseCache.set(params.hid, promise);
		rings = await promise;
		params.promiseCache.delete(params.hid);
	}

	const region = params.regions.find((item) => item.hid === params.hid);
	if (!region || !rings.length) return null;

	const polys = rings.map((ring) =>
		params.DG.polygon(
			ring.map((coord) => [coord[0], coord[1]]),
			{ color: '#1d4ed8', weight: 4, fill: false, fillOpacity: 0 }
		)
	);
	const layer = {
		polygon: params.DG.featureGroup(polys),
		rings: rings.map((r) => r.map((p) => [p[0], p[1]]))
	};
	params.cache.set(params.hid, layer);
	return layer;
}

export async function ensureCityLayer(params: {
	DG: any;
	map: any;
	apiBase: string;
	hid: string;
	cities: CityItem[];
	cache: Map<string, { polygon: any; rings: number[][][] }>;
	promiseCache: Map<string, Promise<number[][][]>>;
}): Promise<{ polygon: any; rings: number[][][] } | null> {
	const cached = params.cache.get(params.hid);
	if (cached) return cached;

	const existingPromise = params.promiseCache.get(params.hid);
	let rings: number[][][] = [];
	if (existingPromise) {
		rings = await existingPromise;
	} else {
		const promise = fetch(`${params.apiBase}/data/cities/${params.hid}/coords`)
			.then((res) => (res.ok ? res.json() : []))
			.then((raw) => normalizeRings(raw))
			.catch(() => []);
		params.promiseCache.set(params.hid, promise);
		rings = await promise;
		params.promiseCache.delete(params.hid);
	}

	const city = params.cities.find((item) => item.hid === params.hid);
	if (!city || !rings.length) return null;

	const polys = rings.map((ring) =>
		params.DG.polygon(
			ring.map((coord) => [coord[0], coord[1]]),
			{ color: '#15803d', weight: 4, fill: false, fillOpacity: 0 }
		)
	);
	const layer = {
		polygon: params.DG.featureGroup(polys),
		rings: rings.map((r) => r.map((p) => [p[0], p[1]]))
	};
	params.cache.set(params.hid, layer);
	return layer;
}
