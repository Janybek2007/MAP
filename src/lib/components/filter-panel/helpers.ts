import type { DistrictItem, StoreState } from '../../types/map';

export function isRegionChecked(state: StoreState, hid?: string) {
	if (!hid) return false;
	return state.selectedRegionHids.includes(hid);
}

export function isCityChecked(state: StoreState, hid?: string) {
	if (!hid) return false;
	return state.selectedCityHids.includes(hid);
}

export function filteredCities(state: StoreState) {
	if (state.selectedRegionHids.length === 0) return state.cities;
	return state.cities.filter(
		(city) => city.region_hid && state.selectedRegionHids.includes(city.region_hid)
	);
}

export function filteredDistricts(state: StoreState) {
	return state.districts.filter((district) => {
		const regionPass =
			state.selectedCityHids.length > 0 ||
			state.selectedRegionHids.length === 0 ||
			(district.region_hid && state.selectedRegionHids.includes(district.region_hid));
		const cityPass =
			state.selectedCityHids.length === 0 ||
			(district.city_hid && state.selectedCityHids.includes(district.city_hid));
		return Boolean(regionPass && cityPass);
	});
}

export function districtGroups(state: StoreState) {
	const items = filteredDistricts(state);
	const groupByRegionAndCity = () => {
		const grouped = new Map<string, DistrictItem[]>();
		items.forEach((district) => {
			const key = `${district.region_hid || 'unknown'}|${district.city_hid || 'none'}`;
			const next = grouped.get(key) || [];
			next.push(district);
			grouped.set(key, next);
		});

		return Array.from(grouped.entries())
			.map(([key, districts]) => {
				const [regionHid, cityHid] = key.split('|');
				const region = state.regions.find((item) => item.hid === regionHid);
				const city = cityHid !== 'none' ? state.cities.find((item) => item.hid === cityHid) : null;
				const title = city
					? `${region?.title || 'Область'} · ${city.title}`
					: region?.title || 'Область';
				return { title, districts };
			})
			.sort((a, b) => a.title.localeCompare(b.title, 'ru'));
	};

	if (state.selectedCityHids.length > 1) {
		const byCity = new Map<string, DistrictItem[]>();
		items.forEach((district) => {
			const key = district.city_hid || 'unknown';
			const next = byCity.get(key) || [];
			next.push(district);
			byCity.set(key, next);
		});

		return Array.from(byCity.entries()).map(([cityHid, districts]) => {
			const city = state.cities.find((c) => c.hid === cityHid);
			return { title: city?.title || 'Город', districts };
		});
	}

	if (state.selectedRegionHids.length > 1) {
		const byRegion = new Map<string, DistrictItem[]>();
		items.forEach((district) => {
			const key = district.region_hid || 'unknown';
			const next = byRegion.get(key) || [];
			next.push(district);
			byRegion.set(key, next);
		});

		return Array.from(byRegion.entries()).map(([regionHid, districts]) => {
			const region = state.regions.find((r) => r.hid === regionHid);
			return { title: region?.title || 'Область', districts };
		});
	}

	if (state.selectedCityHids.length === 0 && state.selectedRegionHids.length === 0) {
		return groupByRegionAndCity();
	}

	return [{ title: '', districts: items }];
}

export function selectedPopulation(state: StoreState) {
	if (state.activeDistricts.length > 0) {
		const activeIds = new Set(state.activeDistricts.map((item) => item.id));
		return filteredDistricts(state).reduce((sum, district) => {
			const idx = state.districts.indexOf(district);
			if (!activeIds.has(idx)) return sum;
			return sum + Number(district.population || 0);
		}, 0);
	}

	if (state.selectedCityHids.length > 0) {
		return state.selectedCityHids.reduce((sum, hid) => {
			const city = state.cities.find((item) => item.hid === hid);
			return sum + Number(city?.population || 0);
		}, 0);
	}

	if (state.selectedRegionHids.length === 1) {
		const region = state.regions.find((item) => item.hid === state.selectedRegionHids[0]);
		if (region?.population) {
			let value = Number(region.population);
			if (region.title === 'Чуйская') {
				const bishkek = state.cities.find((city) => city.title === 'Бишкек');
				if (bishkek?.population) value += Number(bishkek.population);
			} else if (region.title === 'Ошская') {
				const osh = state.cities.find((city) => city.title === 'Ош');
				if (osh?.population) value += Number(osh.population);
			}
			return value;
		}
	}

	if (state.selectedRegionHids.length > 0) {
		const selected = new Set(state.selectedRegionHids);
		const regionsSum = state.regions.reduce(
			(sum, region) =>
				sum + (region.hid && selected.has(region.hid) ? Number(region.population || 0) : 0),
			0
		);
		const citiesSum = state.cities.reduce(
			(sum, city) =>
				sum + (city.region_hid && selected.has(city.region_hid) ? Number(city.population || 0) : 0),
			0
		);
		return regionsSum + citiesSum;
	}

	const regionsSum = state.regions.reduce((sum, region) => sum + Number(region.population || 0), 0);
	const citiesSum = state.cities.reduce((sum, city) => sum + Number(city.population || 0), 0);
	return regionsSum + citiesSum;
}

export function formatPopulation(value: number) {
	return `${value.toLocaleString('ru-RU')} чел.`;
}
