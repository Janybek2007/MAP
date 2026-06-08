package sqlite

const initialSchemaVersion = "20260608_initial_schema"

const initialSchemaSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
	key TEXT PRIMARY KEY,
	label TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	is_system INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS category_children (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	category_key TEXT NOT NULL,
	key TEXT NOT NULL,
	label TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(category_key, key),
	FOREIGN KEY (category_key) REFERENCES categories(key) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS locations (
	hid TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	address TEXT NOT NULL DEFAULT '',
	category_key TEXT NOT NULL,
	child_category_key TEXT NOT NULL DEFAULT '',
	category_display TEXT NOT NULL DEFAULT '',
	child_category_display TEXT NOT NULL DEFAULT '',
	type_key TEXT NOT NULL DEFAULT '',
	type_display TEXT NOT NULL DEFAULT '',
	manager TEXT NOT NULL DEFAULT '',
	is_partnerships INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (category_key) REFERENCES categories(key) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS regions (
	hid TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	population INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cities (
	hid TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	population INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS districts (
	hid TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	population INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS geo_coords (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	owner_type TEXT NOT NULL,
	owner_hid TEXT NOT NULL,
	coord_kind TEXT NOT NULL,
	coord_group INTEGER NOT NULL DEFAULT 0,
	coord_order INTEGER NOT NULL DEFAULT 0,
	lat REAL NOT NULL,
	lng REAL NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT geo_coords_owner_type_check CHECK (owner_type IN ('location', 'region', 'city', 'district')),
	CONSTRAINT geo_coords_kind_check CHECK (coord_kind IN ('point', 'polygon')),
	UNIQUE(owner_type, owner_hid, coord_kind, coord_group, coord_order)
);

CREATE INDEX IF NOT EXISTS idx_category_children_category_key ON category_children(category_key);
CREATE INDEX IF NOT EXISTS idx_locations_category_key ON locations(category_key);
CREATE INDEX IF NOT EXISTS idx_locations_child_category_key ON locations(child_category_key);
CREATE INDEX IF NOT EXISTS idx_geo_coords_owner ON geo_coords(owner_type, owner_hid, coord_kind);
`
