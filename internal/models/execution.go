package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mantenimiento/internal/db"
)

const (
	WorkOrderStatusPending    = "pending"
	WorkOrderStatusAssigned   = "assigned"
	WorkOrderStatusInProgress = "in_progress"
	WorkOrderStatusPaused     = "paused"
	WorkOrderStatusBlocked    = "blocked"
	WorkOrderStatusDone       = "done"
	WorkOrderStatusCancelled  = "cancelled"

	IncidentSeverityLow      = "low"
	IncidentSeverityMedium   = "medium"
	IncidentSeverityHigh     = "high"
	IncidentSeverityCritical = "critical"

	IncidentStatusOpen          = "open"
	IncidentStatusInvestigating = "investigating"
	IncidentStatusResolved      = "resolved"
	IncidentStatusClosed        = "closed"
)

type WorkOrder struct {
	ID                 int64
	ScheduledTaskID    int64
	WorkOrderCode      string
	AssetID            int64
	AssetName          string
	TemplateID         int64
	TemplateName       string
	Title              string
	MaintenanceType    string
	ExecutionStatus    string
	AssignedTo         int64
	AssignedToName     string
	AssignedAt         string
	PublishedBy        int64
	PublishedByName    string
	ScheduledFor       string
	WindowStart        string
	WindowEnd          string
	PublicationNotes   string
	StartTime          string
	EndTime            string
	TotalMinutes       int
	ExecutionNotes     string
	CloseSummary       string
	ChecklistPending   int
	OpenIncidentCount  int
	RequiresChecklist  bool
}

type WorkOrderFilters struct {
	Status          string
	AssetID         int64
	AssignedTo      int64
	FromDate        string
	ToDate          string
	MaintenanceType string
}

type WorkOrderDetail struct {
	WorkOrder
	Checklist []WorkOrderChecklistItem
	Incidents []Incident
}

type WorkOrderChecklistItem struct {
	ID         int64
	WorkOrderID int64
	ItemText   string
	IsDone     bool
	Notes      string
	SortOrder  int
}

type WorkOrderChecklistItemInput struct {
	ItemText  string
	IsDone    bool
	Notes     string
	SortOrder int
}

type WorkOrderProgressInput struct {
	Status         string
	ExecutionNotes string
	CloseSummary   string
	TotalMinutes   int
}

type Incident struct {
	ID              int64
	WorkOrderID     int64
	WorkOrderCode   string
	AssetID         int64
	AssetName       string
	Severity        string
	Status          string
	Title           string
	Description     string
	EscalationNotes string
	ReportedBy      int64
	ReportedByName  string
	ResolvedBy      int64
	ResolvedByName  string
	ResolvedAt      string
	CreatedAt       string
	UpdatedAt       string
}

type IncidentFilters struct {
	WorkOrderID int64
	Status      string
	Severity    string
	AssetID     int64
}

type IncidentInput struct {
	WorkOrderID     int64
	Severity        string
	Status          string
	Title           string
	Description     string
	EscalationNotes string
	ReportedBy      int64
}

type IncidentUpdateInput struct {
	Status          string
	EscalationNotes string
	ResolvedBy      int64
}

