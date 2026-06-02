<script lang="ts">
	import { onMount } from 'svelte';
	import FilterPanel from '$lib/components/FilterPanel.svelte';
	import LoadingScreen from '$lib/components/LoadingScreen.svelte';
	import MapCanvas from '$lib/components/MapCanvas.svelte';
	import Tooltips from '$lib/components/Tooltips.svelte';
	import { MapService } from '$lib/services/mapService';
	import { mapStore } from '$lib/store/mapStore';
	import type { CityItem, DistrictItem, LocationItem, RegionItem } from '$lib/types/map';
	import { env } from '$env/dynamic/public';

	function getApiBase() {
		return env.PUBLIC_API_BASE || window.location.origin;
	}

	let mapEl: HTMLDivElement | null = null;
	let mapService: MapService | null = null;
	let isReady = $state(false);
	let loadingError = $state<string | null>(null);
	let initInFlight = $state(false);

	let tooltipRefs: {
		pinTooltipEl: HTMLDivElement;
		pinTooltipTitleEl: HTMLDivElement;
		pinTooltipCatEl: HTMLDivElement;
		districtTooltipEl: HTMLDivElement;
	} | null = null;

	function setMapEl(el: HTMLDivElement) {
		mapEl = el;
	}

	function setTooltipRefs(refs: {
		pinTooltipEl: HTMLDivElement;
		pinTooltipTitleEl: HTMLDivElement;
		pinTooltipCatEl: HTMLDivElement;
		districtTooltipEl: HTMLDivElement;
	}) {
		tooltipRefs = refs;
	}

	async function tryInit() {
		if (initInFlight || isReady) return;
		if (!mapEl || !tooltipRefs) return;
		initInFlight = true;
		try {
			await init();
		} catch (error) {
			console.error('Ошибка инициализации карты:', error);
			loadingError = error instanceof Error ? error.message : String(error);
		} finally {
			initInFlight = false;
		}
	}

	async function init() {
		isReady = false;
		loadingError = null;
		const API_BASE = getApiBase();

		const DG = (window as any).DG;
		if (!DG) throw new Error('2GIS SDK не инициализирован');
		if (typeof DG.then === 'function') {
			await new Promise<void>((resolve, reject) =>
				DG.then(
					() => resolve(),
					(error: unknown) => reject(error)
				)
			);
		}

		const [locationsData, districtsData, citiesData, regionsData] = await Promise.all([
			fetch(`${API_BASE}/data/locations.json`).then((res) => {
				if (!res.ok) throw new Error('Не удалось загрузить data/locations.json');
				return res.json();
			}),
			fetch(`${API_BASE}/data/districts.json`).then((res) => {
				if (!res.ok) throw new Error('Не удалось загрузить data/districts.json');
				return res.json();
			}),
			fetch(`${API_BASE}/data/cities.json`).then((res) => {
				if (!res.ok) throw new Error('Не удалось загрузить data/cities.json');
				return res.json();
			}),
			fetch(`${API_BASE}/data/regions.json`).then((res) => {
				if (!res.ok) throw new Error('Не удалось загрузить data/regions.json');
				return res.json();
			})
		]);

		const locations: LocationItem[] = Array.isArray(locationsData)
			? locationsData
			: locationsData.locations || [];
		const districts: DistrictItem[] = Array.isArray(districtsData) ? districtsData : [];
		const cities: CityItem[] = Array.isArray(citiesData) ? citiesData : [];
		const regions: RegionItem[] = Array.isArray(regionsData) ? regionsData : [];

		mapStore.hydrateFromSaved();
		mapStore.setData(locations, districts, cities, regions);

		if (!mapEl || !tooltipRefs) return;
		tooltipRefs.pinTooltipEl.remove();
		tooltipRefs.districtTooltipEl.remove();

		mapService = new MapService(DG);
		mapService.setApiBase(API_BASE);
		mapService.init(mapEl, tooltipRefs, locations, districts);
		isReady = true;
	}

	function handleDistrictToggle(districtId: number) {
		mapService?.toggleDistrict(districtId);
	}

	onMount(() => {
		void tryInit();
		return () => {
			mapService?.destroy();
		};
	});

	$effect(() => {
		void tryInit();
	});
</script>

<MapCanvas onReady={setMapEl} />
<Tooltips onReady={setTooltipRefs} />
{#if !isReady}
	<LoadingScreen
		{loadingError}
		onRetry={() =>
			void init().catch((error) => {
				console.error('Ошибка инициализации карты:', error);
				loadingError = error instanceof Error ? error.message : String(error);
			})}
	/>
{:else}
	<FilterPanel onToggleDistrict={handleDistrictToggle} />
{/if}
