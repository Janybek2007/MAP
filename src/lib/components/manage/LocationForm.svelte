<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus } from 'lucide-svelte';
	import Popup from '$lib/components/ui/Popup.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import type { ManageStore } from '$lib/manage/store.svelte';
	import { popupCoords, toSelectOptions } from '$lib/manage/utils';

	let { store }: { store: ManageStore } = $props();

	const defaultCenter: [number, number] = [42.8746, 74.5698];

	let childPopupAnchor: HTMLDivElement | null = null;
	let childPopupPosition = $state({ top: 0, left: 0, minWidth: 420 });

	let mapHost: HTMLDivElement | null = null;
	let mapInstance: any = null;
	let pointMarker: any = null;

	function categoryOptions() {
		return toSelectOptions(store.categories);
	}

	function childOptions() {
		return toSelectOptions(
			store.categories.find((category) => category.key === store.form.category)?.children ?? []
		);
	}

	function renderPointMarker() {
		if (!mapInstance) return;

		const lat = Number(store.form.lat);
		const lng = Number(store.form.lng);
		const hasPoint =
			Number.isFinite(lat) &&
			Number.isFinite(lng) &&
			store.form.lat !== '' &&
			store.form.lng !== '';

		if (!hasPoint) {
			if (pointMarker && mapInstance.hasLayer(pointMarker)) {
				mapInstance.removeLayer(pointMarker);
			}
			pointMarker = null;
			return;
		}

		if (!pointMarker) {
			pointMarker = (window as any).DG.marker([lat, lng]).addTo(mapInstance);
		} else {
			pointMarker.setLatLng([lat, lng]);
			if (!mapInstance.hasLayer(pointMarker)) {
				pointMarker.addTo(mapInstance);
			}
		}

		mapInstance.setView([lat, lng], Math.max(mapInstance.getZoom?.() || 13, 14));
	}

	async function initMap() {
		if (!mapHost || mapInstance) return;

		const DG = (window as any).DG;
		if (!DG) return;

		if (typeof DG.then === 'function') {
			await new Promise<void>((resolve, reject) =>
				DG.then(
					() => resolve(),
					(error: unknown) => reject(error)
				)
			);
		}

		mapInstance = DG.map(mapHost, {
			center: defaultCenter,
			zoom: 12,
			fullscreenControl: false,
			zoomControl: true
		});

		mapInstance.on('click', (event: any) => {
			if (!store.isPickingPoint) return;

			const lat = Number(event.latlng?.lat ?? event.latLng?.lat ?? 0);
			const lng = Number(event.latlng?.lng ?? event.latLng?.lng ?? 0);

			store.form.lat = String(lat);
			store.form.lng = String(lng);
			store.fieldErrors = {
				...store.fieldErrors,
				lat: '',
				lng: ''
			};
			store.isPickingPoint = false;
			renderPointMarker();
		});

		renderPointMarker();
	}

	$effect(() => {
		childPopupPosition = popupCoords(childPopupAnchor, 420);
	});

	$effect(() => {
		store.form.lat;
		store.form.lng;
		renderPointMarker();
	});

	onMount(() => {
		void initMap();

		return () => {
			if (mapInstance) {
				mapInstance.remove();
				mapInstance = null;
				pointMarker = null;
			}
		};
	});
</script>

