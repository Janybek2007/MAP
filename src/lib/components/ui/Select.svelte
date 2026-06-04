<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { ChevronDown } from 'lucide-svelte';
	import type { SelectOption } from './select';

	export let value = '';
	export let options: SelectOption[] = [];
	export let placeholder = 'Выбери значение';
	export let disabled = false;

	const dispatch = createEventDispatcher<{ change: { value: string } }>();

	let isOpen = false;
	let rootEl: HTMLDivElement | null = null;

	$: selectedOption = options.find((option) => option.value === value) || null;

	function toggleOpen() {
		if (disabled) return;
		isOpen = !isOpen;
	}

	function selectOption(nextValue: string) {
		value = nextValue;
		isOpen = false;
		dispatch('change', { value: nextValue });
	}

	function closeOnOutsideClick(event: MouseEvent) {
		if (!rootEl) return;
		if (rootEl.contains(event.target as Node)) return;
		isOpen = false;
	}

	onMount(() => {
		document.addEventListener('click', closeOnOutsideClick);
		return () => {
			document.removeEventListener('click', closeOnOutsideClick);
		};
	});
</script>

<div class:disabled class="custom-select" bind:this={rootEl}>
	<button
		class:open={isOpen}
		class="custom-select-trigger"
		type="button"
		{disabled}
		onclick={toggleOpen}
	>
		<span class:selected={!!selectedOption} class="custom-select-label">
			{selectedOption?.label || placeholder}
		</span>
		<span class={`custom-select-arrow ${isOpen ? 'open' : ''}`}>
			<ChevronDown size={18} strokeWidth={2.25} />
		</span>
	</button>

	{#if isOpen}
		<div class="custom-select-menu">
			{#each options as option}
				<button
					class:active={option.value === value}
					class="custom-select-option"
					type="button"
					onclick={() => selectOption(option.value)}
				>
					{option.label}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.custom-select {
		position: relative;
		width: 100%;
	}

	.custom-select.disabled {
		opacity: 0.7;
	}

	.custom-select-trigger {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		width: 100%;
		height: 44px;
		padding: 0 12px;
		border: 1px solid rgba(148, 163, 184, 0.35);
		border-radius: 8px;
		background: #fff;
		color: #0f172a;
		cursor: pointer;
		transition: 0.15s ease;
	}

	.custom-select-trigger.open {
		border-color: #94a3b8;
		box-shadow: none;
	}

	.custom-select-trigger:disabled {
		background: #f8fafc;
		color: #94a3b8;
		cursor: not-allowed;
	}

	.custom-select-label {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #94a3b8;
	}

	.custom-select-label.selected {
		color: #0f172a;
	}

	.custom-select-arrow {
		flex: 0 0 auto;
		color: #64748b;
		transition: transform 0.15s ease;
	}

	.custom-select-arrow.open {
		transform: rotate(180deg);
	}

	.custom-select-menu {
		position: absolute;
		top: calc(100% + 6px);
		left: 0;
		right: 0;
		z-index: 30;
		display: grid;
		gap: 2px;
		max-height: 260px;
		overflow-y: auto;
		padding: 6px;
		border: 1px solid rgba(148, 163, 184, 0.2);
		border-radius: 10px;
		background: #fff;
		box-shadow: none;
	}

	.custom-select-option {
		padding: 9px 10px;
		border: none;
		border-radius: 8px;
		background: transparent;
		color: #0f172a;
		text-align: left;
		font-size: 13px;
		cursor: pointer;
		transition: 0.15s ease;
	}

	.custom-select-option:hover {
		background: #f8fafc;
	}

	.custom-select-option.active {
		background: #f1f5f9;
		color: #0f172a;
		font-weight: 600;
	}
</style>
