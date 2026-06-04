<script lang="ts">
	import { Check } from 'lucide-svelte';
	import { mapStore } from '../../../store/mapStore';
	import { districtGroups, filteredDistricts } from '../helpers';

	let { onToggleDistrict }: { onToggleDistrict: (districtId: number) => void } = $props();

	function isDistrictActive(id: number) {
		return $mapStore.activeDistricts.some((district) => district.id === id);
	}

	function districtTree() {
		const items = filteredDistricts($mapStore);
		const regionMap = new Map<string, { title: string; districts: typeof items }>();

		for (const district of items) {
			const regionHid = district.region_hid || 'unknown';
			const region = $mapStore.regions.find((r) => r.hid === regionHid);
			const regionTitle = region?.title || 'Область';
			if (!regionMap.has(regionHid)) {
				regionMap.set(regionHid, { title: regionTitle, districts: [] });
			}
			regionMap.get(regionHid)!.districts.push(district);
		}

		return Array.from(regionMap.values())
			.map((regionNode) => ({
				title: regionNode.title,
				districts: regionNode.districts.sort((a, b) =>
					String(a.title || '').localeCompare(String(b.title || ''), 'ru')
				)
			}))
			.sort((a, b) => a.title.localeCompare(b.title, 'ru'));
	}
</script>

<div class="panel-pad">
	<div class="panel-section-title">Районы</div>
	<div class="district-list stack stack-tight custom-scrollbar">
		{#if $mapStore.selectedRegionHids.length === 0 && $mapStore.selectedCityHids.length === 0}
			{#each districtTree() as regionNode}
				<div class="district-group-title district-group-title-region">{regionNode.title}</div>
				{#each regionNode.districts as district}
					{@const idx = $mapStore.districts.findIndex(
						(item) => item.hid && district.hid && item.hid === district.hid
					)}
					<button
						type="button"
						class="district-btn district-btn-nested {idx >= 0 && isDistrictActive(idx)
							? 'active'
							: ''}"
						style="--d-color:#6366f1"
						onclick={() => idx >= 0 && onToggleDistrict(idx)}
					>
						<span class="district-checkbox">
							<Check class="district-checkbox-tick" size={10} strokeWidth={2.6} />
						</span>
						<span>{String(district.title || '').split(',')[0]}</span>
					</button>
				{/each}
			{/each}
		{:else}
			{#each districtGroups($mapStore) as group}
				{#if group.title}
					<div class="district-group-title">{group.title}</div>
				{/if}
				{#each group.districts as district}
					{@const idx = $mapStore.districts.findIndex(
						(item) => item.hid && district.hid && item.hid === district.hid
					)}
					<button
						type="button"
						class="district-btn {idx >= 0 && isDistrictActive(idx) ? 'active' : ''}"
						style="--d-color:#6366f1"
						onclick={() => idx >= 0 && onToggleDistrict(idx)}
					>
						<span class="district-checkbox">
							<Check class="district-checkbox-tick" size={10} strokeWidth={2.6} />
						</span>
						<span>{String(district.title || '').split(',')[0]}</span>
					</button>
				{/each}
			{/each}
		{/if}
	</div>
</div>