func ListWorkOrders(ctx context.Context, d *db.DB, filters WorkOrderFilters) ([]WorkOrder, error) {
	var (
		args []any
		sqlb strings.Builder
	)

	sqlb.WriteString(`
SELECT
    wo.id,
    wo.scheduled_task_id,
    wo.work_order_code,
    COALESCE(wo.asset_id, 0),
    COALESCE(a.name, ''),
    COALESCE(wo.template_id, 0),
    COALESCE(mt.name, ''),
    wo.title,
    COALESCE(st.maintenance_type, ''),
    wo.execution_status,
    COALESCE(wo.assigned_to, 0),
    COALESCE(ua.display_name, ''),
    wo.assigned_at,
    COALESCE(wo.published_by, 0),
    COALESCE(up.display_name, ''),
    COALESCE(st.scheduled_for, ''),
    COALESCE(st.window_start, ''),
    COALESCE(st.window_end, ''),
    COALESCE(st.publication_notes, ''),
    wo.start_time,
    wo.end_time,
    wo.total_minutes,
    wo.execution_notes,
    wo.close_summary,
    COALESCE(mt.requires_checklist, 0),
    (
        SELECT COUNT(*)
        FROM work_order_checklists woc
        WHERE woc.work_order_id = wo.id
          AND woc.is_done = 0
    ) AS checklist_pending,
    (
        SELECT COUNT(*)
        FROM incidents i
        WHERE i.work_order_id = wo.id
          AND i.status NOT IN ('resolved', 'closed')
    ) AS open_incidents
FROM work_orders wo
LEFT JOIN scheduled_tasks st ON st.id = wo.scheduled_task_id
LEFT JOIN assets a ON a.id = wo.asset_id
LEFT JOIN maintenance_templates mt ON mt.id = wo.template_id
LEFT JOIN users ua ON ua.id = wo.assigned_to
LEFT JOIN users up ON up.id = wo.published_by
WHERE 1 = 1`)

	if status := strings.TrimSpace(filters.Status); status != "" {
		sqlb.WriteString(` AND wo.execution_status = ?`)
		args = append(args, status)
	}
	if filters.AssetID > 0 {
		sqlb.WriteString(` AND wo.asset_id = ?`)
		args = append(args, filters.AssetID)
	}
	if filters.AssignedTo > 0 {
		sqlb.WriteString(` AND wo.assigned_to = ?`)
		args = append(args, filters.AssignedTo)
	}
	if from := strings.TrimSpace(filters.FromDate); from != "" {
		sqlb.WriteString(` AND st.scheduled_for >= ?`)
		args = append(args, from)
	}
	if to := strings.TrimSpace(filters.ToDate); to != "" {
		sqlb.WriteString(` AND st.scheduled_for <= ?`)
		args = append(args, to)
	}
	if mt := strings.TrimSpace(filters.MaintenanceType); mt != "" {
		sqlb.WriteString(` AND st.maintenance_type = ?`)
		args = append(args, mt)
	}

	sqlb.WriteString(` ORDER BY st.scheduled_for, wo.work_order_code`)

	rows, err := d.Read.QueryContext(ctx, sqlb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("models: list work orders: %w", err)
	}
	defer rows.Close()

	var items []WorkOrder
	for rows.Next() {
		var (
			item             WorkOrder
			requiresChecklist int
		)
		if err := rows.Scan(
			&item.ID,
			&item.ScheduledTaskID,
			&item.WorkOrderCode,
			&item.AssetID,
			&item.AssetName,
			&item.TemplateID,
			&item.TemplateName,
			&item.Title,
			&item.MaintenanceType,
			&item.ExecutionStatus,
			&item.AssignedTo,
			&item.AssignedToName,
			&item.AssignedAt,
			&item.PublishedBy,
			&item.PublishedByName,
			&item.ScheduledFor,
			&item.WindowStart,
			&item.WindowEnd,
			&item.PublicationNotes,
			&item.StartTime,
			&item.EndTime,
			&item.TotalMinutes,
			&item.ExecutionNotes,
			&item.CloseSummary,
			&requiresChecklist,
			&item.ChecklistPending,
			&item.OpenIncidentCount,
		); err != nil {
			return nil, fmt.Errorf("models: scan work order: %w", err)
		}
		item.RequiresChecklist = requiresChecklist == 1
		items = append(items, item)
	}

	return items, rows.Err()
}

func WorkOrderByID(ctx context.Context, d *db.DB, workOrderID int64) (*WorkOrderDetail, error) {
	items, err := ListWorkOrders(ctx, d, WorkOrderFilters{})
	if err != nil {
		return nil, err
	}

	var selected *WorkOrder
	for i := range items {
		if items[i].ID == workOrderID {
			selected = &items[i]
			break
		}
	}
	if selected == nil {
		return nil, nil
	}

	checklist, err := ListWorkOrderChecklist(ctx, d, workOrderID)
	if err != nil {
		return nil, err
	}
	incidents, err := ListIncidents(ctx, d, IncidentFilters{WorkOrderID: workOrderID})
	if err != nil {
		return nil, err
	}

	return &WorkOrderDetail{
		WorkOrder: *selected,
		Checklist: checklist,
		Incidents: incidents,
	}, nil
}

