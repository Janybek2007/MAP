export type CategoryChild = {
	key: string;
	label: string;
};

export type CategoryConfig = {
	key: string;
	label: string;
	color: string;
	sort_order: number;
	is_system: boolean;
	children: CategoryChild[];
};

export type LocationItem = {
	hid?: string;
	name: string;
	address?: string;
	lat: number;
	lng: number;
	category: string;
	child_category?: string;
	category_display?: string;
	child_category_display?: string;
	type?: string;
	type_display?: string;
	manager?: string;
	is_partnerships?: boolean;
};

export type ApiError = {
	code?: string;
	message?: string;
	fields?: Record<string, string>;
};

export type FormState = {
	hid?: string;
	name: string;
	address: string;
	category: string;
	child_category: string;
	manager: string;
	is_partnerships: boolean;
	lat: string;
	lng: string;
};

export type ChildCreateResponse = {
	key: string;
	label: string;
};

export type FilterState = {
	search: string;
	category: string;
	child_category: string;
	is_partnerships: boolean;
};
