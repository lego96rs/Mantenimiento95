CREATE TABLE IF NOT EXISTS work_orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scheduled_task_id INTEGER NOT NULL UNIQUE,
    work_order_code TEXT NOT NULL UNIQUE,
    asset_id INTEGER,
    template_id INTEGER,
    title TEXT NOT NULL,
    execution_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (execution_status IN ('pending', 'assigned', 'in_progress', 'paused', 'blocked', 'done', 'cancelled')),
    assigned_to INTEGER,
    assigned_at TEXT NOT NULL DEFAULT '',
    published_by INTEGER,
    start_time TEXT NOT NULL DEFAULT '',
    end_time TEXT NOT NULL DEFAULT '',
    total_minutes INTEGER NOT NULL DEFAULT 0 CHECK (total_minutes >= 0),
    execution_notes TEXT NOT NULL DEFAULT '',
    close_summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (scheduled_task_id) REFERENCES scheduled_tasks (id),
    FOREIGN KEY (asset_id) REFERENCES assets (id),
    FOREIGN KEY (template_id) REFERENCES maintenance_templates (id),
    FOREIGN KEY (assigned_to) REFERENCES users (id),
    FOREIGN KEY (published_by) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS work_order_checklists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_order_id INTEGER NOT NULL,
    item_text TEXT NOT NULL,
    is_done INTEGER NOT NULL DEFAULT 0 CHECK (is_done IN (0, 1)),
    notes TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (work_order_id) REFERENCES work_orders (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_order_id INTEGER NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'resolved', 'closed')),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    escalation_notes TEXT NOT NULL DEFAULT '',
    reported_by INTEGER,
    resolved_by INTEGER,
    resolved_at TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (work_order_id) REFERENCES work_orders (id) ON DELETE CASCADE,
    FOREIGN KEY (reported_by) REFERENCES users (id),
    FOREIGN KEY (resolved_by) REFERENCES users (id)
);

CREATE INDEX IF NOT EXISTS idx_work_orders_status_assignee ON work_orders (execution_status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_work_orders_asset_status ON work_orders (asset_id, execution_status);
CREATE INDEX IF NOT EXISTS idx_work_order_checklists_work_order ON work_order_checklists (work_order_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_incidents_status_severity ON incidents (status, severity);
CREATE INDEX IF NOT EXISTS idx_incidents_work_order ON incidents (work_order_id);
