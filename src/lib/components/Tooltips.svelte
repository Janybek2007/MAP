<script lang="ts">
	let {
		onReady
	}: {
		onReady: (refs: {
			pinTooltipEl: HTMLDivElement;
			pinTooltipTitleEl: HTMLDivElement;
			pinTooltipCatEl: HTMLDivElement;
			districtTooltipEl: HTMLDivElement;
		}) => void;
	} = $props();

	let pinTooltipEl: HTMLDivElement;
	let pinTooltipTitleEl: HTMLDivElement;
	let pinTooltipCatEl: HTMLDivElement;
	let districtTooltipEl: HTMLDivElement;

	$effect(() => {
		if (pinTooltipEl && pinTooltipTitleEl && pinTooltipCatEl && districtTooltipEl) {
			onReady({ pinTooltipEl, pinTooltipTitleEl, pinTooltipCatEl, districtTooltipEl });
		}
	});
</script>

<div bind:this={districtTooltipEl} class="district-tooltip"></div>

<div bind:this={pinTooltipEl} class="pin-tooltip">
	<div class="tooltip-stack">
		<div bind:this={pinTooltipTitleEl} class="tooltip-title"></div>
		<div bind:this={pinTooltipCatEl} class="tooltip-meta"></div>
	</div>
</div>
