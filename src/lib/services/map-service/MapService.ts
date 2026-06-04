import { mapStore } from '../../store/mapStore';
import { fetchWithToken } from '../../utils/dataFetch';
import type {
	ActiveDistrict,
	CityItem,
	DistrictItem,
	LocationItem,
	RegionItem,
	StoreState
} from '../../types/map';
import { normalizeRings } from './coords';
import { buildDistrictLayers, createDistrictPolygon } from './districts';
import { placeMarkers } from './markers';
import { ensureCityLayer, ensureRegionLayer } from './outlines';
import { computeMarkerVisibility } from './visibility';
import type { DistrictLayer, MarkerBuckets, TooltipElements } from './types';

export class MapService {
	private DG: any;
	private map: any;
	private markersByType: MarkerBuckets = {};
	private districtLayers: DistrictLayer[] = [];
	private regions: RegionItem[] = [];
	private cities: CityItem[] = [];

	private regionLayerCache = new Map<string, { polygon: any; rings: number[][][] }>();
	private cityLayerCache = new Map<string, { polygon: any; rings: number[][][] }>();
	private districtCoordsCache = new Map<number, number[][][]>();
	private regionCoordsPromise = new Map<string, Promise<number[][][]>>();
	private cityCoordsPromise = new Map<string, Promise<number[][][]>>();
	private districtCoordsPromise = new Map<number, Promise<number[][][]>>();

	private visibleRegionLayers = new Set<string>();
	private visibleCityLayers = new Set<string>();
	private visibleDistrictLayers = new Set<number>();

	private outlineRequestId = 0;
	private lastGeoKey = '';
	private lastCountsKey = '';
	private lastMarkerFilterKey = '';
	private markerVisibilityTimer: number | null = null;
	private unsubscribeStore: (() => void) | null = null;

	private tooltips!: TooltipElements;
	private apiBase = '';

	constructor(DG: any) {
		this.DG = DG;
	}

	setApiBase(apiBase: string) {
		this.apiBase = apiBase.replace(/\/+$/, '');
	}

	init(
		mapEl: HTMLDivElement,
		tooltips: TooltipElements,
		locations: LocationItem[],
		districts: DistrictItem[]
	) {
		this.tooltips = tooltips;
		this.map = this.DG.map(mapEl, {
			center: [42.8746, 74.5698],
			zoom: 13,
			zoomControl: false,
			fullscreenControl: false
		});

		this.markersByType = placeMarkers({
			DG: this.DG,
			map: this.map,
			tooltips,
			locations,
			createPinIcon: (color, overlay) => this.createPinIcon(color, overlay),
			createBrandPinIcon: (imagePath, fallbackText) =>
				this.createBrandPinIcon(imagePath, fallbackText),
			showTooltipNode: (el) => this.showTooltipNode(el),
			hideTooltipNode: (el) => this.hideTooltipNode(el),
			escapeHtml: (v) => this.escapeHtml(v)
		});

		this.districtLayers = buildDistrictLayers(districts);
		const state = mapStore.getState();
		this.regions = state.regions;
		this.cities = state.cities;
		mapStore.setMarkersByType(this.markersByType);
		this.bindStore();
	}

	destroy() {
		if (this.unsubscribeStore) this.unsubscribeStore();
		this.unsubscribeStore = null;
		if (this.markerVisibilityTimer) {
			clearTimeout(this.markerVisibilityTimer);
			this.markerVisibilityTimer = null;
		}
	}

	toggleDistrict(districtId: number) {
		void this.toggleDistrictAsync(districtId);
	}

	private async toggleDistrictAsync(districtId: number) {
		const layer = this.districtLayers.find((item) => item.id === districtId);
		if (!layer) return;
		if (!layer.polygon) {
			if (!layer.rings.length) layer.rings = await this.ensureDistrictCoords(districtId);
			if (!layer.rings.length) return;
			layer.polygon = createDistrictPolygon({
				DG: this.DG,
				rings: layer.rings,
				title: layer.title,
				population: layer.population,
				color: layer.color,
				showTooltipNode: (el) => this.showTooltipNode(el),
				hideTooltipNode: (el) => this.hideTooltipNode(el),
				positionDistrictTooltip: (e) => this.positionDistrictTooltip(e),
				districtTooltipEl: this.tooltips.districtTooltipEl,
				escapeHtml: (v) => this.escapeHtml(v)
			});
		}

		const state = mapStore.getState();
		const exists = state.activeDistricts.some((district) => district.id === districtId);
		let nextActiveDistricts: ActiveDistrict[];

		if (exists) {
			this.map.removeLayer(layer.polygon);
			this.visibleDistrictLayers.delete(districtId);
			nextActiveDistricts = state.activeDistricts.filter((district) => district.id !== districtId);
		} else {
			layer.polygon.addTo(this.map);
			this.visibleDistrictLayers.add(districtId);
			const bounds = layer.polygon.getBounds();
			if (bounds && typeof bounds.isValid === 'function' && bounds.isValid()) {
				this.map.fitBounds(bounds, { padding: [40, 40] });
			}
			nextActiveDistricts = [...state.activeDistricts, { id: districtId, rings: layer.rings }];
		}

		mapStore.setActiveDistricts(nextActiveDistricts);
	}

