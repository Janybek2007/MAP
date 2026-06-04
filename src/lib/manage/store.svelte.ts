import type {
	ApiError,
	CategoryConfig,
	ChildCreateResponse,
	FilterState,
	FormState,
	LocationItem
} from './types';
import { fetchJSON, requestToken, requestTokens } from './api';
import { createEmptyForm, sortCategories } from './utils';

export function createManageStore() {
	let isLoading = $state(true);
	let isSaving = $state(false);
	let loadError = $state('');
	let formError = $state('');
	let isFilterPopupOpen = $state(false);
	let isChildPopupOpen = $state(false);
	let editingHid = $state('');
	let isPickingPoint = $state(false);
	let isCreatingChildCategory = $state(false);
	let categories = $state<CategoryConfig[]>([]);
	let locations = $state<LocationItem[]>([]);
	let fieldErrors = $state<Record<string, string>>({});
	let form = $state<FormState>(createEmptyForm());
	let newChildCategoryLabel = $state('');
	let filter = $state<FilterState>({
		search: '',
		category: '',
		child_category: '',
		is_partnerships: false
	});
	let activeTab = $state<'locations' | 'categories'>('locations');

	async function loadData() {
		isLoading = true;
		loadError = '';
		try {
			const [locationsToken, configToken] = await requestTokens([
				{ url: '/api/locations', method: 'GET' },
				{ url: '/api/location-config', method: 'GET' }
			]);
			const [locRes, confRes] = await Promise.all([
				fetchJSON<{ locations: LocationItem[] }>('/api/locations', {
					headers: { 'X-Resource-Token': locationsToken.token }
				}),
				fetchJSON<{ categories: CategoryConfig[] }>('/api/location-config', {
					headers: { 'X-Resource-Token': configToken.token }
				})
			]);
			locations = locRes.locations ?? [];
			categories = sortCategories(confRes.categories ?? []);
		} catch (err) {
			loadError = (err as ApiError).message ?? 'Не удалось загрузить данные';
		} finally {
			isLoading = false;
		}
	}

	function resetForm() {
		editingHid = '';
		fieldErrors = {};
		formError = '';
		newChildCategoryLabel = '';
		isChildPopupOpen = false;
		form = createEmptyForm();
		isPickingPoint = false;
	}

	function editLocation(location: LocationItem) {
		editingHid = location.hid ?? '';
		formError = '';
		fieldErrors = {};
		isPickingPoint = false;
		form = {
			hid: location.hid,
			name: location.name ?? '',
			address: location.address ?? '',
			category: location.category ?? '',
			child_category:
				location.category === 'bonetsky' ? (location.type ?? '') : (location.child_category ?? ''),
			manager: location.manager ?? '',
			is_partnerships: Boolean(location.is_partnerships),
			lat: String(location.lat ?? ''),
			lng: String(location.lng ?? '')
		};
	}

	async function saveLocation() {
		fieldErrors = {};
		formError = '';
		if (!form.lat || !form.lng) {
			fieldErrors = {
				lat: 'Сначала выбери точку на карте',
				lng: 'Сначала выбери точку на карте'
			};
			return;
		}
		isSaving = true;
		const method = editingHid ? 'PUT' : 'POST';
		const url = editingHid ? `/api/locations/${editingHid}` : '/api/locations';
		try {
			const token = await requestToken(url, method);
			await fetchJSON(url, {
				method,
				headers: { 'X-Resource-Token': token },
				body: JSON.stringify({
					name: form.name.trim(),
					address: form.address.trim(),
					category: form.category,
					child_category: form.child_category,
					manager: form.manager.trim(),
					is_partnerships: form.is_partnerships,
					lat: Number(form.lat),
					lng: Number(form.lng)
				})
			});
			await loadData();
			resetForm();
		} catch (err) {
			const e = err as ApiError;
			fieldErrors = e.fields ?? {};
			formError = e.message ?? 'Не удалось сохранить локацию';
		} finally {
			isSaving = false;
		}
	}

	async function deleteLocation(hid: string) {
		if (!window.confirm('Удалить эту локацию?')) return;
		const url = `/api/locations/${hid}`;
		try {
			const token = await requestToken(url, 'DELETE');
			const res = await fetch(url, {
				method: 'DELETE',
				headers: { 'X-Resource-Token': token }
			});
			if (!res.ok) throw (await res.json().catch(() => null)) ?? { message: `HTTP ${res.status}` };
			if (editingHid === hid) resetForm();
			await loadData();
		} catch (err) {
			formError = (err as ApiError).message ?? 'Не удалось удалить локацию';
		}
	}

	async function createChildCategory() {
		fieldErrors = {};
		formError = '';
		if (!form.category) {
			fieldErrors = { category: 'Сначала выбери категорию' };
			return;
		}
		const label = newChildCategoryLabel.trim();
		if (!label) {
			fieldErrors = { child_category: 'Название подкатегории обязательно' };
			return;
		}
		isCreatingChildCategory = true;
		const url = `/api/location-config/${form.category}/children`;
		try {
			const token = await requestToken(url, 'POST');
			const created = await fetchJSON<ChildCreateResponse>(url, {
				method: 'POST',
				headers: { 'X-Resource-Token': token },
				body: JSON.stringify({ label })
			});
			await loadData();
			form.child_category = created.key;
			newChildCategoryLabel = '';
			isChildPopupOpen = false;
		} catch (err) {
			const e = err as ApiError;
			fieldErrors = e.fields ?? {};
			formError = e.message ?? 'Не удалось добавить подкатегорию';
		} finally {
			isCreatingChildCategory = false;
		}
	}

	function onCategoryChange(value: string) {
		form.category = value;
		form.child_category = '';
		newChildCategoryLabel = '';
		isChildPopupOpen = false;
	}

	function onFilterCategoryChange(value: string) {
		filter.category = value;
		filter.child_category = '';
	}

	return {
		get isLoading() {
			return isLoading;
		},
		get isSaving() {
			return isSaving;
		},
		get loadError() {
			return loadError;
		},
		get formError() {
			return formError;
		},
		set formError(v: string) {
			formError = v;
		},
		get isFilterPopupOpen() {
			return isFilterPopupOpen;
		},
		set isFilterPopupOpen(v: boolean) {
			isFilterPopupOpen = v;
		},
		get isChildPopupOpen() {
			return isChildPopupOpen;
		},
		set isChildPopupOpen(v: boolean) {
			isChildPopupOpen = v;
		},
		get editingHid() {
			return editingHid;
		},
		get isPickingPoint() {
			return isPickingPoint;
		},
		set isPickingPoint(v: boolean) {
			isPickingPoint = v;
		},
		get isCreatingChildCategory() {
			return isCreatingChildCategory;
		},
		get categories() {
			return categories;
		},
		get locations() {
			return locations;
		},
		get fieldErrors() {
			return fieldErrors;
		},
		set fieldErrors(v: Record<string, string>) {
			fieldErrors = v;
		},
		get form() {
			return form;
		},
		get newChildCategoryLabel() {
			return newChildCategoryLabel;
		},
		set newChildCategoryLabel(v: string) {
			newChildCategoryLabel = v;
		},
		get filter() {
			return filter;
		},
		set filter(v: FilterState) {
			filter = v;
		},
		get activeTab() {
			return activeTab;
		},
		set activeTab(v: 'locations' | 'categories') {
			activeTab = v;
		},
		loadData,
		resetForm,
		editLocation,
		saveLocation,
		deleteLocation,
		createChildCategory,
		onCategoryChange,
		onFilterCategoryChange
	};
}

export type ManageStore = ReturnType<typeof createManageStore>;
