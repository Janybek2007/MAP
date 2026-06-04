<script lang="ts">
	import { Check, MapPinned } from 'lucide-svelte';
	import { mapStore } from '../../store/mapStore';
	import { filteredCities, isCityChecked, isRegionChecked } from './helpers';

	let open = $state(true);
</script>

<div class="geo-panel {open ? 'open shown' : ''}">
	<div class="geo-panel-header">
		<button
			class="filter-toggle-btn"
			type="button"
			aria-label="Открыть или закрыть локацию"
			onclick={() => (open = !open)}
		>
			<MapPinned size={16} strokeWidth={2.1} />
		</button>
		<span class="filter-title">Локация</span>
	</div>
	{#if open}
		<div class="geo-panel-body custom-scrollbar">
			<div class="panel-section-title">Область</div>
			<div class="stack stack-tight">
				{#each $mapStore.regions as region}
					<button
						type="button"
						class="district-btn {isRegionChecked($mapStore, region.hid) ? 'active' : ''}"
						style="--d-color:#2563eb"
						onclick={() => region.hid && mapStore.toggleRegionHid(region.hid)}
					>
						<span class="district-checkbox">
							<Check class="district-checkbox-tick" size={10} strokeWidth={2.6} />
						</span>
						<span>{region.title}</span>
					</button>
				{/each}
			</div>
			<div class="panel-divider"></div>
			<div class="panel-section-title">Город</div>
			<div class="stack stack-tight">
				{#each filteredCities($mapStore) as city}
					<button
						type="button"
						class="district-btn {isCityChecked($mapStore, city.hid) ? 'active' : ''}"
						style="--d-color:#16a34a"
						onclick={() => city.hid && mapStore.toggleCityHid(city.hid)}
					>
						<span class="district-checkbox">
							<Check class="district-checkbox-tick" size={10} strokeWidth={2.6} />
						</span>
						<span>{city.title}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
