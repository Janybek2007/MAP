<script lang="ts">
	import type { ManageStore } from '$lib/manage/store.svelte';
	import { fetchJSON, requestToken } from '$lib/manage/api';
	import type { ApiError } from '$lib/manage/types';

	let { store }: { store: ManageStore } = $props();

	let subEditKey = $state('');
	let subEditLabel = $state('');
	let subAddCategory = $state('');
	let subNewLabel = $state('');
	let subUpdating = $state(false);
	let subDeleting = $state(false);
	let subAdding = $state(false);

	async function subUpdate(categoryKey: string, childKey: string) {
		const label = subEditLabel.trim();
		if (!label) return;
		subUpdating = true;
		const url = `/api/location-config/${categoryKey}/children/${childKey}`;
		try {
			const token = await requestToken(url, 'PUT');
			await fetchJSON(url, {
				method: 'PUT',
				headers: { 'X-Resource-Token': token },
				body: JSON.stringify({ label })
			});
			await store.loadData();
			subEditKey = '';
			subEditLabel = '';
			store.showToast('Подкатегория обновлена', 'success');
		} catch (err) {
			store.showToast((err as ApiError).message ?? 'Не удалось обновить', 'error');
		} finally {
			subUpdating = false;
		}
	}

	async function subDelete(categoryKey: string, childKey: string) {
		subDeleting = true;
		const url = `/api/location-config/${categoryKey}/children/${childKey}`;
		try {
			const token = await requestToken(url, 'DELETE');
			const res = await fetch(url, {
				method: 'DELETE',
				headers: { 'X-Resource-Token': token }
			});
			if (!res.ok) throw (await res.json().catch(() => null)) ?? { message: `HTTP ${res.status}` };
			await store.loadData();
			store.showToast('Подкатегория удалена', 'success');
		} catch (err) {
			store.showToast((err as ApiError).message ?? 'Не удалось удалить', 'error');
		} finally {
			subDeleting = false;
		}
	}

	function confirmSubDelete(categoryKey: string, childKey: string, childLabel: string) {
		store.openConfirm({
			title: 'Удалить подкатегорию?',
			message: `Подкатегория "${childLabel}" будет удалена. Это действие нельзя отменить.`,
			confirmLabel: 'Удалить',
			tone: 'danger',
			onConfirm: () => subDelete(categoryKey, childKey)
		});
	}

	async function subCreate(categoryKey: string) {
		const label = subNewLabel.trim();
		if (!label) return;
		subAdding = true;
		const url = `/api/location-config/${categoryKey}/children`;
		try {
			const token = await requestToken(url, 'POST');
			await fetchJSON(url, {
				method: 'POST',
				headers: { 'X-Resource-Token': token },
				body: JSON.stringify({ label })
			});
			await store.loadData();
			subNewLabel = '';
			subAddCategory = '';
			store.showToast('Подкатегория добавлена', 'success');
		} catch (err) {
			store.showToast((err as ApiError).message ?? 'Не удалось добавить', 'error');
		} finally {
			subAdding = false;
		}
	}
</script>

<div class="list-body">
	{#if store.isLoading}
		<div class="manage-empty">Загрузка...</div>
	{:else if store.categories.length === 0}
		<div class="manage-empty">Категории не найдены</div>
	{:else}
		<div class="sub-list">
			{#each store.categories as category}
				<div class="sub-group">
					<div class="sub-group-title">{category.label}</div>
					{#if category.children.length === 0}
						<div class="sub-empty">Нет подкатегорий</div>
					{:else}
						{#each category.children as child}
							{@const editKey = `${category.key}:${child.key}`}
							<div class="sub-row">
								{#if subEditKey === editKey}
									<input
										class="text-input sub-input"
										bind:value={subEditLabel}
										onkeydown={(e) => {
											if (e.key === 'Enter') subUpdate(category.key, child.key);
											if (e.key === 'Escape') {
												subEditKey = '';
												subEditLabel = '';
											}
										}}
									/>
									<div class="sub-actions">
										<button
											class="ghost-btn"
											type="button"
											disabled={subUpdating}
											onclick={() => subUpdate(category.key, child.key)}
											>{subUpdating ? '...' : 'Сохранить'}</button
										>
										<button
											class="ghost-btn"
											type="button"
											onclick={() => {
												subEditKey = '';
												subEditLabel = '';
											}}>Отмена</button
										>
									</div>
								{:else}
									<span class="sub-label">{child.label}</span>
									<div class="sub-actions">
										<button
											class="ghost-btn"
											type="button"
											onclick={() => {
												subEditKey = editKey;
												subEditLabel = child.label;
											}}>Изменить</button
										>
										<button
											class="danger-btn"
											type="button"
											disabled={subDeleting}
											onclick={() => confirmSubDelete(category.key, child.key, child.label)}
											>Удалить</button
										>
									</div>
								{/if}
							</div>
						{/each}
					{/if}

					{#if subAddCategory === category.key}
						<div class="sub-row sub-add-row">
							<input
								class="text-input sub-input"
								bind:value={subNewLabel}
								placeholder="Название подкатегории"
								onkeydown={(e) => {
									if (e.key === 'Enter') subCreate(category.key);
									if (e.key === 'Escape') {
										subAddCategory = '';
										subNewLabel = '';
									}
								}}
							/>
							<div class="sub-actions">
								<button
									class="ghost-btn"
									type="button"
									disabled={subAdding}
									onclick={() => subCreate(category.key)}>{subAdding ? '...' : 'Добавить'}</button
								>
								<button
									class="ghost-btn"
									type="button"
									onclick={() => {
										subAddCategory = '';
										subNewLabel = '';
									}}>Отмена</button
								>
							</div>
						</div>
					{:else}
						<button
							class="ghost-btn sub-add-btn"
							type="button"
							onclick={() => {
								subAddCategory = category.key;
								subNewLabel = '';
							}}>+ Добавить</button
						>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
