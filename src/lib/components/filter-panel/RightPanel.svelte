<script lang="ts">
	import { SlidersHorizontal } from 'lucide-svelte';
	import { mapStore } from '../../store/mapStore';
	import Categories from './right/Categories.svelte';
	import Districts from './right/Districts.svelte';

	let { onToggleDistrict }: { onToggleDistrict: (districtId: number) => void } = $props();
</script>

{#if !$mapStore.isFilterOpen}
	<button
		type="button"
		class="filter-launcher shown"
		aria-label="Открыть фильтры"
		onclick={() => mapStore.setPanelOpen(true)}
	>
		<SlidersHorizontal size={16} strokeWidth={2.2} />
	</button>
{:else}
	<div class="filter-panel open shown">
		<div class="filter-header">
			<span class="filter-title">Фильтры</span>
			<button
				class="filter-toggle-btn filter-toggle-btn-right"
				type="button"
				aria-label="Закрыть фильтры"
				onclick={() => mapStore.setPanelOpen(false)}
			>
				<SlidersHorizontal size={16} strokeWidth={2.2} />
			</button>
		</div>

		<div class="filter-body custom-scrollbar">
			<Categories />
			<div class="panel-divider panel-divider-wide"></div>
			<Districts {onToggleDistrict} />
		</div>
	</div>
{/if}
