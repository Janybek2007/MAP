<script lang="ts">
	import { Check, ChevronDown } from 'lucide-svelte';
	import { CATEGORY_COLORS, CATEGORY_LABELS } from '../../../config/mapConfig';
	import { mapStore } from '../../../store/mapStore';

	const CATEGORY_ORDER = ['bonetsky', 'gos', 'rival', 'chastnyi'];

	function getCategoryLabel(category: string) {
		const fromLocation = $mapStore.locations.find(
			(item) => item.category === category
		)?.category_display;
		return fromLocation || CATEGORY_LABELS[category] || category;
	}

	function getChildren(category: string) {
		const map = new Map<string, string>();
		for (const location of $mapStore.locations) {
			const childValue = category === 'bonetsky' ? location.type : location.child_category;
			const childDisplay =
				category === 'bonetsky'
					? location.type_display || location.type
					: location.child_category_display || location.child_category;
			if (location.category !== category || !childValue) continue;
			if (!map.has(childValue)) {
				map.set(childValue, childDisplay || childValue);
			}
		}
		return Array.from(map.entries())
			.map(([value, display]) => ({ value, display }))
			.sort((a, b) => a.display.localeCompare(b.display, 'ru'));
	}

	function childKey(category: string, child: string) {
		return `${category}:${child}`;
	}

	function getCategoryCount(category: string) {
		return (
			$mapStore.filteredCategoryCounts[category] ?? ($mapStore.markersByType[category] || []).length
		);
	}

	function getChildCount(category: string, child: string) {
		return (
			$mapStore.filteredChildCounts[childKey(category, child)] ??
			($mapStore.markersByType[category] || []).filter((item) => item.child_category === child)
				.length
		);
	}

	function getParentState(category: string) {
		const children = getChildren(category).map((item) => item.value);
		const enabled = children.filter(
			(child) => $mapStore.childActive[childKey(category, child)] !== false
		).length;
		if (enabled === 0) return 'none';
		if (enabled === children.length) return 'all';
		return 'partial';
	}
</script>

<div class="panel-pad">
	<div class="panel-section-title">Категории</div>
	{#each CATEGORY_ORDER as category}
		{#if ($mapStore.markersByType[category] || []).length > 0}
			{@const parentState = getParentState(category)}
			<div class="category-parent full-width {$mapStore.expanded[category] ? 'expanded' : ''}">
				<div
					class="category-item {parentState === 'all' ? 'active' : ''} {parentState === 'partial'
						? 'partial'
						: ''}"
					style="--c-color:{CATEGORY_COLORS[category] || '#888'}"
					role="button"
					tabindex="0"
					onclick={() => mapStore.toggleCategory(category)}
					onkeydown={(event) => {
						if (event.key === 'Enter' || event.key === ' ') {
							event.preventDefault();
							mapStore.toggleCategory(category);
						}
					}}
				>
					<span class="category-checkbox" style="--c-color:{CATEGORY_COLORS[category] || '#888'}">
						<Check class="category-checkbox-tick" size={10} strokeWidth={2.6} />
					</span>
					<span>{getCategoryLabel(category)} ({getCategoryCount(category)})</span>
					<button
						type="button"
						class="category-chevron"
						aria-label="Показать подкатегории"
						onclick={(event) => {
							event.stopPropagation();
							mapStore.toggleExpanded(category);
						}}
					>
						<ChevronDown size={12} strokeWidth={2.4} />
					</button>
				</div>
				{#if $mapStore.expanded[category]}
					<div
						class="child-category-list custom-scrollbar"
						style="max-height:260px; overflow-y:auto;"
					>
						{#each getChildren(category) as child}
							<button
								type="button"
								class="child-category-item {$mapStore.childActive[
									childKey(category, child.value)
								] !== false
									? 'active'
									: ''}"
								style="--c-color:{CATEGORY_COLORS[category] || '#888'}"
								onclick={() => mapStore.toggleChild(category, child.value)}
							>
								<span
									class="category-checkbox"
									style="--c-color:{CATEGORY_COLORS[category] || '#888'}"
								>
									<Check class="category-checkbox-tick" size={10} strokeWidth={2.6} />
								</span>
								<span>{child.display} ({getChildCount(category, child.value)})</span>
							</button>
						{/each}
					</div>
				{:else}
					<div class="child-category-list"></div>
				{/if}
			</div>
		{/if}
	{/each}
</div>
