import type { CategoryConfig, FilterState, FormState, LocationItem } from './types';

export function createEmptyForm(): FormState {
	return {
		name: '',
		address: '',
		category: '',
		child_category: '',
		manager: '',
		is_partnerships: false,
		lat: '',
		lng: ''
	};
}

export function sortCategories(input: CategoryConfig[]): CategoryConfig[] {
	return [...input].sort((a, b) => {
		if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order;
		return a.label.localeCompare(b.label, 'ru');
	});
}

export function validateLocationForm(
	form: FormState,
	categories: CategoryConfig[]
): Record<string, string> {
	const errors: Record<string, string> = {};
	const name = form.name.trim();
	const category = form.category.trim();
	const childCategory = form.child_category.trim();
	const lat = Number(form.lat);
	const lng = Number(form.lng);

	if (!name) {
		errors.name = 'Название обязательно';
	}

	if (!category) {
		errors.category = 'Категория обязательна';
	}

	if (!childCategory) {
		errors.child_category = category === 'bonetsky' ? 'Тип обязателен' : 'Подкатегория обязательна';
	}

	if (!form.lat || !form.lng) {
		errors.lat = 'Сначала выбери точку на карте';
		errors.lng = 'Сначала выбери точку на карте';
		return errors;
	}

	if (!Number.isFinite(lat)) {
		errors.lat = 'Широта должна быть числом';
	} else if (lat < -90 || lat > 90) {
		errors.lat = 'Широта должна быть в диапазоне от -90 до 90';
	}

	if (!Number.isFinite(lng)) {
		errors.lng = 'Долгота должна быть числом';
	} else if (lng < -180 || lng > 180) {
		errors.lng = 'Долгота должна быть в диапазоне от -180 до 180';
	}

	if (category) {
		const categoryItem = categories.find((item) => item.key === category);
		if (!categoryItem) {
			errors.category = 'Категория не найдена';
		} else if (childCategory) {
			const hasChild = categoryItem.children.some((item) => item.key === childCategory);
			if (!hasChild) {
				errors.child_category =
					category === 'bonetsky' ? 'Выбери существующий тип' : 'Выбери существующую подкатегорию';
			}
		}
	}

	return errors;
}

export function childLabel(location: LocationItem): string {
	if (location.category === 'bonetsky') {
		return location.type_display || location.type || '';
	}
	return location.child_category_display || location.child_category || '';
}

export function filteredLocations(locations: LocationItem[], filter: FilterState): LocationItem[] {
	const query = filter.search.trim().toLowerCase();
	return locations.filter((loc) => {
		if (filter.category && loc.category !== filter.category) return false;
		if (filter.child_category) {
			const cur = loc.category === 'bonetsky' ? (loc.type ?? '') : (loc.child_category ?? '');
			if (cur !== filter.child_category) return false;
		}
		if (filter.is_partnerships && !loc.is_partnerships) return false;
		if (!query) return true;
		return [loc.name, loc.address, loc.manager]
			.filter(Boolean)
			.some((v) => String(v).toLowerCase().includes(query));
	});
}

export function toSelectOptions(
	items: { key?: string; hid?: string; label?: string; title?: string }[]
): { value: string; label: string }[] {
	return items
		.map((i) => ({ value: i.key ?? i.hid ?? '', label: i.label ?? i.title ?? '' }))
		.filter((i) => i.value && i.label);
}

export function popupCoords(anchor: HTMLElement | null, minWidth = 320) {
	if (!anchor) return { top: 0, left: 0, minWidth };
	return {
		top: anchor.offsetTop + anchor.offsetHeight + 8,
		left: anchor.offsetLeft,
		minWidth: Math.max(anchor.offsetWidth, minWidth)
	};
}

export function escapeHtml(text: string): string {
	return text
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

export function highlight(text: string | undefined, query: string): string {
	const safe = escapeHtml(text ?? '');
	const q = query.trim();
	if (!q) return safe;
	const pattern = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	return safe.replace(new RegExp(`(${pattern})`, 'gi'), '<mark class="hl">$1</mark>');
}
