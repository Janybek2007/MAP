import type {
	ApiError,
	CategoryConfig,
	ChildCreateResponse,
	FilterState,
	FormState,
	LocationItem
} from './types';
import { fetchJSON, requestToken, requestTokens } from './api';
import { createEmptyForm, sortCategories, validateLocationForm } from './utils';

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
	let confirmDialog = $state({
		open: false,
		title: '',
		message: '',
		confirmLabel: 'Подтвердить',
		cancelLabel: 'Отмена',
		tone: 'danger' as 'danger' | 'default'
	});
	let confirmAction = $state<null | (() => Promise<void> | void)>(null);
	let isConfirming = $state(false);
	let toastId = 0;
	let toasts = $state<Array<{ id: number; message: string; type: 'success' | 'error' | 'info' }>>([]);

	function showToast(
		message: string,
		type: 'success' | 'error' | 'info' = 'info',
		duration = 4200
	) {
		const id = ++toastId;
		toasts = [...toasts, { id, message, type }];
		if (duration > 0) {
			window.setTimeout(() => dismissToast(id), duration);
		}
	}

	function dismissToast(id: number) {
		toasts = toasts.filter((toast) => toast.id !== id);
	}

	function openConfirm(options: {
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		tone?: 'danger' | 'default';
		onConfirm: () => Promise<void> | void;
	}) {
		confirmDialog = {
			open: true,
			title: options.title,
			message: options.message,
			confirmLabel: options.confirmLabel ?? 'Подтвердить',
			cancelLabel: options.cancelLabel ?? 'Отмена',
			tone: options.tone ?? 'danger'
		};
		confirmAction = options.onConfirm;
	}

	function closeConfirm() {
		if (isConfirming) return;
		resetConfirmState();
	}

	function resetConfirmState() {
		confirmDialog = {
			open: false,
			title: '',
			message: '',
			confirmLabel: 'Подтвердить',
			cancelLabel: 'Отмена',
			tone: 'danger'
		};
		confirmAction = null;
	}

	async function submitConfirm() {
		if (!confirmAction || isConfirming) return;
		isConfirming = true;
		try {
			await confirmAction();
			resetConfirmState();
		} finally {
			isConfirming = false;
		}
	}

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
			showToast(loadError, 'error');
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
		const clientFieldErrors = validateLocationForm(form, categories);
		if (Object.keys(clientFieldErrors).length > 0) {
			fieldErrors = clientFieldErrors;
			showToast('Проверь заполнение формы', 'error');
			return;
		}
		isSaving = true;
		const method = editingHid ? 'PUT' : 'POST';
		const url = editingHid ? `/api/locations/${editingHid}` : '/api/locations';
		const isEditing = Boolean(editingHid);
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
			showToast(isEditing ? 'Локация обновлена' : 'Локация создана', 'success');
		} catch (err) {
			const e = err as ApiError;
			fieldErrors = e.fields ?? {};
			formError = e.message ?? 'Не удалось сохранить локацию';
			if (formError) showToast(formError, 'error');
		} finally {
			isSaving = false;
		}
	}

	async function deleteLocation(hid: string) {
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
			showToast('Локация удалена', 'success');
		} catch (err) {
			formError = (err as ApiError).message ?? 'Не удалось удалить локацию';
			if (formError) showToast(formError, 'error');
		}
	}

	function confirmDeleteLocation(hid: string) {
		openConfirm({
			title: 'Удалить локацию?',
			message: 'Локация будет удалена из списка. Это действие нельзя отменить.',
			confirmLabel: 'Удалить',
			tone: 'danger',
			onConfirm: () => deleteLocation(hid)
		});
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
			showToast('Подкатегория добавлена', 'success');
		} catch (err) {
			const e = err as ApiError;
			fieldErrors = e.fields ?? {};
			formError = e.message ?? 'Не удалось добавить подкатегорию';
			if (formError) showToast(formError, 'error');
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
		get confirmDialog() {
			return confirmDialog;
		},
		get isConfirming() {
			return isConfirming;
		},
		get toasts() {
			return toasts;
		},
		loadData,
		resetForm,
		editLocation,
		saveLocation,
		deleteLocation,
		confirmDeleteLocation,
		createChildCategory,
		onCategoryChange,
		onFilterCategoryChange,
		openConfirm,
		closeConfirm,
		submitConfirm,
		showToast,
		dismissToast
	};
}

export type ManageStore = ReturnType<typeof createManageStore>;