func CreateWorkOrderFromScheduledTask(ctx context.Context, d *db.DB, scheduledTaskID, publishedBy int64) (int64, error) {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("models: begin create work order tx: %w", err)
	}
	defer tx.Rollback()

	task, err := loadPublishedScheduledTask(ctx, tx, scheduledTaskID)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, fmt.Errorf("models: scheduled task %d not found", scheduledTaskID)
	}
	if task.Status != PlannedStatusPublished {
		return 0, fmt.Errorf("models: scheduled task %d must be published before creating a work order", scheduledTaskID)
	}

	var existing int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM work_orders WHERE scheduled_task_id = ?`,
		scheduledTaskID,
	).Scan(&existing)
	if err == nil {
		return 0, fmt.Errorf("models: work order already exists for scheduled task %d", scheduledTaskID)
	}
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("models: existing work order lookup: %w", err)
	}

	if publishedBy > 0 {
		if err := ensureActiveUser(ctx, tx, publishedBy); err != nil {
			return 0, err
		}
	} else {
		publishedBy = task.PublishedBy
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO work_orders (
		    scheduled_task_id,
		    work_order_code,
		    asset_id,
		    template_id,
		    title,
		    published_by
		) VALUES (?, ?, ?, ?, ?, ?)`,
		scheduledTaskID,
		generateWorkOrderCode(scheduledTaskID),
		nullableID(task.AssetID),
		nullableID(task.TemplateID),
		task.Title,
		nullableID(publishedBy),
	)
	if err != nil {
		return 0, fmt.Errorf("models: insert work order: %w", err)
	}

	workOrderID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("models: work order id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("models: commit create work order: %w", err)
	}
	return workOrderID, nil
}

func SetWorkOrderAssignment(ctx context.Context, d *db.DB, workOrderID, assignedTo int64) error {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("models: begin assignment tx: %w", err)
	}
	defer tx.Rollback()

	state, err := loadWorkOrderState(ctx, tx, workOrderID)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("models: work order %d not found", workOrderID)
	}
	if state.ExecutionStatus == WorkOrderStatusDone || state.ExecutionStatus == WorkOrderStatusCancelled {
		return fmt.Errorf("models: work order %d can no longer be reassigned", workOrderID)
	}

	status := state.ExecutionStatus
	assignedAt := state.AssignedAt

	if assignedTo > 0 {
		if err := ensureAssignableUser(ctx, tx, assignedTo); err != nil {
			return err
		}
		assignedAt = nowUTC()
		if status == WorkOrderStatusPending || status == "" {
			status = WorkOrderStatusAssigned
		}
	} else {
		if status == WorkOrderStatusInProgress {
			return fmt.Errorf("models: work order %d cannot be unassigned while in progress", workOrderID)
		}
		assignedAt = ""
		if status == WorkOrderStatusAssigned || status == WorkOrderStatusPending {
			status = WorkOrderStatusPending
		}
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE work_orders
		 SET assigned_to = ?,
		     assigned_at = ?,
		     execution_status = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		nullableID(assignedTo),
		assignedAt,
		status,
		workOrderID,
	)
	if err != nil {
		return fmt.Errorf("models: update work order assignment: %w", err)
	}

	return tx.Commit()
}