<section class="manage-card manage-form-card">
	<div class="form-scroll custom-scrollbar">
		<div class="manage-toolbar">
			<h2>{store.editingHid ? 'Редактирование' : 'Добавление'}</h2>
			{#if store.editingHid}
				<div class="editing-badge">ID: {store.editingHid}</div>
			{/if}
		</div>

		<div class="map-panel">
			<div class="map-actions">
				<button
					class:is-picking={store.isPickingPoint}
					class="primary-btn"
					type="button"
					onclick={() => (store.isPickingPoint = !store.isPickingPoint)}
				>
					{store.isPickingPoint ? 'Отменить выбор точки' : 'Выбрать точку'}
				</button>
				<div class="map-hint">
					{#if store.form.lat && store.form.lng}
						Точка выбрана: {store.form.lat}, {store.form.lng}
					{:else}
						Сначала выбери точку на карте
					{/if}
				</div>
			</div>

			<div bind:this={mapHost} class="manage-map"></div>
		</div>

		<div class="form-grid">
			<label class="field">
				<span>Название</span>
				<input class="text-input" bind:value={store.form.name} placeholder="Название локации" />
				{#if store.fieldErrors.name}
					<small>{store.fieldErrors.name}</small>
				{/if}
			</label>

			<label class="field">
				<span>Адрес</span>
				<input class="text-input" bind:value={store.form.address} placeholder="Адрес" />
				{#if store.fieldErrors.address}
					<small>{store.fieldErrors.address}</small>
				{/if}
			</label>

			<label class="field">
				<span>Категория</span>
				<Select
					bind:value={store.form.category}
					options={[{ value: '', label: 'Выбери категорию' }, ...categoryOptions()]}
					placeholder="Выбери категорию"
					on:change={(event) => store.onCategoryChange(event.detail.value)}
				/>
				{#if store.fieldErrors.category}
					<small>{store.fieldErrors.category}</small>
				{/if}
			</label>

			<label class="field">
				<span>Подкатегория</span>
				<div bind:this={childPopupAnchor} class="inline-action">
					<Select
						bind:value={store.form.child_category}
						options={[{ value: '', label: 'Выбери подкатегорию' }, ...childOptions()]}
						placeholder="Выбери подкатегорию"
						disabled={!store.form.category}
					/>
					<button
						class="ghost-btn icon-btn"
						type="button"
						disabled={!store.form.category}
						onclick={(event) => {
							event.stopPropagation();
							store.isChildPopupOpen = !store.isChildPopupOpen;
						}}
					>
						<Plus size={18} strokeWidth={2.4} />
					</button>
				</div>
				{#if store.fieldErrors.child_category}
					<small>{store.fieldErrors.child_category}</small>
				{/if}
			</label>

			<Popup
				open={store.isChildPopupOpen}
				top={childPopupPosition.top}
				left={childPopupPosition.left}
				minWidth={childPopupPosition.minWidth}
				className="child-popup-floating"
				onclose={() => (store.isChildPopupOpen = false)}
			>
				<label class="field">
					<span>Новая подкатегория</span>
					<input
						class="text-input"
						bind:value={store.newChildCategoryLabel}
						disabled={!store.form.category || store.isCreatingChildCategory}
						placeholder="Например, Эндокринология"
					/>
				</label>
				<div class="popup-actions">
					<button class="ghost-btn" type="button" onclick={() => (store.isChildPopupOpen = false)}>
						Закрыть
					</button>
					<button
						class="primary-btn"
						type="button"
						disabled={!store.form.category || store.isCreatingChildCategory}
						onclick={() => store.createChildCategory()}
					>
						{store.isCreatingChildCategory ? 'Добавление...' : 'Добавить'}
					</button>
				</div>
			</Popup>

			<label class="field">
				<span>Менеджер</span>
				<input class="text-input" bind:value={store.form.manager} placeholder="Менеджер" />
			</label>

			<label class="field">
				<span>Координаты</span>
				<div class="coord-grid">
					<input class="text-input" bind:value={store.form.lat} placeholder="lat" readonly />
					<input class="text-input" bind:value={store.form.lng} placeholder="lng" readonly />
				</div>
				{#if store.fieldErrors.lat || store.fieldErrors.lng}
					<small>{store.fieldErrors.lat || store.fieldErrors.lng}</small>
				{/if}
			</label>

			<label class="field checkbox-field">
				<input type="checkbox" bind:checked={store.form.is_partnerships} />
				<span>Партнерство</span>
			</label>
		</div>

		<div class="form-actions">
			<button
				class="primary-btn"
				type="button"
				disabled={store.isSaving}
				onclick={() => store.saveLocation()}
			>
				{store.isSaving
					? 'Сохранение...'
					: store.editingHid
						? 'Сохранить изменения'
						: 'Создать локацию'}
			</button>
			<button class="ghost-btn" type="button" onclick={() => store.resetForm()}>Сбросить</button>
		</div>
	</div>
</section>
