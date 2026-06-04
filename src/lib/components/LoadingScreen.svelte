<script lang="ts">
	let { loadingError, onRetry }: { loadingError: string | null; onRetry: () => void } = $props();
</script>

<div class="loading-overlay" role="status" aria-live="polite">
	<div class="loading-box">
		<button
			type="button"
			class="loading-spinner"
			aria-label={loadingError ? 'Ошибка загрузки. Нажмите, чтобы повторить' : 'Загрузка'}
			title={loadingError ? 'Ошибка загрузки. Кликните, чтобы повторить' : 'Загрузка'}
			onclick={loadingError ? onRetry : undefined}
		></button>
		{#if loadingError}
			<div class="loading-error">{loadingError}</div>
			<button type="button" class="loading-retry" onclick={onRetry}>Повторить</button>
		{/if}
	</div>
</div>

<style>
	.loading-overlay {
		position: fixed;
		inset: 0;
		z-index: 12000;
		display: grid;
		place-items: center;
		background: rgba(8, 11, 16, 0.72);
		backdrop-filter: blur(10px);
	}
	.loading-box {
		display: grid;
		justify-items: center;
		gap: 14px;
		max-width: 360px;
		padding: 20px;
		text-align: center;
	}
	.loading-spinner {
		width: 56px;
		height: 56px;
		border-radius: 999px;
		border: 4px solid rgba(255, 255, 255, 0.22);
		border-top-color: rgba(255, 255, 255, 0.92);
		background: transparent;
		box-shadow: 0 8px 26px rgba(0, 0, 0, 0.35);
		animation: spin 0.9s linear infinite;
		cursor: default;
	}
	.loading-spinner[onclick] {
		cursor: pointer;
	}
	.loading-error {
		color: #fff;
		font-size: 14px;
		line-height: 1.5;
	}
	.loading-retry {
		height: 38px;
		padding: 0 14px;
		border: 1px solid rgba(255, 255, 255, 0.25);
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		color: #fff;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
	}
	.loading-retry:hover {
		background: rgba(255, 255, 255, 0.14);
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
