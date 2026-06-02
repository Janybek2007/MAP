export type CategoryKey = 'gos' | 'bonetsky' | 'chastnyi' | 'rival';

export type LocationItem = {
	lat: number;
	lng: number;
	name?: string;
	address?: string;
	manager?: string;
	hid?: string | number;
	type?: string;
	type_display?: string;
	category?: CategoryKey;
	child_category?: string;
	category_display?: string;
	child_category_display?: string;
	is_partnerships?: boolean;
};

export type DistrictItem = {
	title: string;
	population: number;
	lat?: number | null;
	lng?: number | null;
	coords?: number[][][];
	hid?: string;
	city_hid?: string | null;
	region_hid?: string | null;
};

export type RegionItem = {
	title: string;
	population: number;
	lat?: number | null;
	lng?: number | null;
	coords?: number[][][];
	hid?: string;
};

export type CityItem = {
	title: string;
	population: number;
	lat?: number | null;
	lng?: number | null;
	coords?: number[][][];
	hid?: string;
	region_hid?: string | null;
};

export type MarkerEntry = {
	marker: any;
	lat: number;
	lng: number;
	category: CategoryKey;
	child_category?: string;
	category_display?: string;
	child_category_display?: string;
	manager?: string;
};

export type ActiveDistrict = {
	id: number;
	rings: number[][][];
};

export type StoreState = {
	isFilterOpen: boolean;
	markersByType: Record<string, MarkerEntry[]>;
	categoryActive: Record<string, boolean>;
	childActive: Record<string, boolean>;
	expanded: Record<string, boolean>;
	activeDistricts: ActiveDistrict[];
	locations: LocationItem[];
	districts: DistrictItem[];
	cities: CityItem[];
	regions: RegionItem[];
	selectedCityHids: string[];
	selectedRegionHids: string[];
	filteredCounts: {
		bonetsky: number;
		gos: number;
		rival: number;
		chastnyi: number;
	};
	filteredCategoryCounts: Record<string, number>;
	filteredChildCounts: Record<string, number>;
};
