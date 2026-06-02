<script lang="ts">
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
			<svg width="16" height="16" viewBox="0 0 14 14" fill="none">
				<path
					d="M1 2.5h12M3 7h8M5.5 11.5h3"
					stroke="currentColor"
					stroke-width="1.7"
					stroke-linecap="round"
				></path>
			</svg>
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
							<svg viewBox="0 0 10 10" class="district-checkbox-tick">
								<polyline
									points="1.5,5 4,7.5 8.5,2"
									stroke="#fff"
									stroke-width="1.8"
									fill="none"
									stroke-linecap="round"
									stroke-linejoin="round"
								></polyline>
							</svg>
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
							<svg viewBox="0 0 10 10" class="district-checkbox-tick">
								<polyline
									points="1.5,5 4,7.5 8.5,2"
									stroke="#fff"
									stroke-width="1.8"
									fill="none"
									stroke-linecap="round"
									stroke-linejoin="round"
								></polyline>
							</svg>
						</span>
						<span>{city.title}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
