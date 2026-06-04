import { get, writable } from 'svelte/store';
import type {
	ActiveDistrict,
	CityItem,
	DistrictItem,
	LocationItem,
	RegionItem,
	StoreState
} from '../types/map';
import { loadSaved, savePatch } from '../utils/storage';

const initialState: StoreState = {
	isFilterOpen: false,
	markersByType: {},
	categoryActive: {},
	childActive: {},
	expanded: {},
	activeDistricts: [],
	locations: [],
	districts: [],
	cities: [],
	regions: [],
	selectedCityHids: [],
	selectedRegionHids: [],
	filteredCounts: {
		bonetsky: 0,
		gos: 0,
		rival: 0,
		chastnyi: 0
	},
	filteredCategoryCounts: {},
	filteredChildCounts: {}
};

const { subscribe, set, update } = writable<StoreState>(initialState);

function getChildrenByCategory(locations: LocationItem[]) {
	const map: Record<string, string[]> = {};
	for (const location of locations) {
		const category = location.category;
		const child = category === 'bonetsky' ? location.type : location.child_category;
		if (!category || !child) continue;
		if (!map[category]) map[category] = [];
		if (!map[category].includes(child)) map[category].push(child);
	}
	return map;
}

function childKey(category: string, child: string) {
	return `${category}:${child}`;
}

function computeParentActive(
	state: StoreState,
	category: string,
	childrenMap: Record<string, string[]>
) {
	const children = childrenMap[category] || [];
	if (!children.length) return false;
	const enabledCount = children.filter(
		(child) => state.childActive[childKey(category, child)] !== false
	).length;
	return enabledCount > 0;
}

function filterActiveDistricts(nextState: StoreState) {
	const selectedCityHids = nextState.selectedCityHids;
	const selectedRegionHids = nextState.selectedRegionHids;
	return nextState.activeDistricts.filter((active) => {
		const district = nextState.districts[active.id];
		if (!district) return false;
		if (selectedCityHids.length > 0) {
			return Boolean(district.city_hid && selectedCityHids.includes(district.city_hid));
		}
		if (selectedRegionHids.length === 0) return true;
		return Boolean(district.region_hid && selectedRegionHids.includes(district.region_hid));
	});
}

export const mapStore = {
	subscribe,
	reset() {
		set(initialState);
	},
	clearDistricts() {
		update((state) => ({ ...state, activeDistricts: [] }));
	},
	hydrateFromSaved() {
		const saved = loadSaved();
		update((state) => ({
			...state,
			isFilterOpen: Boolean(saved.open),
			expanded: saved.expanded || {}
		}));
	},
	setData(
		locations: LocationItem[],
		districts: DistrictItem[],
		cities: CityItem[],
		regions: RegionItem[]
	) {
		const saved = loadSaved();
		const savedChildren = saved.children || {};
		const childActive: Record<string, boolean> = {};
		const categoryActive: Record<string, boolean> = {};
		const childrenMap = getChildrenByCategory(locations);

		Object.keys(childrenMap).forEach((category) => {
			(childrenMap[category] || []).forEach((child) => {
				const key = childKey(category, child);
				childActive[key] = savedChildren[key] !== false;
			});
			categoryActive[category] = computeParentActive(
				{ ...initialState, childActive } as StoreState,
				category,
				childrenMap
			);
		});

		update((state) => ({
			...state,
			locations,
			districts,
			cities,
			regions,
			childActive,
			categoryActive,
			expanded: saved.expanded || {}
		}));
	},
	toggleRegionHid(hid: string) {
		update((state) => {
			const exists = state.selectedRegionHids.includes(hid);
			const selectedRegionHids = exists
				? state.selectedRegionHids.filter((item) => item !== hid)
				: [...state.selectedRegionHids, hid];
			const nextState = { ...state, selectedRegionHids, selectedCityHids: [] } as StoreState;
			return { ...nextState, activeDistricts: filterActiveDistricts(nextState) };
		});
	},
	toggleCityHid(hid: string) {
		update((state) => {
			const exists = state.selectedCityHids.includes(hid);
			const selectedCityHids = exists
				? state.selectedCityHids.filter((item) => item !== hid)
				: [...state.selectedCityHids, hid];
			const nextState = { ...state, selectedCityHids } as StoreState;
			return { ...nextState, activeDistricts: filterActiveDistricts(nextState) };
		});
	},
	setMarkersByType(markersByType: Record<string, any[]>) {
		update((state) => ({ ...state, markersByType }));
	},
	setPanelOpen(value: boolean) {
		savePatch({ open: value });
		update((state) => ({ ...state, isFilterOpen: value }));
	},
	toggleChild(category: string, child: string) {
		update((state) => {
			const childrenMap = getChildrenByCategory(state.locations);
			const key = childKey(category, child);
			const next = !state.childActive[key];
			const nextChildActive = { ...state.childActive, [key]: next };
			const nextCategoryActive = {
				...state.categoryActive,
				[category]: computeParentActive(
					{ ...state, childActive: nextChildActive },
					category,
					childrenMap
				)
			};
			const savedChildren = loadSaved().children || {};
			savedChildren[key] = next;
			savePatch({ children: savedChildren, categories: nextCategoryActive });
			return { ...state, childActive: nextChildActive, categoryActive: nextCategoryActive };
		});
	},
	toggleCategory(category: string) {
		update((state) => {
			const childrenMap = getChildrenByCategory(state.locations);
			const children = childrenMap[category] || [];
			const enabledCount = children.filter(
				(child) => state.childActive[childKey(category, child)] !== false
			).length;
			const nextValue = enabledCount !== children.length;
			const nextChildActive = { ...state.childActive };
			children.forEach((child) => {
				nextChildActive[childKey(category, child)] = nextValue;
			});
			const nextCategoryActive = { ...state.categoryActive, [category]: nextValue };
			const savedChildren = loadSaved().children || {};
			children.forEach((child) => {
				savedChildren[childKey(category, child)] = nextValue;
			});
			savePatch({ children: savedChildren, categories: nextCategoryActive });
			return { ...state, childActive: nextChildActive, categoryActive: nextCategoryActive };
		});
	},
	toggleExpanded(category: string) {
		update((state) => {
			const nextExpanded = { ...state.expanded, [category]: !state.expanded[category] };
			savePatch({ expanded: nextExpanded });
			return { ...state, expanded: nextExpanded };
		});
	},
	setActiveDistricts(activeDistricts: ActiveDistrict[]) {
		update((state) => ({ ...state, activeDistricts }));
	},
	setFilteredCounts(filteredCounts: StoreState['filteredCounts']) {
		update((state) => ({ ...state, filteredCounts }));
	},
	setFilteredMarkerCounts(
		filteredCategoryCounts: Record<string, number>,
		filteredChildCounts: Record<string, number>
	) {
		update((state) => ({ ...state, filteredCategoryCounts, filteredChildCounts }));
	},
	getState() {
		return get({ subscribe });
	}
};