func UpdateWorkOrderProgress(ctx context.Context, d *db.DB, workOrderID int64, input WorkOrderProgressInput) error {
	if !validWorkOrderStatus(input.Status) {
		return fmt.Errorf("models: invalid work order status")
	}
	if input.TotalMinutes < 0 {
		return fmt.Errorf("models: total minutes cannot be negative")
	}

	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("models: begin work order progress tx: %w", err)
	}
	defer tx.Rollback()

	state, err := loadWorkOrderState(ctx, tx, workOrderID)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("models: work order %d not found", workOrderID)
	}
	if state.ExecutionStatus == WorkOrderStatusDone || state.ExecutionStatus == WorkOrderStatusCancelled {
		return fmt.Errorf("models: work order %d is already closed", workOrderID)
	}

	startTime := state.StartTime
	endTime := state.EndTime
	closeSummary := strings.TrimSpace(input.CloseSummary)

	switch input.Status {
	case WorkOrderStatusAssigned:
		if !state.AssignedTo.Valid {
			return fmt.Errorf("models: work order %d must have an assignee before moving to assigned", workOrderID)
		}
	case WorkOrderStatusInProgress:
		if !state.AssignedTo.Valid {
			return fmt.Errorf("models: work order %d must be assigned before starting", workOrderID)
		}
		if startTime == "" {
			startTime = nowUTC()
		}
		endTime = ""
	case WorkOrderStatusPaused, WorkOrderStatusBlocked:
		if startTime == "" {
			startTime = nowUTC()
		}
		endTime = ""
	case WorkOrderStatusDone:
		if !state.AssignedTo.Valid {
			return fmt.Errorf("models: work order %d must have an assignee before closing", workOrderID)
		}
		if startTime == "" {
			startTime = nowUTC()
		}
		if closeSummary == "" {
			return fmt.Errorf("models: close summary is required to complete a work order")
		}
		if err := validateChecklistForClosure(ctx, tx, state); err != nil {
			return err
		}
		if err := validateIncidentClosure(ctx, tx, workOrderID); err != nil {
			return err
		}
		endTime = nowUTC()
	case WorkOrderStatusCancelled:
		endTime = nowUTC()
	default:
		return fmt.Errorf("models: unsupported work order status transition")
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE work_orders
		 SET execution_status = ?,
		     start_time = ?,
		     end_time = ?,
		     total_minutes = ?,
		     execution_notes = ?,
		     close_summary = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		input.Status,
		startTime,
		endTime,
		input.TotalMinutes,
		strings.TrimSpace(input.ExecutionNotes),
		closeSummary,
		workOrderID,
	); err != nil {
		return fmt.Errorf("models: update work order progress: %w", err)
	}

	switch input.Status {
	case WorkOrderStatusDone:
		if err := updateScheduledTaskStatusTx(ctx, tx, state.ScheduledTaskID, PlannedStatusCompleted); err != nil {
			return err
		}
	case WorkOrderStatusCancelled:
		if err := updateScheduledTaskStatusTx(ctx, tx, state.ScheduledTaskID, PlannedStatusCancelled); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ListWorkOrderChecklist(ctx context.Context, d *db.DB, workOrderID int64) ([]WorkOrderChecklistItem, error) {
	rows, err := d.Read.QueryContext(ctx,
		`SELECT id, work_order_id, item_text, is_done, notes, sort_order
		 FROM work_order_checklists
		 WHERE work_order_id = ?
		 ORDER BY sort_order, id`,
		workOrderID,
	)
	if err != nil {
		return nil, fmt.Errorf("models: list work order checklist: %w", err)
	}
	defer rows.Close()

	var items []WorkOrderChecklistItem
	for rows.Next() {
		var item WorkOrderChecklistItem
		if err := rows.Scan(
			&item.ID,
			&item.WorkOrderID,
			&item.ItemText,
			&item.IsDone,
			&item.Notes,
			&item.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("models: scan work order checklist item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func SetWorkOrderChecklist(ctx context.Context, d *db.DB, workOrderID int64, items []WorkOrderChecklistItemInput) error {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("models: begin checklist tx: %w", err)
	}
	defer tx.Rollback()

	ok, err := existsByID(ctx, tx, "work_orders", workOrderID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("models: work order %d not found", workOrderID)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM work_order_checklists WHERE work_order_id = ?`, workOrderID); err != nil {
		return fmt.Errorf("models: delete checklist items: %w", err)
	}

	for i, item := range items {
		text := strings.TrimSpace(item.ItemText)
		if text == "" {
			return fmt.Errorf("models: checklist item text is required")
		}
		sortOrder := item.SortOrder
		if sortOrder <= 0 {
			sortOrder = i + 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO work_order_checklists (work_order_id, item_text, is_done, notes, sort_order)
			 VALUES (?, ?, ?, ?, ?)`,
			workOrderID,
			text,
			boolToInt(item.IsDone),
			strings.TrimSpace(item.Notes),
			sortOrder,
		); err != nil {
			return fmt.Errorf("models: insert checklist item: %w", err)
		}
	}

	return tx.Commit()
}

func UpdateChecklistItem(ctx context.Context, d *db.DB, itemID int64, isDone bool, notes string) error {
	res, err := d.Write.ExecContext(ctx,
		`UPDATE work_order_checklists
		 SET is_done = ?,
		     notes = ?
		 WHERE id = ?`,
		boolToInt(isDone),
		strings.TrimSpace(notes),
		itemID,
	)
	if err != nil {
		return fmt.Errorf("models: update checklist item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("models: checklist item %d not found", itemID)
	}
	return nil
}

func ListIncidents(ctx context.Context, d *db.DB, filters IncidentFilters) ([]Incident, error) {
	var (
		args []any
		sqlb strings.Builder
	)

	sqlb.WriteString(`
SELECT
    i.id,
    i.work_order_id,
    COALESCE(wo.work_order_code, ''),
    COALESCE(wo.asset_id, 0),
    COALESCE(a.name, ''),
    i.severity,
    i.status,
    i.title,
    i.description,
    i.escalation_notes,
    COALESCE(i.reported_by, 0),
    COALESCE(ur.display_name, ''),
    COALESCE(i.resolved_by, 0),
    COALESCE(uz.display_name, ''),
    i.resolved_at,
    i.created_at,
    i.updated_at
FROM incidents i
LEFT JOIN work_orders wo ON wo.id = i.work_order_id
LEFT JOIN assets a ON a.id = wo.asset_id
LEFT JOIN users ur ON ur.id = i.reported_by
LEFT JOIN users uz ON uz.id = i.resolved_by
WHERE 1 = 1`)

	if filters.WorkOrderID > 0 {
		sqlb.WriteString(` AND i.work_order_id = ?`)
		args = append(args, filters.WorkOrderID)
	}
	if status := strings.TrimSpace(filters.Status); status != "" {
		sqlb.WriteString(` AND i.status = ?`)
		args = append(args, status)
	}
	if severity := strings.TrimSpace(filters.Severity); severity != "" {
		sqlb.WriteString(` AND i.severity = ?`)
		args = append(args, severity)
	}
	if filters.AssetID > 0 {
		sqlb.WriteString(` AND wo.asset_id = ?`)
		args = append(args, filters.AssetID)
	}

	sqlb.WriteString(` ORDER BY i.created_at DESC, i.id DESC`)

	rows, err := d.Read.QueryContext(ctx, sqlb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("models: list incidents: %w", err)
	}
	defer rows.Close()

	var items []Incident
	for rows.Next() {
		var item Incident
		if err := rows.Scan(
			&item.ID,
			&item.WorkOrderID,
			&item.WorkOrderCode,
			&item.AssetID,
			&item.AssetName,
			&item.Severity,
			&item.Status,
			&item.Title,
			&item.Description,
			&item.EscalationNotes,
			&item.ReportedBy,
			&item.ReportedByName,
			&item.ResolvedBy,
			&item.ResolvedByName,
			&item.ResolvedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("models: scan incident: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func CreateIncident(ctx context.Context, d *db.DB, input IncidentInput) (int64, error) {
	if err := validateIncidentInput(ctx, d, input); err != nil {
		return 0, err
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = IncidentStatusOpen
	}

	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO incidents (
		    work_order_id,
		    severity,
		    status,
		    title,
		    description,
		    escalation_notes,
		    reported_by
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.WorkOrderID,
		input.Severity,
		input.Status,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.EscalationNotes),
		nullableID(input.ReportedBy),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create incident: %w", err)
	}
	return res.LastInsertId()
}

func UpdateIncident(ctx context.Context, d *db.DB, incidentID int64, input IncidentUpdateInput) error {
	if !validIncidentStatus(input.Status) {
		return fmt.Errorf("models: invalid incident status")
	}

	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("models: begin incident update tx: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM incidents WHERE id = ?`,
		incidentID,
	).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("models: incident %d not found", incidentID)
	}
	if err != nil {
		return fmt.Errorf("models: load incident: %w", err)
	}

	resolvedBy := nullableID(input.ResolvedBy)
	resolvedAt := ""
	if input.Status == IncidentStatusResolved || input.Status == IncidentStatusClosed {
		if input.ResolvedBy <= 0 {
			return fmt.Errorf("models: resolved incidents require a resolver")
		}
		if err := ensureActiveUser(ctx, tx, input.ResolvedBy); err != nil {
			return err
		}
		resolvedAt = nowUTC()
	}
	if currentStatus == IncidentStatusClosed {
		return fmt.Errorf("models: incident %d is already closed", incidentID)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE incidents
		 SET status = ?,
		     escalation_notes = ?,
		     resolved_by = ?,
		     resolved_at = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		input.Status,
		strings.TrimSpace(input.EscalationNotes),
		resolvedBy,
		resolvedAt,
		incidentID,
	); err != nil {
		return fmt.Errorf("models: update incident: %w", err)
	}

	return tx.Commit()
}

type scheduledTaskSummary struct {
	ID          int64
	TemplateID  int64
	AssetID     int64
	Title       string
	Status      string
	PublishedBy int64
}

type workOrderState struct {
	WorkOrderID         int64
	ScheduledTaskID    int64
	ExecutionStatus    string
	AssignedTo         sql.NullInt64
	AssignedAt         string
	StartTime          string
	EndTime            string
	RequiresChecklist  bool
}

func loadPublishedScheduledTask(ctx context.Context, tx *sql.Tx, scheduledTaskID int64) (*scheduledTaskSummary, error) {
	var (
		item       scheduledTaskSummary
		templateID sql.NullInt64
		assetID    sql.NullInt64
		publishedBy sql.NullInt64
	)

	err := tx.QueryRowContext(ctx,
		`SELECT id, template_id, asset_id, title, status, published_by
		 FROM scheduled_tasks
		 WHERE id = ?`,
		scheduledTaskID,
	).Scan(
		&item.ID,
		&templateID,
		&assetID,
		&item.Title,
		&item.Status,
		&publishedBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: load scheduled task summary: %w", err)
	}
	if templateID.Valid {
		item.TemplateID = templateID.Int64
	}
	if assetID.Valid {
		item.AssetID = assetID.Int64
	}
	if publishedBy.Valid {
		item.PublishedBy = publishedBy.Int64
	}
	return &item, nil
}

func loadWorkOrderState(ctx context.Context, tx *sql.Tx, workOrderID int64) (*workOrderState, error) {
	var (
		item              workOrderState
		requiresChecklist int
	)
	err := tx.QueryRowContext(ctx,
		`SELECT
		    wo.id,
		    wo.scheduled_task_id,
		    wo.execution_status,
		    wo.assigned_to,
		    wo.assigned_at,
		    wo.start_time,
		    wo.end_time,
		    COALESCE(mt.requires_checklist, 0)
		 FROM work_orders wo
		 LEFT JOIN maintenance_templates mt ON mt.id = wo.template_id
		 WHERE wo.id = ?`,
		workOrderID,
	).Scan(
		&item.WorkOrderID,
		&item.ScheduledTaskID,
		&item.ExecutionStatus,
		&item.AssignedTo,
		&item.AssignedAt,
		&item.StartTime,
		&item.EndTime,
		&requiresChecklist,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: load work order state: %w", err)
	}
	item.RequiresChecklist = requiresChecklist == 1
	return &item, nil
}

func validateChecklistForClosure(ctx context.Context, tx *sql.Tx, state *workOrderState) error {
	if !state.RequiresChecklist {
		return nil
	}

	var total int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM work_order_checklists WHERE work_order_id = ?`,
		state.WorkOrderID,
	).Scan(&total); err != nil {
		return fmt.Errorf("models: count checklist items: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("models: work order requires a checklist before closing")
	}

	var pending int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM work_order_checklists
		 WHERE work_order_id = ?
		   AND is_done = 0`,
		state.WorkOrderID,
	).Scan(&pending); err != nil {
		return fmt.Errorf("models: count pending checklist items: %w", err)
	}
	if pending > 0 {
		return fmt.Errorf("models: all checklist items must be completed before closing the work order")
	}
	return nil
}

func validateIncidentClosure(ctx context.Context, tx *sql.Tx, workOrderID int64) error {
	var open int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM incidents
		 WHERE work_order_id = ?
		   AND status NOT IN ('resolved', 'closed')`,
		workOrderID,
	).Scan(&open); err != nil {
		return fmt.Errorf("models: count open incidents: %w", err)
	}
	if open > 0 {
		return fmt.Errorf("models: work order has open incidents")
	}
	return nil
}

func validateIncidentInput(ctx context.Context, d *db.DB, input IncidentInput) error {
	switch {
	case input.WorkOrderID <= 0:
		return fmt.Errorf("models: work order is required")
	case strings.TrimSpace(input.Title) == "":
		return fmt.Errorf("models: incident title is required")
	case strings.TrimSpace(input.Description) == "":
		return fmt.Errorf("models: incident description is required")
	case !validIncidentSeverity(input.Severity):
		return fmt.Errorf("models: invalid incident severity")
	}
	if strings.TrimSpace(input.Status) != "" && !validIncidentStatus(input.Status) {
		return fmt.Errorf("models: invalid incident status")
	}

	workOrder, err := WorkOrderByID(ctx, d, input.WorkOrderID)
	if err != nil {
		return err
	}
	if workOrder == nil {
		return fmt.Errorf("models: work order %d not found", input.WorkOrderID)
	}
	if input.ReportedBy > 0 {
		user, err := UserByID(ctx, d, input.ReportedBy)
		if err != nil {
			return err
		}
		if user == nil || !user.Active {
			return fmt.Errorf("models: reporting user %d not found", input.ReportedBy)
		}
	}
	return nil
}

func ensureActiveUser(ctx context.Context, tx *sql.Tx, userID int64) error {
	var active bool
	err := tx.QueryRowContext(ctx,
		`SELECT active FROM users WHERE id = ?`,
		userID,
	).Scan(&active)
	if err == sql.ErrNoRows {
		return fmt.Errorf("models: user %d not found", userID)
	}
	if err != nil {
		return fmt.Errorf("models: load user %d: %w", userID, err)
	}
	if !active {
		return fmt.Errorf("models: user %d is inactive", userID)
	}
	return nil
}

func ensureAssignableUser(ctx context.Context, tx *sql.Tx, userID int64) error {
	var (
		role   string
		active bool
	)
	err := tx.QueryRowContext(ctx,
		`SELECT role, active FROM users WHERE id = ?`,
		userID,
	).Scan(&role, &active)
	if err == sql.ErrNoRows {
		return fmt.Errorf("models: assignee %d not found", userID)
	}
	if err != nil {
		return fmt.Errorf("models: load assignee %d: %w", userID, err)
	}
	if !active {
		return fmt.Errorf("models: assignee %d is inactive", userID)
	}
	if role == RoleViewer {
		return fmt.Errorf("models: viewers cannot be assigned work orders")
	}
	return nil
}

func updateScheduledTaskStatusTx(ctx context.Context, tx *sql.Tx, scheduledTaskID int64, status string) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE scheduled_tasks
		 SET status = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		status,
		scheduledTaskID,
	); err != nil {
		return fmt.Errorf("models: update scheduled task status: %w", err)
	}
	return nil
}

func validWorkOrderStatus(status string) bool {
	switch status {
	case WorkOrderStatusPending, WorkOrderStatusAssigned, WorkOrderStatusInProgress, WorkOrderStatusPaused, WorkOrderStatusBlocked, WorkOrderStatusDone, WorkOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func validIncidentSeverity(severity string) bool {
	switch severity {
	case IncidentSeverityLow, IncidentSeverityMedium, IncidentSeverityHigh, IncidentSeverityCritical:
		return true
	default:
		return false
	}
}

func validIncidentStatus(status string) bool {
	switch status {
	case IncidentStatusOpen, IncidentStatusInvestigating, IncidentStatusResolved, IncidentStatusClosed:
		return true
	default:
		return false
	}
}

func generateWorkOrderCode(scheduledTaskID int64) string {
	return fmt.Sprintf("OT-%06d", scheduledTaskID)
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
