export function pointInRing(lat: number, lng: number, ring: number[][]) {
	let inside = false;
	for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
		const a = ring[i];
		const b = ring[j];
		if (a[0] > lat !== b[0] > lat && lng < ((b[1] - a[1]) * (lat - a[0])) / (b[0] - a[0]) + a[1]) {
			inside = !inside;
		}
	}
	return inside;
}
