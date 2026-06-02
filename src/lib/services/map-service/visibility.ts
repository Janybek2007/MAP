import { pointInRing } from '../../utils/geo';
import type { MarkerBuckets, MarkerFilterCounts, StoreGeoState } from './types';

function inActiveDistricts(
	lat: number,
	lng: number,
	activeDistricts: StoreGeoState['activeDistricts']
) {
	if (activeDistricts.length === 0) return true;
	return activeDistricts.some((district) =>
		district.rings.some((ring: number[][]) => pointInRing(lat, lng, ring))
	);
}

function inSelectedLayers(params: {
	lat: number;
	lng: number;
	selectedHids: string[];
	layerCache: Map<string, { polygon: any; rings: number[][][] }>;
}) {
	if (params.selectedHids.length === 0) return true;
	return params.selectedHids.some((hid) => {
		const layer = params.layerCache.get(hid);
		if (!layer) return false;
		const bounds = layer.polygon?.getBounds?.();
		if (
			bounds &&
			typeof bounds.contains === 'function' &&
			!bounds.contains([params.lat, params.lng])
		)
			return false;
		return layer.rings.some((ring) => pointInRing(params.lat, params.lng, ring));
	});
}

function childKey(category: string, child: string) {
	return `${category}:${child}`;
}

export function computeMarkerVisibility(params: {
	state: StoreGeoState;
	markersByType: MarkerBuckets;
	map: any;
	regionLayerCache: Map<string, { polygon: any; rings: number[][][] }>;
	cityLayerCache: Map<string, { polygon: any; rings: number[][][] }>;
}): MarkerFilterCounts {
	const { state } = params;
	const visibleCounts = { bonetsky: 0, gos: 0, rival: 0, chastnyi: 0 };
	const geoCategoryCounts: Record<string, number> = {};
	const geoChildCounts: Record<string, number> = {};

	Object.keys(params.markersByType).forEach((category) => {
		geoCategoryCounts[category] = 0;
		params.markersByType[category].forEach((item) => {
			if (item.child_category) geoChildCounts[childKey(item.category, item.child_category)] = 0;
		});
	});

	Object.keys(params.markersByType).forEach((category) => {
		params.markersByType[category].forEach((item) => {
			const districtOn = inActiveDistricts(item.lat, item.lng, state.activeDistricts);
			const regionOn = inSelectedLayers({
				lat: item.lat,
				lng: item.lng,
				selectedHids: state.selectedRegionHids,
				layerCache: params.regionLayerCache
			});
			const cityOn = inSelectedLayers({
				lat: item.lat,
				lng: item.lng,
				selectedHids: state.selectedCityHids,
				layerCache: params.cityLayerCache
			});
			const geoOn = districtOn && regionOn && cityOn;

			if (geoOn) {
				geoCategoryCounts[item.category] = (geoCategoryCounts[item.category] || 0) + 1;
				if (item.child_category) {
					const key = childKey(item.category, item.child_category);
					geoChildCounts[key] = (geoChildCounts[key] || 0) + 1;
				}
			}

			const categoryOn = state.categoryActive[item.category] !== false;
			const childOn = item.child_category ? state.childActive[childKey(item.category, item.child_category)] !== false : true;

			if (geoOn && categoryOn && childOn) {
				if (!params.map.hasLayer(item.marker)) item.marker.addTo(params.map);
				if (item.category in visibleCounts)
					visibleCounts[item.category as keyof typeof visibleCounts] += 1;
			} else if (params.map.hasLayer(item.marker)) {
				params.map.removeLayer(item.marker);
			}
		});
	});

	return { geoCategoryCounts, geoChildCounts, visibleCounts };
}
