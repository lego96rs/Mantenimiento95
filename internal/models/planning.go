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
	FrequencyDaily       = "T"
	FrequencyWeekly      = "W"
	FrequencyMonthly     = "M"
	FrequencyQuarterly   = "3M"
	FrequencySemiAnnual  = "6M"
	FrequencyYearly      = "12M"
	FrequencyConditional = "B"

	MaintenanceTypePreventive = "preventive"
	MaintenanceTypeInspection = "inspection"
	MaintenanceTypeCleaning   = "cleaning"
	MaintenanceTypeSafety     = "safety"
	MaintenanceTypeCorrective = "corrective"

	PlannedStatusDraft        = "draft"
	PlannedStatusScheduled    = "scheduled"
	PlannedStatusPublished    = "published"
	PlannedStatusCancelled    = "cancelled"
	PlannedStatusReprogrammed = "reprogrammed"
	PlannedStatusCompleted    = "completed"
)

type MaintenanceTemplate struct {
	ID                         int64
	Name                       string
	AssetID                    int64
	AssetName                  string
	AssetCategoryID            int64
	AssetCategoryName          string
	SourceDocumentID           int64
	SourceDocumentTitle        string
	SourceRef                  string
	FrequencyCode              string
	MaintenanceType            string
	ProcedureSummary           string
	ValidationCriteria         string
	RequiresChecklist          bool
	RequiresSupervisor         bool
	RequiresQualifiedPersonnel bool
	Priority                   string
	EstimatedMinutes           int
	Active                     bool
}

type MaintenanceTemplateInput struct {
	Name                       string
	AssetID                    int64
	AssetCategoryID            int64
	SourceDocumentID           int64
	SourceRef                  string
	FrequencyCode              string
	MaintenanceType            string
	ProcedureSummary           string
	ValidationCriteria         string
	RequiresChecklist          bool
	RequiresSupervisor         bool
	RequiresQualifiedPersonnel bool
	Priority                   string
	EstimatedMinutes           int
	Active                     bool
}

type TemplateFilters struct {
	Query           string
	AssetID         int64
	AssetCategoryID int64
	FrequencyCode   string
	MaintenanceType string
	ActiveFilter    string
}

type ScheduledTask struct {
	ID                  int64
	TemplateID          int64
	TemplateName        string
	AssetID             int64
	AssetName           string
	SourceDocumentID    int64
	SourceDocumentTitle string
	Title               string
	FrequencyCode       string
	MaintenanceType     string
	Status              string
	ScheduledFor        string
	WindowStart         string
	WindowEnd           string
	PublicationNotes    string
	CreatedByName       string
	PublishedByName     string
	PublishedAt         string
}

type ScheduledTaskInput struct {
	TemplateID       int64
	AssetID          int64
	SourceDocumentID int64
	Title            string
	FrequencyCode    string
	MaintenanceType  string
	Status           string
	ScheduledFor     string
	WindowStart      string
	WindowEnd        string
	PublicationNotes string
	CreatedBy        int64
	PublishedBy      int64
	PublishedAt      string
}

type ScheduleFilters struct {
	Status          string
	AssetID         int64
	TemplateID      int64
	FromDate        string
	ToDate          string
	MaintenanceType string
}