	private bindStore() {
		this.unsubscribeStore = mapStore.subscribe((state: StoreState) => {
			this.scheduleUpdateMarkerVisibility();
			const geoKey = `${state.selectedRegionHids.join(',')}|${state.selectedCityHids.join(',')}`;
			if (geoKey !== this.lastGeoKey) {
				this.lastGeoKey = geoKey;
				this.updateOutlineVisibility();
			}
			this.syncDistrictLayers();
		});
	}

	private syncDistrictLayers() {
		if (!this.map) return;
		const state = mapStore.getState();
		const activeIds = new Set<number>(state.activeDistricts.map((item) => item.id));

		for (const id of Array.from(this.visibleDistrictLayers)) {
			if (!activeIds.has(id)) {
				const layer = this.districtLayers.find((item) => item.id === id);
				if (layer?.polygon && this.map.hasLayer(layer.polygon)) this.map.removeLayer(layer.polygon);
				this.visibleDistrictLayers.delete(id);
			}
		}

		for (const id of activeIds) {
			if (!this.visibleDistrictLayers.has(id)) {
				const layer = this.districtLayers.find((item) => item.id === id);
				if (layer?.polygon && !this.map.hasLayer(layer.polygon)) layer.polygon.addTo(this.map);
				this.visibleDistrictLayers.add(id);
			}
		}
	}

	private scheduleUpdateMarkerVisibility() {
		if (this.markerVisibilityTimer) clearTimeout(this.markerVisibilityTimer);
		this.markerVisibilityTimer = window.setTimeout(() => {
			this.markerVisibilityTimer = null;
			this.updateMarkerVisibility();
		}, 16);
	}

	private updateMarkerVisibility() {
		const state = mapStore.getState();
		const { geoCategoryCounts, geoChildCounts, visibleCounts } = computeMarkerVisibility({
			state,
			markersByType: this.markersByType,
			map: this.map,
			regionLayerCache: this.regionLayerCache,
			cityLayerCache: this.cityLayerCache
		});

		const markerFilterKey = `${JSON.stringify(geoCategoryCounts)}|${JSON.stringify(geoChildCounts)}`;
		if (markerFilterKey !== this.lastMarkerFilterKey) {
			this.lastMarkerFilterKey = markerFilterKey;
			mapStore.setFilteredMarkerCounts(geoCategoryCounts, geoChildCounts);
		}

		const countsKey = `${visibleCounts.bonetsky}|${visibleCounts.gos}|${visibleCounts.rival}|${visibleCounts.chastnyi}`;
		if (
			countsKey !== this.lastCountsKey &&
			(state.filteredCounts.bonetsky !== visibleCounts.bonetsky ||
				state.filteredCounts.gos !== visibleCounts.gos ||
				state.filteredCounts.rival !== visibleCounts.rival ||
				state.filteredCounts.chastnyi !== visibleCounts.chastnyi)
		) {
			this.lastCountsKey = countsKey;
			mapStore.setFilteredCounts(visibleCounts);
		}
	}

	private updateOutlineVisibility() {
		void this.updateOutlineVisibilityAsync();
	}

	private async updateOutlineVisibilityAsync() {
		const requestId = ++this.outlineRequestId;
		const state = mapStore.getState();

		for (const hid of Array.from(this.visibleRegionLayers)) {
			if (!state.selectedRegionHids.includes(hid)) {
				const layer = this.regionLayerCache.get(hid);
				if (layer) this.map.removeLayer(layer.polygon);
				this.visibleRegionLayers.delete(hid);
			}
		}
		for (const hid of state.selectedRegionHids) {
			if (!this.visibleRegionLayers.has(hid)) {
				const layer = await ensureRegionLayer({
					DG: this.DG,
					map: this.map,
					apiBase: this.apiBase,
					hid,
					regions: this.regions,
					cache: this.regionLayerCache,
					promiseCache: this.regionCoordsPromise
				});
				const liveState = mapStore.getState();
				if (requestId !== this.outlineRequestId) return;
				if (layer && liveState.selectedRegionHids.includes(hid)) {
					layer.polygon.addTo(this.map);
					this.visibleRegionLayers.add(hid);
				}
			}
		}

		for (const hid of Array.from(this.visibleCityLayers)) {
			if (!state.selectedCityHids.includes(hid)) {
				const layer = this.cityLayerCache.get(hid);
				if (layer) this.map.removeLayer(layer.polygon);
				this.visibleCityLayers.delete(hid);
			}
		}
		for (const hid of state.selectedCityHids) {
			if (!this.visibleCityLayers.has(hid)) {
				const layer = await ensureCityLayer({
					DG: this.DG,
					map: this.map,
					apiBase: this.apiBase,
					hid,
					cities: this.cities,
					cache: this.cityLayerCache,
					promiseCache: this.cityCoordsPromise
				});
				const liveState = mapStore.getState();
				if (requestId !== this.outlineRequestId) return;
				if (layer && liveState.selectedCityHids.includes(hid)) {
					layer.polygon.addTo(this.map);
					this.visibleCityLayers.add(hid);
				}
			}
		}

		this.scheduleUpdateMarkerVisibility();
	}

