CREATE TABLE IF NOT EXISTS maintenance_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    asset_id INTEGER,
    asset_category_id INTEGER,
    source_document_id INTEGER,
    source_ref TEXT NOT NULL DEFAULT '',
    frequency_code TEXT NOT NULL
        CHECK (frequency_code IN ('T', 'W', 'M', '3M', '6M', '12M', 'B')),
    maintenance_type TEXT NOT NULL
        CHECK (maintenance_type IN ('preventive', 'inspection', 'cleaning', 'safety', 'corrective')),
    procedure_summary TEXT NOT NULL,
    validation_criteria TEXT NOT NULL DEFAULT '',
    requires_checklist INTEGER NOT NULL DEFAULT 0 CHECK (requires_checklist IN (0, 1)),
    requires_supervisor INTEGER NOT NULL DEFAULT 0 CHECK (requires_supervisor IN (0, 1)),
    requires_qualified_personnel INTEGER NOT NULL DEFAULT 0 CHECK (requires_qualified_personnel IN (0, 1)),
    priority TEXT NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    estimated_minutes INTEGER NOT NULL DEFAULT 60 CHECK (estimated_minutes > 0),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (asset_id) REFERENCES assets (id),
    FOREIGN KEY (asset_category_id) REFERENCES asset_categories (id),
    FOREIGN KEY (source_document_id) REFERENCES technical_documents (id)
);

CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER,
    asset_id INTEGER,
    source_document_id INTEGER,
    title TEXT NOT NULL,
    frequency_code TEXT NOT NULL
        CHECK (frequency_code IN ('T', 'W', 'M', '3M', '6M', '12M', 'B')),
    maintenance_type TEXT NOT NULL
        CHECK (maintenance_type IN ('preventive', 'inspection', 'cleaning', 'safety', 'corrective')),
    status TEXT NOT NULL DEFAULT 'scheduled'
        CHECK (status IN ('draft', 'scheduled', 'published', 'cancelled', 'reprogrammed', 'completed')),
    scheduled_for TEXT NOT NULL,
    window_start TEXT NOT NULL DEFAULT '',
    window_end TEXT NOT NULL DEFAULT '',
    publication_notes TEXT NOT NULL DEFAULT '',
    created_by INTEGER,
    published_by INTEGER,
    published_at TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (template_id) REFERENCES maintenance_templates (id),
    FOREIGN KEY (asset_id) REFERENCES assets (id),
    FOREIGN KEY (source_document_id) REFERENCES technical_documents (id),
    FOREIGN KEY (created_by) REFERENCES users (id),
    FOREIGN KEY (published_by) REFERENCES users (id)
);

CREATE INDEX IF NOT EXISTS idx_maintenance_templates_asset ON maintenance_templates (asset_id, active);
CREATE INDEX IF NOT EXISTS idx_maintenance_templates_category ON maintenance_templates (asset_category_id, active);
CREATE INDEX IF NOT EXISTS idx_maintenance_templates_frequency ON maintenance_templates (frequency_code, active);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_template ON scheduled_tasks (template_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_asset_date ON scheduled_tasks (asset_id, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_status_date ON scheduled_tasks (status, scheduled_for);
