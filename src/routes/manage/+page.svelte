<script lang="ts">
	import { onMount } from 'svelte';
	import CategoryList from '$lib/components/manage/CategoryList.svelte';
	import LocationForm from '$lib/components/manage/LocationForm.svelte';
	import LocationList from '$lib/components/manage/LocationList.svelte';
	import ConfirmModal from '$lib/components/ui/ConfirmModal.svelte';
	import Toast from '$lib/components/ui/Toast.svelte';
	import { createManageStore } from '$lib/manage/store.svelte';
	import './manage.css';

	const store = createManageStore();
	const manageReloadKey = 'map:manage:reload-once';
	const reloadDelayMs = 1;

	onMount(() => {
		if (!sessionStorage.getItem(manageReloadKey)) {
			sessionStorage.setItem(manageReloadKey, '1');
			const reloadTimer = window.setTimeout(() => location.reload(), reloadDelayMs);
			return () => window.clearTimeout(reloadTimer);
		}

		void store.loadData();
	});
</script>

<svelte:head>
	<title>Управление локациями</title>
</svelte:head>

<div class="manage-page">
	<div class="manage-layout">
		<section class="manage-card manage-list-card">
			<div class="tab-bar">
				<button
					class="tab-btn {store.activeTab === 'locations' ? 'active' : ''}"
					type="button"
					onclick={() => (store.activeTab = 'locations')}
				>
					Список локаций
				</button>
				<button
					class="tab-btn {store.activeTab === 'categories' ? 'active' : ''}"
					type="button"
					onclick={() => (store.activeTab = 'categories')}
				>
					Список подкатегорий
				</button>
			</div>

			{#if store.activeTab === 'locations'}
				<LocationList {store} />
			{:else}
				<CategoryList {store} />
			{/if}
		</section>

		<LocationForm {store} />
	</div>
</div>

<ConfirmModal
	open={store.confirmDialog.open}
	title={store.confirmDialog.title}
	message={store.confirmDialog.message}
	confirmLabel={store.confirmDialog.confirmLabel}
	cancelLabel={store.confirmDialog.cancelLabel}
	tone={store.confirmDialog.tone}
	loading={store.isConfirming}
	onconfirm={() => store.submitConfirm()}
	onclose={() => store.closeConfirm()}
/>

<Toast items={store.toasts} onclose={(id) => store.dismissToast(id)} />
