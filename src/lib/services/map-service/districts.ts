import type { DistrictItem } from '../../types/map';
import type { DistrictLayer } from './types';

export function buildDistrictLayers(districts: DistrictItem[]): DistrictLayer[] {
	const palette = [
		'#6366f1',
		'#0ea5e9',
		'#14b8a6',
		'#22c55e',
		'#84cc16',
		'#eab308',
		'#f59e0b',
		'#ef4444',
		'#ec4899',
		'#8b5cf6'
	];
	const layers: DistrictLayer[] = [];
	districts.forEach((district, index) => {
		layers.push({
			id: index,
			title: district.title,
			population: district.population,
			color: palette[index % palette.length],
			rings: []
		});
	});
	return layers;
}

export function createDistrictPolygon(params: {
	DG: any;
	rings: number[][][];
	title: string;
	population: number;
	color: string;
	showTooltipNode: (el: HTMLElement & { _removeTimer?: number | null }) => void;
	hideTooltipNode: (el: HTMLElement & { _removeTimer?: number | null }) => void;
	positionDistrictTooltip: (event: MouseEvent) => void;
	districtTooltipEl: HTMLDivElement;
	escapeHtml: (value: unknown) => string;
}): any {
	const { DG, rings, color } = params;
	const tooltip = params.districtTooltipEl;
	const subPolygons = rings.map((ring) => {
		const poly = DG.polygon(
			ring.map((coord) => [coord[0], coord[1]]),
			{
				color,
				weight: 2,
				fillColor: color,
				fillOpacity: 0.18
			}
		)
			.on('mouseover', (e: any) => {
				tooltip.innerHTML = `<strong>${params.escapeHtml(params.title)}</strong><span>${params.escapeHtml(
					`${Number(params.population || 0).toLocaleString('ru-RU')} чел.`
				)}</span>`;
				params.positionDistrictTooltip(e.originalEvent as MouseEvent);
				params.showTooltipNode(tooltip as any);
			})
			.on('mousemove', (e: any) => params.positionDistrictTooltip(e.originalEvent as MouseEvent))
			.on('mouseout', () => params.hideTooltipNode(tooltip as any));
		return poly;
	});
	return DG.featureGroup(subPolygons);
}
