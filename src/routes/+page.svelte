<script lang="ts">
	import { onMount } from 'svelte';
	import FilterPanel from '$lib/components/FilterPanel.svelte';
	import LoadingScreen from '$lib/components/LoadingScreen.svelte';
	import MapCanvas from '$lib/components/MapCanvas.svelte';
	import Tooltips from '$lib/components/Tooltips.svelte';
	import { createQuery } from '$lib/query/createQuery.svelte';
	import { MapService } from '$lib/services/mapService';
	import { mapStore } from '$lib/store/mapStore';
	import type { CityItem, DistrictItem, LocationItem, RegionItem } from '$lib/types/map';

	// Все 4 запроса батчируются в один POST /api/tokens
	const locationsQ = createQuery<{ locations: LocationItem[] }>('/data/locations');
	const districtsQ = createQuery<DistrictItem[]>('/data/districts');
	const citiesQ = createQuery<CityItem[]>('/data/cities');
	const regionsQ = createQuery<RegionItem[]>('/data/regions');

	// Не реактивные — устанавливаются через стабильные колбэки до первого $effect
	let mapEl: HTMLDivElement | null = null;
	let tooltipRefs: {
		pinTooltipEl: HTMLDivElement;
		pinTooltipTitleEl: HTMLDivElement;
		pinTooltipCatEl: HTMLDivElement;
		districtTooltipEl: HTMLDivElement;
	} | null = null;
	let mapService: MapService | null = null;
	const homeReloadKey = 'map:home:reload-once';
	const reloadDelayMs = 1;

	let isReady = $state(false);
	let loadingError = $state<string | null>(null);
	let dgInstance = $state<any>(null);

	const allLoading = $derived(
		locationsQ.loading || districtsQ.loading || citiesQ.loading || regionsQ.loading
	);
	const dataError = $derived(
		locationsQ.error || districtsQ.error || citiesQ.error || regionsQ.error || null
	);

	$effect(() => {
		if (isReady || !dgInstance || allLoading) return;

		if (dataError) {
			loadingError = dataError;
			return;
		}

		if (!locationsQ.data || !districtsQ.data || !citiesQ.data || !regionsQ.data) return;
		if (!mapEl || !tooltipRefs) return;

		const locations = locationsQ.data.locations ?? [];
		const districts = districtsQ.data;
		const cities = citiesQ.data;
		const regions = regionsQ.data;

		mapStore.hydrateFromSaved();
		mapStore.setData(locations, districts, cities, regions);

		tooltipRefs.pinTooltipEl.remove();
		tooltipRefs.districtTooltipEl.remove();

		mapService = new MapService(dgInstance);
		mapService.setApiBase('');
		mapService.init(mapEl, tooltipRefs, locations, districts);
		isReady = true;
	});

	function setMapEl(el: HTMLDivElement) {
		mapEl = el;
	}

	function setTooltipRefs(refs: typeof tooltipRefs) {
		tooltipRefs = refs;
	}

	function retry() {
		mapService?.destroy();
		mapService = null;
		isReady = false;
		loadingError = null;
		locationsQ.refetch();
		districtsQ.refetch();
		citiesQ.refetch();
		regionsQ.refetch();
	}

	onMount(() => {
		if (!sessionStorage.getItem(homeReloadKey)) {
			sessionStorage.setItem(homeReloadKey, '1');
			const reloadTimer = window.setTimeout(() => location.reload(), reloadDelayMs);
			return () => window.clearTimeout(reloadTimer);
		}

		const DG = (window as any).DG;
		if (!DG) {
			loadingError = '2GIS SDK не инициализирован';
			return;
		}

		const ready =
			typeof DG.then === 'function'
				? new Promise<void>((res, rej) => DG.then(res, rej))
				: Promise.resolve();

		ready
			.then(() => {
				dgInstance = DG;
			})
			.catch((err: unknown) => {
				loadingError = err instanceof Error ? err.message : String(err);
			});

		return () => mapService?.destroy();
	});
</script>

<MapCanvas onReady={setMapEl} />
<Tooltips onReady={setTooltipRefs} />
{#if !isReady}
	<LoadingScreen {loadingError} onRetry={retry} />
{:else}
	<FilterPanel onToggleDistrict={(id) => mapService?.toggleDistrict(id)} />
{/if}
