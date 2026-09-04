CREATE TABLE IF NOT EXISTS areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS asset_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS technical_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL COLLATE NOCASE UNIQUE,
    document_type TEXT NOT NULL CHECK (document_type IN ('manual', 'plan', 'datasheet', 'instruction', 'other')),
    source_ref TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS assets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL COLLATE NOCASE UNIQUE,
    name TEXT NOT NULL,
    family TEXT NOT NULL DEFAULT '',
    area_id INTEGER,
    category_id INTEGER,
    subarea TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    serial_number TEXT NOT NULL DEFAULT '',
    operational_state TEXT NOT NULL DEFAULT 'active'
        CHECK (operational_state IN ('active', 'maintenance', 'fault', 'inactive', 'retired')),
    criticality TEXT NOT NULL DEFAULT 'medium'
        CHECK (criticality IN ('low', 'medium', 'high', 'critical')),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (area_id) REFERENCES areas (id),
    FOREIGN KEY (category_id) REFERENCES asset_categories (id)
);

CREATE TABLE IF NOT EXISTS asset_documents (
    asset_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    relation_kind TEXT NOT NULL DEFAULT 'reference'
        CHECK (relation_kind IN ('source', 'manual', 'datasheet', 'plan', 'reference')),
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (asset_id, document_id),
    FOREIGN KEY (asset_id) REFERENCES assets (id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES technical_documents (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_areas_name ON areas (name);
CREATE INDEX IF NOT EXISTS idx_asset_categories_name ON asset_categories (name);
CREATE INDEX IF NOT EXISTS idx_technical_documents_type ON technical_documents (document_type, active);
CREATE INDEX IF NOT EXISTS idx_assets_area_id ON assets (area_id);
CREATE INDEX IF NOT EXISTS idx_assets_category_id ON assets (category_id);
CREATE INDEX IF NOT EXISTS idx_assets_state_criticality ON assets (operational_state, criticality);
CREATE INDEX IF NOT EXISTS idx_asset_documents_document_id ON asset_documents (document_id);