	private async ensureDistrictCoords(districtId: number) {
		const cached = this.districtCoordsCache.get(districtId);
		if (cached) return cached;
		const existingPromise = this.districtCoordsPromise.get(districtId);
		if (existingPromise) return existingPromise;

		const state = mapStore.getState();
		const districtMeta = state.districts[districtId];
		if (!districtMeta?.hid) return [];

		const promise = fetchWithToken(this.apiBase, `/data/districts/${districtMeta.hid}/coords`)
			.then((res) => (res.ok ? res.json() : []))
			.then((raw) => normalizeRings(raw))
			.catch(() => []);

		this.districtCoordsPromise.set(districtId, promise);
		const rings = await promise;
		this.districtCoordsPromise.delete(districtId);
		this.districtCoordsCache.set(districtId, rings);
		return rings;
	}

	private escapeHtml(value: unknown) {
		return String(value || '')
			.replaceAll('&', '&amp;')
			.replaceAll('<', '&lt;')
			.replaceAll('>', '&gt;')
			.replaceAll('"', '&quot;')
			.replaceAll("'", '&#039;');
	}

	private showTooltipNode(el: HTMLElement & { _removeTimer?: number | null }) {
		if (el._removeTimer) {
			clearTimeout(el._removeTimer);
			el._removeTimer = null;
		}
		if (!el.isConnected) document.body.appendChild(el);
		el.classList.remove('shown');
		requestAnimationFrame(() => el.classList.add('shown'));
	}

	private hideTooltipNode(el: HTMLElement & { _removeTimer?: number | null }) {
		if (!el.isConnected) return;
		if (el._removeTimer) {
			clearTimeout(el._removeTimer);
			el._removeTimer = null;
		}
		el.classList.remove('shown');
		const remove = () => {
			if (!el.classList.contains('shown') && el.isConnected) el.remove();
		};
		const onEnd = (event: Event) => {
			if (event.target !== el) return;
			el.removeEventListener('transitionend', onEnd);
			remove();
		};
		el.addEventListener('transitionend', onEnd);
		el._removeTimer = window.setTimeout(() => {
			el.removeEventListener('transitionend', onEnd);
			remove();
		}, 220);
	}

	private createPinIcon(color: string, overlay?: { paths: string }) {
		const size = 34;
		const cx = 17;
		let svg = '';
		svg += `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">`;
		svg += `<circle cx="${cx}" cy="${cx}" r="16" fill="white" stroke="${color}" stroke-width="2.5"/>`;
		if (overlay) {
			const isHeart = overlay.paths.includes('9.5');
			const fill = isHeart ? color : 'none';
			svg += `<g transform="translate(6,6) scale(0.917)" stroke="${color}" stroke-width="2" fill="${fill}" stroke-linecap="round" stroke-linejoin="round">${overlay.paths}</g>`;
		}
		svg += '</svg>';
		return this.DG.divIcon({
			className: '',
			html: svg,
			iconSize: [size, size],
			iconAnchor: [cx, cx],
			popupAnchor: [0, -(cx + 4)]
		});
	}

	private createBrandPinIcon(imagePath?: string, fallbackText?: string) {
		const size = 34;
		const cx = 17;
		const content =
			`<div style="width:${size}px;height:${size}px;border-radius:50%;background:#fff;border:2px solid #111827;display:flex;align-items:center;justify-content:center;overflow:hidden;">` +
			(imagePath
				? `<img src="${imagePath}" alt="" style="width:100%;height:100%;object-fit:cover;border-radius:50%;" />`
				: `<span style="font:900 17px/1 Segoe UI,Arial,sans-serif;color:#111827;">${fallbackText || ''}</span>`) +
			'</div>';
		return this.DG.divIcon({
			className: '',
			html: content,
			iconSize: [size, size],
			iconAnchor: [cx, cx],
			popupAnchor: [0, -(cx + 4)]
		});
	}

	private positionDistrictTooltip(event: MouseEvent) {
		const tooltip = this.tooltips.districtTooltipEl;
		const gap = 14;
		const tw = tooltip.offsetWidth;
		const th = tooltip.offsetHeight;
		let x = event.clientX + gap;
		let y = event.clientY - Math.round(th / 2);
		if (x + tw > window.innerWidth - 8) x = event.clientX - tw - gap;
		if (y < 8) y = 8;
		if (y + th > window.innerHeight - 8) y = window.innerHeight - th - 8;
		tooltip.style.left = `${x}px`;
		tooltip.style.top = `${y}px`;
	}
}
