package models

import (
	"context"
	"testing"
	"time"
)

func TestPlanningTemplatesAndSchedules(t *testing.T) {
	ctx := context.Background()
	database := openModelTestDB(t)

	areaID, err := CreateArea(ctx, database, "Operaciones", "Área principal")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := CreateAssetCategory(ctx, database, "Transportador", "Clase")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	documentID, err := CreateTechnicalDocument(ctx, database, "System Maintenance Plan", "docs/System Maintenance Plan.pdf", DocumentTypePlan, "Tabla 2", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument: %v", err)
	}
	assetID, err := CreateAsset(ctx, database, AssetInput{
		Code:             "EQ-100",
		Name:             "Transportador principal",
		AreaID:           areaID,
		CategoryID:       categoryID,
		OperationalState: AssetStateActive,
		Criticality:      AssetCriticalityHigh,
		Active:           true,
		DocumentIDs:      []int64{documentID},
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}

	templateID, err := CreateMaintenanceTemplate(ctx, database, MaintenanceTemplateInput{
		Name:                       "Inspección mensual de transportador",
		AssetID:                    assetID,
		AssetCategoryID:            categoryID,
		SourceDocumentID:           documentID,
		SourceRef:                  "Capítulo 4",
		FrequencyCode:              FrequencyMonthly,
		MaintenanceType:            MaintenanceTypeInspection,
		ProcedureSummary:           "Revisar rodillos, sensores y limpieza",
		ValidationCriteria:         "Sensores alineados y sin daños",
		RequiresChecklist:          true,
		RequiresSupervisor:         true,
		RequiresQualifiedPersonnel: false,
		Priority:                   AssetCriticalityHigh,
		EstimatedMinutes:           90,
		Active:                     true,
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceTemplate: %v", err)
	}

	templates, err := ListMaintenanceTemplates(ctx, database, TemplateFilters{AssetID: assetID, ActiveFilter: "active"})
	if err != nil {
		t.Fatalf("ListMaintenanceTemplates: %v", err)
	}
	if len(templates) != 1 || templates[0].SourceDocumentTitle != "System Maintenance Plan" {
		t.Fatalf("templates = %#v", templates)
	}

	date, err := NextFrequencyDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), FrequencyMonthly)
	if err != nil {
		t.Fatalf("NextFrequencyDate: %v", err)
	}
	if got := date.Format("2006-01-02"); got != "2026-02-15" {
		t.Fatalf("next monthly date = %s, want 2026-02-15", got)
	}

	if _, err := CreateScheduledTask(ctx, database, ScheduledTaskInput{
		TemplateID:       templateID,
		AssetID:          assetID,
		SourceDocumentID: documentID,
		Title:            "Inspección mensual manual",
		FrequencyCode:    FrequencyMonthly,
		MaintenanceType:  MaintenanceTypeInspection,
		Status:           PlannedStatusScheduled,
		ScheduledFor:     "2026-02-20",
		CreatedBy:        0,
	}); err != nil {
		t.Fatalf("CreateScheduledTask: %v", err)
	}

	if _, err := CreateScheduledTaskFromTemplate(ctx, database, templateID, "2026-03-20", 0, PlannedStatusPublished, "Turno nocturno"); err != nil {
		t.Fatalf("CreateScheduledTaskFromTemplate: %v", err)
	}

	tasks, err := ListScheduledTasks(ctx, database, ScheduleFilters{
		AssetID:  assetID,
		FromDate: "2026-02-01",
		ToDate:   "2026-03-31",
	})
	if err != nil {
		t.Fatalf("ListScheduledTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	if tasks[1].Status != PlannedStatusPublished || tasks[1].TemplateName == "" {
		t.Fatalf("published task = %#v", tasks[1])
	}
}
