<script lang="ts">
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
		<svg width="16" height="16" viewBox="0 0 14 14" fill="none">
			<path
				d="M1 2.5h12M3 7h8M5.5 11.5h3"
				stroke="currentColor"
				stroke-width="1.7"
				stroke-linecap="round"
			></path>
		</svg>
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
				<svg width="16" height="16" viewBox="0 0 14 14" fill="none">
					<path
						d="M1 2.5h12M3 7h8M5.5 11.5h3"
						stroke="currentColor"
						stroke-width="1.7"
						stroke-linecap="round"
					></path>
				</svg>
			</button>
		</div>

		<div class="filter-body custom-scrollbar">
			<Categories />
			<div class="panel-divider panel-divider-wide"></div>
			<Districts {onToggleDistrict} />
		</div>
	</div>
{/if}
