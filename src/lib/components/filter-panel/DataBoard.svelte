<script lang="ts">
	import { mapStore } from '../../store/mapStore';
	import { formatPopulation, selectedPopulation } from './helpers';

	let open = $state(true);
</script>

<div class="stats-panel {$mapStore.isFilterOpen ? 'with-filter' : ''} {open ? 'open' : 'closed'}">
	<button
		type="button"
		class="stats-handle"
		aria-label="Открыть или закрыть data board"
		onclick={() => (open = !open)}
	>
		<svg width="14" height="14" viewBox="0 0 12 12" fill="none">
			{#if open}
				<path
					d="M7.5 2L3.5 6l4 4"
					stroke="currentColor"
					stroke-width="1.8"
					stroke-linecap="round"
					stroke-linejoin="round"
				></path>
			{:else}
				<path
					d="M4.5 2l4 4-4 4"
					stroke="currentColor"
					stroke-width="1.8"
					stroke-linecap="round"
					stroke-linejoin="round"
				></path>
			{/if}
		</svg>
	</button>
	{#if open}
		<div class="stats-content">
			<div class="stats-grid headers">
				<div class="stats-cell stats-label">Популяция</div>
				<div class="stats-cell stats-label">Филиалы</div>
				<div class="stats-cell stats-label">ГОС</div>
				<div class="stats-cell stats-label">Конкуренты</div>
				<div class="stats-cell stats-label">Частные клиники</div>
			</div>
			<div class="stats-grid values">
				<div class="stats-cell">{formatPopulation(selectedPopulation($mapStore))}</div>
				<div class="stats-cell">{$mapStore.filteredCounts.bonetsky}</div>
				<div class="stats-cell">{$mapStore.filteredCounts.gos}</div>
				<div class="stats-cell">{$mapStore.filteredCounts.rival}</div>
				<div class="stats-cell">{$mapStore.filteredCounts.chastnyi}</div>
			</div>
		</div>
	{/if}
</div>
