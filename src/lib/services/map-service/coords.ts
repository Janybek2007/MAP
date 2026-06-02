export function normalizeRings(coords: any): number[][][] {
	if (!Array.isArray(coords)) return [];
	if (coords.length === 0) return [];
	if (Array.isArray(coords[0]) && Array.isArray(coords[0][0])) return coords as number[][][];

	const rings: number[][][] = [];
	let current: number[][] = [];
	for (const item of coords) {
		if (!Array.isArray(item)) continue;
		if (item.length === 0) {
			if (current.length > 2) rings.push(current);
			current = [];
			continue;
		}
		if (item.length >= 2) current.push([item[0], item[1]]);
	}
	if (current.length > 2) rings.push(current);
	return rings;
}
