import { CATEGORY_COLORS, CATEGORY_LABELS, CATEGORY_OVERLAYS } from '../../config/mapConfig';
import type { CategoryKey, LocationItem, MarkerEntry } from '../../types/map';
import type { MarkerBuckets, TooltipElements } from './types';

function isCategoryKey(value: string): value is CategoryKey {
	return value === 'gos' || value === 'bonetsky' || value === 'chastnyi' || value === 'rival';
}

export function placeMarkers(params: {
	DG: any;
	map: any;
	tooltips: TooltipElements;
	locations: LocationItem[];
	createPinIcon: (color: string, overlay?: { paths: string }) => any;
	createBrandPinIcon: (imagePath?: string, fallbackText?: string) => any;
	showTooltipNode: (el: HTMLElement & { _removeTimer?: number | null }) => void;
	hideTooltipNode: (el: HTMLElement & { _removeTimer?: number | null }) => void;
	escapeHtml: (value: unknown) => string;
}): MarkerBuckets {
	const { DG, map, tooltips, locations } = params;
	const buckets: MarkerBuckets = {};
	const BONETSKY_SALES = 128;

	function showPinTooltip(loc: LocationItem) {
	const { pinTooltipEl, pinTooltipTitleEl, pinTooltipCatEl } = tooltips;
	const category = loc.category || loc.type || '';
	const color = CATEGORY_COLORS[category] || '#888';
	const categoryLabel = loc.category_display || CATEGORY_LABELS[category] || category || 'Категория';
	const childLabel =
		category === 'bonetsky'
			? loc.type_display || loc.type || ''
			: loc.child_category_display || loc.child_category || '';

		pinTooltipEl.style.borderColor = color;
		pinTooltipTitleEl.textContent = loc.name || loc.address || categoryLabel;

		const rows = [
			`<span><b>${params.escapeHtml(categoryLabel)}</b>${childLabel ? ` · ${params.escapeHtml(childLabel)}` : ''}</span>`
		];
		if (loc.address) rows.push(`<span>${params.escapeHtml(loc.address)}</span>`);
		if (category === 'bonetsky') {
			rows.push(`<span class="tooltip-divider"></span>`);
			if (loc.manager)
				rows.push(`<span class="tooltip-manager">Менеджер: ${params.escapeHtml(loc.manager)}</span>`);
			rows.push(`<span class="tooltip-sales">Продажи: ${params.escapeHtml(BONETSKY_SALES)}</span>`);
		}
		pinTooltipCatEl.innerHTML = rows.join('');

		params.showTooltipNode(pinTooltipEl as HTMLElement & { _removeTimer?: number | null });
		const point = map.latLngToContainerPoint([loc.lat, loc.lng]);
		const tooltipH = pinTooltipEl.offsetHeight;
		pinTooltipEl.style.left = `${point.x}px`;
		pinTooltipEl.style.top = `${point.y - 24 - tooltipH - 6}px`;
	}

	locations.forEach((loc) => {
		const rawCategory = loc.category || loc.type || '';
		if (!rawCategory || !isCategoryKey(rawCategory)) return;
		const category = rawCategory;

		const color = CATEGORY_COLORS[category] || '#888';
		const overlay = CATEGORY_OVERLAYS[category] || null;
		let markerIcon = params.createPinIcon(color, overlay);

		if (category === 'rival') {
			if (loc.child_category === 'rival_express')
				markerIcon = params.createBrandPinIcon('images/express-plus.jpg');
			else if (loc.child_category === 'rival_sapat')
				markerIcon = params.createBrandPinIcon('images/sapatlab.jpg');
			else if (loc.child_category === 'rival_akvalab')
				markerIcon = params.createBrandPinIcon('images/aqualab.png');
			else if (loc.child_category === 'rival_evrolab')
				markerIcon = params.createBrandPinIcon('images/evrolab.svg');
		}

		const marker = DG.marker([loc.lat, loc.lng], { icon: markerIcon })
			.on('mouseover', () => showPinTooltip(loc))
			.on('mouseout', () =>
				params.hideTooltipNode(
					tooltips.pinTooltipEl as HTMLElement & { _removeTimer?: number | null }
				)
			);

		if (!buckets[category]) buckets[category] = [];
		buckets[category].push({
			marker,
			lat: loc.lat,
			lng: loc.lng,
			category,
			child_category: category === 'bonetsky' ? loc.type : loc.child_category,
			category_display: loc.category_display,
			child_category_display:
				category === 'bonetsky'
					? loc.type_display || loc.type
					: loc.child_category_display,
			manager: loc.manager
		} satisfies MarkerEntry);
	});

	return buckets;
}
