<script lang="ts">
	import { X } from 'lucide-svelte';
	import Popup from '$lib/components/ui/Popup.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import type { ManageStore } from '$lib/manage/store.svelte';
	import { filteredLocations, highlight, popupCoords, toSelectOptions } from '$lib/manage/utils';

	let { store }: { store: ManageStore } = $props();

	let filterPopupAnchor: HTMLDivElement | null = null;
	let filterPopupPosition = $state({ top: 0, left: 0, minWidth: 340 });

	$effect(() => {
		filterPopupPosition = popupCoords(filterPopupAnchor, 340);
	});

	function categoryOptions() {
		return toSelectOptions(store.categories);
	}

	function childOptions() {
		if (!store.filter.category) return [];
		return toSelectOptions(
			store.categories.find((c) => c.key === store.filter.category)?.children ?? []
		);
	}

	function childLabel(location: (typeof store.locations)[number]) {
		if (location.category === 'bonetsky') return location.type_display || location.type || '';
		return location.child_category_display || location.child_category || '';
	}
</script>

<div class="list-head">
	<div class="manage-toolbar" style="margin-bottom:0">
		<div bind:this={filterPopupAnchor} class="toolbar-actions">
			<button
				class="ghost-btn"
				type="button"
				onclick={(e) => {
					e.stopPropagation();
					store.isFilterPopupOpen = !store.isFilterPopupOpen;
				}}>Фильтры</button
			>
			<button class="ghost-btn" type="button" onclick={() => store.resetForm()}>
				Новая локация
			</button>
		</div>
	</div>

	<input
		class="text-input"
		bind:value={store.filter.search}
		placeholder="Поиск по имени, адресу и менеджеру"
	/>

	{#if store.filter.category || store.filter.child_category || store.filter.is_partnerships}
		<div class="active-filters">
			{#if store.filter.category}
				{@const label =
					store.categories.find((c) => c.key === store.filter.category)?.label ??
					store.filter.category}
				<span class="filter-chip">
					{label}
					<button type="button" onclick={() => store.onFilterCategoryChange('')}>
						<X size={11} strokeWidth={2.8} />
					</button>
				</span>
			{/if}
			{#if store.filter.child_category}
				{@const label =
					store.categories
						.find((c) => c.key === store.filter.category)
						?.children.find((ch) => ch.key === store.filter.child_category)?.label ??
					store.filter.child_category}
				<span class="filter-chip">
					{label}
					<button type="button" onclick={() => (store.filter.child_category = '')}>
						<X size={11} strokeWidth={2.8} />
					</button>
				</span>
			{/if}
			{#if store.filter.is_partnerships}
				<span class="filter-chip">
					Партнерство
					<button type="button" onclick={() => (store.filter.is_partnerships = false)}>
						<X size={11} strokeWidth={2.8} />
					</button>
				</span>
			{/if}
		</div>
	{/if}

	<Popup
		open={store.isFilterPopupOpen}
		top={filterPopupPosition.top}
		left={filterPopupPosition.left}
		minWidth={filterPopupPosition.minWidth}
		onclose={() => (store.isFilterPopupOpen = false)}
	>
		<label class="field">
			<span>Категория</span>
			<Select
				bind:value={store.filter.category}
				options={[{ value: '', label: 'Все категории' }, ...categoryOptions()]}
				placeholder="Все категории"
				on:change={(e) => store.onFilterCategoryChange(e.detail.value)}
			/>
		</label>
		<label class="field">
			<span>Подкатегория</span>
			<Select
				bind:value={store.filter.child_category}
				options={[{ value: '', label: 'Все подкатегории' }, ...childOptions()]}
				placeholder="Все подкатегории"
				disabled={!store.filter.category}
			/>
		</label>
		<label class="checkbox-inline checkbox-inline-filter">
			<input type="checkbox" bind:checked={store.filter.is_partnerships} />
			<span>Партнерство</span>
		</label>
		<div class="popup-actions">
			<button
				class="ghost-btn"
				type="button"
				onclick={() =>
					(store.filter = {
						search: store.filter.search,
						category: '',
						child_category: '',
						is_partnerships: false
					})}>Сбросить</button
			>
		</div>
	</Popup>
</div>

<div class="list-body">
	{#if store.isLoading}
		<div class="manage-empty">Загрузка...</div>
	{:else if filteredLocations(store.locations, store.filter).length === 0}
		<div class="manage-empty">Локации не найдены</div>
	{:else}
		<div class="locations-table">
			{#each filteredLocations(store.locations, store.filter) as location}
				<div class="location-row">
					<div class="location-main">
						<div class="location-title">{@html highlight(location.name, store.filter.search)}</div>
						<div class="location-meta">
							<span>{location.category_display || location.category}</span>
							{#if childLabel(location)}<span>· {childLabel(location)}</span>{/if}
						</div>
						{#if location.address}
							<div class="location-address">
								{@html highlight(location.address, store.filter.search)}
							</div>
						{/if}
						{#if location.manager}
							<div class="location-manager">
								{@html highlight(location.manager, store.filter.search)}
							</div>
						{/if}
					</div>
					<div class="location-actions">
						<button class="ghost-btn" type="button" onclick={() => store.editLocation(location)}>
							Изменить
						</button>
						<button
							class="danger-btn"
							type="button"
							onclick={() => store.deleteLocation(location.hid ?? '')}>Удалить</button
						>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
