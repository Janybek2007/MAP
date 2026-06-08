<script lang="ts">
	import { onMount } from 'svelte';

	let {
		open = false,
		title = 'Подтверждение',
		message = '',
		confirmLabel = 'Подтвердить',
		cancelLabel = 'Отмена',
		tone = 'danger',
		loading = false,
		onconfirm,
		onclose
	}: {
		open?: boolean;
		title?: string;
		message?: string;
		confirmLabel?: string;
		cancelLabel?: string;
		tone?: 'danger' | 'default';
		loading?: boolean;
		onconfirm?: () => void;
		onclose?: () => void;
	} = $props();

	function handleBackdropClick(event: MouseEvent) {
		if (event.target !== event.currentTarget || loading) return;
		onclose?.();
	}

	function handleBackdropKeydown(event: KeyboardEvent) {
		if (loading) return;
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			onclose?.();
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && !loading) onclose?.();
	}

	onMount(() => {
		document.addEventListener('keydown', handleKeydown);
		return () => {
			document.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

{#if open}
	<div
		class="confirm-modal-backdrop"
		role="button"
		tabindex="0"
		onclick={handleBackdropClick}
		onkeydown={handleBackdropKeydown}
	>
		<div class="confirm-modal-card" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
			<div class="confirm-modal-body">
				<h3 id="confirm-title">{title}</h3>
				<p>{message}</p>
			</div>
			<div class="confirm-modal-actions">
				<button class="confirm-cancel-btn" type="button" disabled={loading} onclick={() => onclose?.()}>
					{cancelLabel}
				</button>
				<button
					class={`confirm-primary-btn ${tone === 'danger' ? 'danger' : 'default'}`}
					type="button"
					disabled={loading}
					onclick={() => onconfirm?.()}
				>
					{loading ? '...' : confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.confirm-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 120;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 20px;
		background: rgba(15, 23, 42, 0.32);
		backdrop-filter: blur(3px);
	}

	.confirm-modal-card {
		width: min(100%, 420px);
		display: grid;
		gap: 18px;
		padding: 20px;
		border: 1px solid rgba(148, 163, 184, 0.2);
		border-radius: 14px;
		background: #fff;
		box-shadow: 0 20px 50px rgba(15, 23, 42, 0.18);
	}

	.confirm-modal-body {
		display: grid;
		gap: 8px;
	}

	.confirm-modal-body h3 {
		margin: 0;
		font-size: 18px;
		line-height: 1.2;
		color: #0f172a;
	}

	.confirm-modal-body p {
		margin: 0;
		font-size: 14px;
		line-height: 1.45;
		color: #475569;
	}

	.confirm-modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
	}

	.confirm-cancel-btn,
	.confirm-primary-btn {
		height: 40px;
		padding: 0 14px;
		border-radius: 10px;
		border: 1px solid transparent;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		transition: 0.15s ease;
	}

	.confirm-cancel-btn {
		background: #fff;
		border-color: rgba(148, 163, 184, 0.35);
		color: #0f172a;
	}

	.confirm-cancel-btn:hover:enabled {
		background: #f8fafc;
	}

	.confirm-primary-btn.default {
		background: #0f172a;
		color: #fff;
	}

	.confirm-primary-btn.default:hover:enabled {
		background: #1e293b;
	}

	.confirm-primary-btn.danger {
		background: #b91c1c;
		color: #fff;
	}

	.confirm-primary-btn.danger:hover:enabled {
		background: #991b1b;
	}

	button:disabled {
		opacity: 0.65;
		cursor: not-allowed;
	}

	@media (max-width: 720px) {
		.confirm-modal-actions {
			flex-direction: column;
		}
	}
</style>
