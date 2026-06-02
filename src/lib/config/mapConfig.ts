export const CATEGORY_COLORS: Record<string, string> = {
	gos: '#3B82F6',
	bonetsky: '#22C55E',
	chastnyi: '#F59E0B',
	rival: '#EF4444'
};

export const CATEGORY_LABELS: Record<string, string> = {
	gos: 'ГОС',
	bonetsky: 'Филиалы Бонецкого',
	chastnyi: 'Частные клиники',
	rival: 'Конкуренты'
};

export const CATEGORY_OVERLAYS: Record<string, { paths: string }> = {
	chastnyi: { paths: '<path d="M5 12h14"/><path d="M12 5v14"/>' },
	gos: { paths: '<path d="M5 12h14"/><path d="M12 5v14"/>' },
	bonetsky: {
		paths:
			'<path d="M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0A5.49 5.49 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5l-5.492 5.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5"/>'
	},
	rival: {
		paths:
			'<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>'
	}
};

export const DISTRICT_COLOR = '#6366f1';
export const STORAGE_KEY = 'map_filter_v2';