func ListMaintenanceTemplates(ctx context.Context, d *db.DB, filters TemplateFilters) ([]MaintenanceTemplate, error) {
	var (
		args []any
		sqlb strings.Builder
	)

	sqlb.WriteString(`
SELECT
    mt.id,
    mt.name,
    COALESCE(mt.asset_id, 0),
    COALESCE(a.name, ''),
    COALESCE(mt.asset_category_id, 0),
    COALESCE(ac.name, ''),
    COALESCE(mt.source_document_id, 0),
    COALESCE(td.title, ''),
    mt.source_ref,
    mt.frequency_code,
    mt.maintenance_type,
    mt.procedure_summary,
    mt.validation_criteria,
    mt.requires_checklist,
    mt.requires_supervisor,
    mt.requires_qualified_personnel,
    mt.priority,
    mt.estimated_minutes,
    mt.active
FROM maintenance_templates mt
LEFT JOIN assets a ON a.id = mt.asset_id
LEFT JOIN asset_categories ac ON ac.id = mt.asset_category_id
LEFT JOIN technical_documents td ON td.id = mt.source_document_id
WHERE 1 = 1`)

	if q := strings.TrimSpace(filters.Query); q != "" {
		like := "%" + q + "%"
		sqlb.WriteString(`
  AND (
        mt.name LIKE ?
     OR mt.procedure_summary LIKE ?
     OR a.name LIKE ?
     OR ac.name LIKE ?
     OR td.title LIKE ?
  )`)
		for i := 0; i < 5; i++ {
			args = append(args, like)
		}
	}
	if filters.AssetID > 0 {
		sqlb.WriteString(` AND mt.asset_id = ?`)
		args = append(args, filters.AssetID)
	}
	if filters.AssetCategoryID > 0 {
		sqlb.WriteString(` AND mt.asset_category_id = ?`)
		args = append(args, filters.AssetCategoryID)
	}
	if f := strings.TrimSpace(filters.FrequencyCode); f != "" {
		sqlb.WriteString(` AND mt.frequency_code = ?`)
		args = append(args, f)
	}
	if mt := strings.TrimSpace(filters.MaintenanceType); mt != "" {
		sqlb.WriteString(` AND mt.maintenance_type = ?`)
		args = append(args, mt)
	}
	switch filters.ActiveFilter {
	case "inactive":
		sqlb.WriteString(` AND mt.active = 0`)
	case "all":
	default:
		sqlb.WriteString(` AND mt.active = 1`)
	}

	sqlb.WriteString(` ORDER BY mt.name, mt.id`)

	rows, err := d.Read.QueryContext(ctx, sqlb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("models: list maintenance templates: %w", err)
	}
	defer rows.Close()

	var items []MaintenanceTemplate
	for rows.Next() {
		var item MaintenanceTemplate
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.AssetID,
			&item.AssetName,
			&item.AssetCategoryID,
			&item.AssetCategoryName,
			&item.SourceDocumentID,
			&item.SourceDocumentTitle,
			&item.SourceRef,
			&item.FrequencyCode,
			&item.MaintenanceType,
			&item.ProcedureSummary,
			&item.ValidationCriteria,
			&item.RequiresChecklist,
			&item.RequiresSupervisor,
			&item.RequiresQualifiedPersonnel,
			&item.Priority,
			&item.EstimatedMinutes,
			&item.Active,
		); err != nil {
			return nil, fmt.Errorf("models: scan maintenance template: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func MaintenanceTemplateByID(ctx context.Context, d *db.DB, id int64) (*MaintenanceTemplateInput, error) {
	var (
		item             MaintenanceTemplateInput
		assetID          sql.NullInt64
		assetCategoryID  sql.NullInt64
		sourceDocumentID sql.NullInt64
	)

	err := d.Read.QueryRowContext(ctx,
		`SELECT
		    name,
		    asset_id,
		    asset_category_id,
		    source_document_id,
		    source_ref,
		    frequency_code,
		    maintenance_type,
		    procedure_summary,
		    validation_criteria,
		    requires_checklist,
		    requires_supervisor,
		    requires_qualified_personnel,
		    priority,
		    estimated_minutes,
		    active
		 FROM maintenance_templates
		 WHERE id = ?`,
		id,
	).Scan(
		&item.Name,
		&assetID,
		&assetCategoryID,
		&sourceDocumentID,
		&item.SourceRef,
		&item.FrequencyCode,
		&item.MaintenanceType,
		&item.ProcedureSummary,
		&item.ValidationCriteria,
		&item.RequiresChecklist,
		&item.RequiresSupervisor,
		&item.RequiresQualifiedPersonnel,
		&item.Priority,
		&item.EstimatedMinutes,
		&item.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: maintenance template by id: %w", err)
	}
	if assetID.Valid {
		item.AssetID = assetID.Int64
	}
	if assetCategoryID.Valid {
		item.AssetCategoryID = assetCategoryID.Int64
	}
	if sourceDocumentID.Valid {
		item.SourceDocumentID = sourceDocumentID.Int64
	}
	return &item, nil
}

func CreateMaintenanceTemplate(ctx context.Context, d *db.DB, input MaintenanceTemplateInput) (int64, error) {
	if err := validateTemplateInput(ctx, d, input); err != nil {
		return 0, err
	}
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO maintenance_templates (
		    name,
		    asset_id,
		    asset_category_id,
		    source_document_id,
		    source_ref,
		    frequency_code,
		    maintenance_type,
		    procedure_summary,
		    validation_criteria,
		    requires_checklist,
		    requires_supervisor,
		    requires_qualified_personnel,
		    priority,
		    estimated_minutes,
		    active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Name),
		nullableID(input.AssetID),
		nullableID(input.AssetCategoryID),
		nullableID(input.SourceDocumentID),
		strings.TrimSpace(input.SourceRef),
		input.FrequencyCode,
		input.MaintenanceType,
		strings.TrimSpace(input.ProcedureSummary),
		strings.TrimSpace(input.ValidationCriteria),
		boolToInt(input.RequiresChecklist),
		boolToInt(input.RequiresSupervisor),
		boolToInt(input.RequiresQualifiedPersonnel),
		input.Priority,
		input.EstimatedMinutes,
		boolToInt(input.Active),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create maintenance template: %w", err)
	}
	return res.LastInsertId()
}

func UpdateMaintenanceTemplate(ctx context.Context, d *db.DB, templateID int64, input MaintenanceTemplateInput) error {
	if err := validateTemplateInput(ctx, d, input); err != nil {
		return err
	}
	res, err := d.Write.ExecContext(ctx,
		`UPDATE maintenance_templates
		 SET name = ?,
		     asset_id = ?,
		     asset_category_id = ?,
		     source_document_id = ?,
		     source_ref = ?,
		     frequency_code = ?,
		     maintenance_type = ?,
		     procedure_summary = ?,
		     validation_criteria = ?,
		     requires_checklist = ?,
		     requires_supervisor = ?,
		     requires_qualified_personnel = ?,
		     priority = ?,
		     estimated_minutes = ?,
		     active = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		strings.TrimSpace(input.Name),
		nullableID(input.AssetID),
		nullableID(input.AssetCategoryID),
		nullableID(input.SourceDocumentID),
		strings.TrimSpace(input.SourceRef),
		input.FrequencyCode,
		input.MaintenanceType,
		strings.TrimSpace(input.ProcedureSummary),
		strings.TrimSpace(input.ValidationCriteria),
		boolToInt(input.RequiresChecklist),
		boolToInt(input.RequiresSupervisor),
		boolToInt(input.RequiresQualifiedPersonnel),
		input.Priority,
		input.EstimatedMinutes,
		boolToInt(input.Active),
		templateID,
	)
	if err != nil {
		return fmt.Errorf("models: update maintenance template: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("models: update maintenance template: template %d not found", templateID)
	}
	return nil
}

func ListScheduledTasks(ctx context.Context, d *db.DB, filters ScheduleFilters) ([]ScheduledTask, error) {
	var (
		args []any
		sqlb strings.Builder
	)

	sqlb.WriteString(`
SELECT
    st.id,
    COALESCE(st.template_id, 0),
    COALESCE(mt.name, ''),
    COALESCE(st.asset_id, 0),
    COALESCE(a.name, ''),
    COALESCE(st.source_document_id, 0),
    COALESCE(td.title, ''),
    st.title,
    st.frequency_code,
    st.maintenance_type,
    st.status,
    st.scheduled_for,
    st.window_start,
    st.window_end,
    st.publication_notes,
    COALESCE(uc.display_name, ''),
    COALESCE(up.display_name, ''),
    st.published_at
FROM scheduled_tasks st
LEFT JOIN maintenance_templates mt ON mt.id = st.template_id
LEFT JOIN assets a ON a.id = st.asset_id
LEFT JOIN technical_documents td ON td.id = st.source_document_id
LEFT JOIN users uc ON uc.id = st.created_by
LEFT JOIN users up ON up.id = st.published_by
WHERE 1 = 1`)

	if status := strings.TrimSpace(filters.Status); status != "" {
		sqlb.WriteString(` AND st.status = ?`)
		args = append(args, status)
	}
	if filters.AssetID > 0 {
		sqlb.WriteString(` AND st.asset_id = ?`)
		args = append(args, filters.AssetID)
	}
	if filters.TemplateID > 0 {
		sqlb.WriteString(` AND st.template_id = ?`)
		args = append(args, filters.TemplateID)
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

	sqlb.WriteString(` ORDER BY st.scheduled_for, st.id`)

	rows, err := d.Read.QueryContext(ctx, sqlb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("models: list scheduled tasks: %w", err)
	}
	defer rows.Close()

	var items []ScheduledTask
	for rows.Next() {
		var item ScheduledTask
		if err := rows.Scan(
			&item.ID,
			&item.TemplateID,
			&item.TemplateName,
			&item.AssetID,
			&item.AssetName,
			&item.SourceDocumentID,
			&item.SourceDocumentTitle,
			&item.Title,
			&item.FrequencyCode,
			&item.MaintenanceType,
			&item.Status,
			&item.ScheduledFor,
			&item.WindowStart,
			&item.WindowEnd,
			&item.PublicationNotes,
			&item.CreatedByName,
			&item.PublishedByName,
			&item.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("models: scan scheduled task: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func CreateScheduledTask(ctx context.Context, d *db.DB, input ScheduledTaskInput) (int64, error) {
	if err := validateScheduledTaskInput(ctx, d, input); err != nil {
		return 0, err
	}
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO scheduled_tasks (
		    template_id,
		    asset_id,
		    source_document_id,
		    title,
		    frequency_code,
		    maintenance_type,
		    status,
		    scheduled_for,
		    window_start,
		    window_end,
		    publication_notes,
		    created_by,
		    published_by,
		    published_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableID(input.TemplateID),
		nullableID(input.AssetID),
		nullableID(input.SourceDocumentID),
		strings.TrimSpace(input.Title),
		input.FrequencyCode,
		input.MaintenanceType,
		input.Status,
		input.ScheduledFor,
		strings.TrimSpace(input.WindowStart),
		strings.TrimSpace(input.WindowEnd),
		strings.TrimSpace(input.PublicationNotes),
		nullableID(input.CreatedBy),
		nullableID(input.PublishedBy),
		strings.TrimSpace(input.PublishedAt),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create scheduled task: %w", err)
	}
	return res.LastInsertId()
}

func CreateScheduledTaskFromTemplate(ctx context.Context, d *db.DB, templateID int64, scheduledFor string, createdBy int64, status string, publicationNotes string) (int64, error) {
	if strings.TrimSpace(status) == "" {
		status = PlannedStatusScheduled
	}

	template, err := loadTemplateSummary(ctx, d, templateID)
	if err != nil {
		return 0, err
	}
	if template == nil {
		return 0, fmt.Errorf("models: template %d not found", templateID)
	}

	input := ScheduledTaskInput{
		TemplateID:       template.ID,
		AssetID:          template.AssetID,
		SourceDocumentID: template.SourceDocumentID,
		Title:            template.Name,
		FrequencyCode:    template.FrequencyCode,
		MaintenanceType:  template.MaintenanceType,
		Status:           status,
		ScheduledFor:     scheduledFor,
		PublicationNotes: publicationNotes,
		CreatedBy:        createdBy,
	}
	if status == PlannedStatusPublished {
		input.PublishedBy = createdBy
		input.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return CreateScheduledTask(ctx, d, input)
}

func NextFrequencyDate(base time.Time, frequencyCode string) (time.Time, error) {
	switch frequencyCode {
	case FrequencyDaily:
		return base.AddDate(0, 0, 1), nil
	case FrequencyWeekly:
		return base.AddDate(0, 0, 7), nil
	case FrequencyMonthly:
		return base.AddDate(0, 1, 0), nil
	case FrequencyQuarterly:
		return base.AddDate(0, 3, 0), nil
	case FrequencySemiAnnual:
		return base.AddDate(0, 6, 0), nil
	case FrequencyYearly:
		return base.AddDate(1, 0, 0), nil
	case FrequencyConditional:
		return time.Time{}, fmt.Errorf("models: conditional frequency does not generate automatic dates")
	default:
		return time.Time{}, fmt.Errorf("models: invalid frequency code %q", frequencyCode)
	}
}

func loadTemplateSummary(ctx context.Context, d *db.DB, templateID int64) (*MaintenanceTemplate, error) {
	var (
		item             MaintenanceTemplate
		assetID          sql.NullInt64
		sourceDocumentID sql.NullInt64
	)
	err := d.Read.QueryRowContext(ctx,
		`SELECT id, name, asset_id, source_document_id, frequency_code, maintenance_type
		 FROM maintenance_templates
		 WHERE id = ?`,
		templateID,
	).Scan(
		&item.ID,
		&item.Name,
		&assetID,
		&sourceDocumentID,
		&item.FrequencyCode,
		&item.MaintenanceType,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: load template summary: %w", err)
	}
	if assetID.Valid {
		item.AssetID = assetID.Int64
	}
	if sourceDocumentID.Valid {
		item.SourceDocumentID = sourceDocumentID.Int64
	}
	return &item, nil
}

func validateTemplateInput(ctx context.Context, d *db.DB, input MaintenanceTemplateInput) error {
	switch {
	case strings.TrimSpace(input.Name) == "":
		return fmt.Errorf("models: template name is required")
	case strings.TrimSpace(input.ProcedureSummary) == "":
		return fmt.Errorf("models: template procedure summary is required")
	case input.EstimatedMinutes <= 0:
		return fmt.Errorf("models: estimated minutes must be positive")
	case !validFrequencyCode(input.FrequencyCode):
		return fmt.Errorf("models: invalid frequency code")
	case !validMaintenanceType(input.MaintenanceType):
		return fmt.Errorf("models: invalid maintenance type")
	case !validPlanningPriority(input.Priority):
		return fmt.Errorf("models: invalid priority")
	}

	if input.AssetID > 0 {
		asset, err := AssetByID(ctx, d, input.AssetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return fmt.Errorf("models: asset %d not found", input.AssetID)
		}
	}
	if input.AssetCategoryID > 0 {
		items, err := ListAssetCategories(ctx, d)
		if err != nil {
			return err
		}
		found := false
		for _, item := range items {
			if item.ID == input.AssetCategoryID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("models: asset category %d not found", input.AssetCategoryID)
		}
	}
	if input.SourceDocumentID > 0 {
		docs, err := ListTechnicalDocuments(ctx, d)
		if err != nil {
			return err
		}
		found := false
		for _, doc := range docs {
			if doc.ID == input.SourceDocumentID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("models: source document %d not found", input.SourceDocumentID)
		}
	}
	return nil
}

func validateScheduledTaskInput(ctx context.Context, d *db.DB, input ScheduledTaskInput) error {
	switch {
	case strings.TrimSpace(input.Title) == "":
		return fmt.Errorf("models: scheduled task title is required")
	case strings.TrimSpace(input.ScheduledFor) == "":
		return fmt.Errorf("models: scheduled date is required")
	case !validFrequencyCode(input.FrequencyCode):
		return fmt.Errorf("models: invalid frequency code")
	case !validMaintenanceType(input.MaintenanceType):
		return fmt.Errorf("models: invalid maintenance type")
	case !validPlannedStatus(input.Status):
		return fmt.Errorf("models: invalid planned status")
	}
	if _, err := time.Parse("2006-01-02", input.ScheduledFor); err != nil {
		return fmt.Errorf("models: scheduled date must use YYYY-MM-DD")
	}

	if input.TemplateID > 0 {
		template, err := loadTemplateSummary(ctx, d, input.TemplateID)
		if err != nil {
			return err
		}
		if template == nil {
			return fmt.Errorf("models: template %d not found", input.TemplateID)
		}
	}
	if input.AssetID > 0 {
		asset, err := AssetByID(ctx, d, input.AssetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return fmt.Errorf("models: asset %d not found", input.AssetID)
		}
	}
	if input.SourceDocumentID > 0 {
		docs, err := ListTechnicalDocuments(ctx, d)
		if err != nil {
			return err
		}
		found := false
		for _, doc := range docs {
			if doc.ID == input.SourceDocumentID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("models: source document %d not found", input.SourceDocumentID)
		}
	}
	return nil
}

func validFrequencyCode(code string) bool {
	switch code {
	case FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyQuarterly, FrequencySemiAnnual, FrequencyYearly, FrequencyConditional:
		return true
	default:
		return false
	}
}

func validMaintenanceType(maintenanceType string) bool {
	switch maintenanceType {
	case MaintenanceTypePreventive, MaintenanceTypeInspection, MaintenanceTypeCleaning, MaintenanceTypeSafety, MaintenanceTypeCorrective:
		return true
	default:
		return false
	}
}

func validPlanningPriority(priority string) bool {
	switch priority {
	case AssetCriticalityLow, AssetCriticalityMedium, AssetCriticalityHigh, AssetCriticalityCritical:
		return true
	default:
		return false
	}
}

func validPlannedStatus(status string) bool {
	switch status {
	case PlannedStatusDraft, PlannedStatusScheduled, PlannedStatusPublished, PlannedStatusCancelled, PlannedStatusReprogrammed, PlannedStatusCompleted:
		return true
	default:
		return false
	}
}
