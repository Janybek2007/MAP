import type { ActiveDistrict, MarkerEntry } from '../../types/map';

export type TooltipElements = {
	pinTooltipEl: HTMLDivElement;
	pinTooltipTitleEl: HTMLDivElement;
	pinTooltipCatEl: HTMLDivElement;
	districtTooltipEl: HTMLDivElement;
};

export type DistrictLayer = {
	id: number;
	title: string;
	population: number;
	color: string;
	rings: number[][][];
	polygon?: any;
};

export type MarkerBuckets = Record<string, MarkerEntry[]>;

export type MarkerCounts = {
	bonetsky: number;
	gos: number;
	rival: number;
	chastnyi: number;
};

export type MarkerFilterCounts = {
	geoCategoryCounts: Record<string, number>;
	geoChildCounts: Record<string, number>;
	visibleCounts: MarkerCounts;
};

export type ServiceCaches = {
	regionLayerCache: Map<string, { polygon: any; rings: number[][][] }>;
	cityLayerCache: Map<string, { polygon: any; rings: number[][][] }>;
	districtCoordsCache: Map<number, number[][][]>;
	regionCoordsPromise: Map<string, Promise<number[][][]>>;
	cityCoordsPromise: Map<string, Promise<number[][][]>>;
	districtCoordsPromise: Map<number, Promise<number[][][]>>;
	visibleRegionLayers: Set<string>;
	visibleCityLayers: Set<string>;
	visibleDistrictLayers: Set<number>;
};

export type StoreGeoState = {
	selectedRegionHids: string[];
	selectedCityHids: string[];
	activeDistricts: ActiveDistrict[];
	categoryActive: Record<string, boolean>;
	childActive: Record<string, boolean>;
	filteredCounts: MarkerCounts;
};
