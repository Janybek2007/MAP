<script lang="ts">
	import { Check, CircleAlert, X } from 'lucide-svelte';

	type ToastItem = {
		id: number;
		message: string;
		type: 'success' | 'error' | 'info';
	};

	let {
		items = [],
		onclose
	}: {
		items?: ToastItem[];
		onclose?: (id: number) => void;
	} = $props();

	function iconFor(type: ToastItem['type']) {
		return type === 'success' ? Check : CircleAlert;
	}
</script>

{#if items.length > 0}
	<div class="toast-stack" aria-live="polite" aria-atomic="true">
		{#each items as item (item.id)}
			{@const Icon = iconFor(item.type)}
			<div class={`toast-item ${item.type}`}>
				<div class="toast-icon">
					<Icon size={16} strokeWidth={2.4} />
				</div>
				<div class="toast-message">{item.message}</div>
				<button class="toast-close" type="button" onclick={() => onclose?.(item.id)}>
					<X size={14} strokeWidth={2.6} />
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	.toast-stack {
		position: fixed;
		left: 50%;
		bottom: 20px;
		z-index: 130;
		display: grid;
		gap: 10px;
		width: min(calc(100vw - 24px), 460px);
		transform: translateX(-50%);
		pointer-events: none;
	}

	.toast-item {
		display: grid;
		grid-template-columns: auto 1fr auto;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		border: 1px solid rgba(148, 163, 184, 0.22);
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 16px 40px rgba(15, 23, 42, 0.14);
		backdrop-filter: blur(12px);
		pointer-events: auto;
	}

	.toast-item.success {
		border-color: rgba(22, 163, 74, 0.24);
	}

	.toast-item.error {
		border-color: rgba(220, 38, 38, 0.22);
	}

	.toast-item.info {
		border-color: rgba(59, 130, 246, 0.22);
	}

	.toast-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 999px;
		background: #f8fafc;
		color: #475569;
	}

	.toast-item.success .toast-icon {
		background: #f0fdf4;
		color: #15803d;
	}

	.toast-item.error .toast-icon {
		background: #fef2f2;
		color: #b91c1c;
	}

	.toast-item.info .toast-icon {
		background: #eff6ff;
		color: #2563eb;
	}

	.toast-message {
		font-size: 13px;
		font-weight: 600;
		line-height: 1.45;
		color: #0f172a;
	}

	.toast-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		padding: 0;
		border: none;
		border-radius: 999px;
		background: transparent;
		color: #94a3b8;
		cursor: pointer;
	}

	.toast-close:hover {
		background: #f1f5f9;
		color: #334155;
	}
</style>
