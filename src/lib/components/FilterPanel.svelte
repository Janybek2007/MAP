<script lang="ts">
	import DataBoard from './filter-panel/DataBoard.svelte';
	import GeoPanel from './filter-panel/GeoPanel.svelte';
	import RightPanel from './filter-panel/RightPanel.svelte';

	let { onToggleDistrict }: { onToggleDistrict: (districtId: number) => void } = $props();
</script>

<DataBoard />
<GeoPanel />
<RightPanel {onToggleDistrict} />
