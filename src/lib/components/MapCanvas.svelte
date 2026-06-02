<script lang="ts">
	let { onReady }: { onReady: (el: HTMLDivElement) => void } = $props();
	let element: HTMLDivElement;

	$effect(() => {
		if (element) onReady(element);
	});
</script>

<div bind:this={element} class="map-canvas"></div>
