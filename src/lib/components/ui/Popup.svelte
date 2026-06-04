<script lang="ts">
	import { onMount, tick, type Snippet } from 'svelte';

	let {
		open = false,
		className = '',
		top = 0,
		left = 0,
		minWidth = 280,
		onclose,
		children
	}: {
		open?: boolean;
		className?: string;
		top?: number;
		left?: number;
		minWidth?: number;
		onclose?: () => void;
		children?: Snippet;
	} = $props();

	let popupEl = $state<HTMLDivElement | null>(null);
	let adjustedTop = $state(0);
	let adjustedLeft = $state(0);

	function requestClose() {
		onclose?.();
	}

	async function reposition() {
		if (!open || !popupEl || typeof window === 'undefined') return;

		await tick();
		const rect = popupEl.getBoundingClientRect();
		const parentRect =
			popupEl.offsetParent instanceof HTMLElement
				? popupEl.offsetParent.getBoundingClientRect()
				: { left: 0, top: 0 };

		let nextLeft = left;
		let nextTop = top;

		const rightOverflow = rect.right - window.innerWidth;
		if (rightOverflow > 0) {
			nextLeft = Math.max(8, left - rightOverflow - 8);
		}

		const leftViewport = parentRect.left + nextLeft;
		if (leftViewport < 8) {
			nextLeft += 8 - leftViewport;
		}

		const bottomOverflow = rect.bottom - window.innerHeight;
		if (bottomOverflow > 0) {
			nextTop = Math.max(8, top - rect.height - 16);
		}

		const topViewport = parentRect.top + nextTop;
		if (topViewport < 8) {
			nextTop += 8 - topViewport;
		}

		adjustedLeft = nextLeft;
		adjustedTop = nextTop;
	}

	function handleDocumentClick(event: MouseEvent) {
		if (!open || !popupEl) return;
		if (popupEl.contains(event.target as Node)) return;
		requestClose();
	}

	function handleDocumentKeydown(event: KeyboardEvent) {
		if (!open) return;
		if (event.key === 'Escape') requestClose();
	}

	$effect(() => {
		if (!open) return;
		adjustedTop = top;
		adjustedLeft = left;
		void reposition();
	});

	onMount(() => {
		document.addEventListener('click', handleDocumentClick);
		document.addEventListener('keydown', handleDocumentKeydown);
		window.addEventListener('resize', reposition);

		return () => {
			document.removeEventListener('click', handleDocumentClick);
			document.removeEventListener('keydown', handleDocumentKeydown);
			window.removeEventListener('resize', reposition);
		};
	});
</script>

{#if open}
	<div
		class={`popup-layer ${className}`.trim()}
		style={`top:${adjustedTop}px; left:${adjustedLeft}px; min-width:${minWidth}px;`}
		bind:this={popupEl}
	>
		<div class="popup-card">
			{@render children?.()}
		</div>
	</div>
{/if}

<style>
	.popup-layer {
		position: absolute;
		z-index: 50;
	}

	.popup-card {
		display: grid;
		gap: 10px;
		padding: 12px;
		border: 1px solid rgba(148, 163, 184, 0.25);
		border-radius: 10px;
		background: #fff;
		box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
	}
</style>
