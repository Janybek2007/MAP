<script lang="ts">
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';
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
		{#if open}
			<ChevronLeft size={14} strokeWidth={2.4} />
		{:else}
			<ChevronRight size={14} strokeWidth={2.4} />
		{/if}
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
