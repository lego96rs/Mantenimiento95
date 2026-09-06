package models

import (
	"context"
	"testing"
)

func TestExecutionWorkOrdersAndIncidents(t *testing.T) {
	ctx := context.Background()
	database := openModelTestDB(t)

	plannerID, err := CreateUser(ctx, database, "planner1", "Planner Uno", RolePlanner, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser planner: %v", err)
	}
	technicianID, err := CreateUser(ctx, database, "tech1", "Técnico Uno", RoleTechnician, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser technician: %v", err)
	}
	supervisorID, err := CreateUser(ctx, database, "sup1", "Supervisor Uno", RoleSupervisor, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser supervisor: %v", err)
	}

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
		Code:             "EQ-200",
		Name:             "Transportador secundario",
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
		Name:                       "Inspección operativa",
		AssetID:                    assetID,
		AssetCategoryID:            categoryID,
		SourceDocumentID:           documentID,
		SourceRef:                  "Capítulo 6",
		FrequencyCode:              FrequencyMonthly,
		MaintenanceType:            MaintenanceTypeInspection,
		ProcedureSummary:           "Revisar estado general del equipo",
		ValidationCriteria:         "Sin daños visibles",
		RequiresChecklist:          true,
		RequiresSupervisor:         false,
		RequiresQualifiedPersonnel: false,
		Priority:                   AssetCriticalityHigh,
		EstimatedMinutes:           75,
		Active:                     true,
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceTemplate: %v", err)
	}

	scheduledTaskID, err := CreateScheduledTaskFromTemplate(ctx, database, templateID, "2026-04-10", plannerID, PlannedStatusPublished, "Turno mañana")
	if err != nil {
		t.Fatalf("CreateScheduledTaskFromTemplate: %v", err)
	}

	workOrderID, err := CreateWorkOrderFromScheduledTask(ctx, database, scheduledTaskID, plannerID)
	if err != nil {
		t.Fatalf("CreateWorkOrderFromScheduledTask: %v", err)
	}
	if _, err := CreateWorkOrderFromScheduledTask(ctx, database, scheduledTaskID, plannerID); err == nil {
		t.Fatal("expected duplicate work order creation to fail")
	}

	workOrders, err := ListWorkOrders(ctx, database, WorkOrderFilters{Status: WorkOrderStatusPending})
	if err != nil {
		t.Fatalf("ListWorkOrders: %v", err)
	}
	if len(workOrders) != 1 {
		t.Fatalf("len(workOrders) = %d, want 1", len(workOrders))
	}
	if workOrders[0].WorkOrderCode != "OT-000001" || workOrders[0].ScheduledFor != "2026-04-10" {
		t.Fatalf("work order = %#v", workOrders[0])
	}

	if err := SetWorkOrderAssignment(ctx, database, workOrderID, technicianID); err != nil {
		t.Fatalf("SetWorkOrderAssignment assign: %v", err)
	}
	if err := SetWorkOrderChecklist(ctx, database, workOrderID, []WorkOrderChecklistItemInput{
		{ItemText: "Verificar sensores"},
		{ItemText: "Validar guardas"},
	}); err != nil {
		t.Fatalf("SetWorkOrderChecklist: %v", err)
	}

	detail, err := WorkOrderByID(ctx, database, workOrderID)
	if err != nil {
		t.Fatalf("WorkOrderByID: %v", err)
	}
	if detail == nil || len(detail.Checklist) != 2 || detail.AssignedToName != "Técnico Uno" {
		t.Fatalf("detail after checklist = %#v", detail)
	}

	if err := UpdateWorkOrderProgress(ctx, database, workOrderID, WorkOrderProgressInput{
		Status:         WorkOrderStatusInProgress,
		ExecutionNotes: "Inicio de tarea",
		TotalMinutes:   15,
	}); err != nil {
		t.Fatalf("UpdateWorkOrderProgress in_progress: %v", err)
	}

	if err := UpdateWorkOrderProgress(ctx, database, workOrderID, WorkOrderProgressInput{
		Status:         WorkOrderStatusDone,
		ExecutionNotes: "Intento de cierre temprano",
		CloseSummary:   "No debería cerrar",
		TotalMinutes:   35,
	}); err == nil {
		t.Fatal("expected close with pending checklist to fail")
	}

	checklist, err := ListWorkOrderChecklist(ctx, database, workOrderID)
	if err != nil {
		t.Fatalf("ListWorkOrderChecklist: %v", err)
	}
	for _, item := range checklist {
		if err := UpdateChecklistItem(ctx, database, item.ID, true, "OK"); err != nil {
			t.Fatalf("UpdateChecklistItem(%d): %v", item.ID, err)
		}
	}

	incidentID, err := CreateIncident(ctx, database, IncidentInput{
		WorkOrderID:     workOrderID,
		Severity:        IncidentSeverityHigh,
		Title:           "Ruido anómalo",
		Description:     "Se detecta vibración durante la inspección",
		EscalationNotes: "Revisar soporte",
		ReportedBy:      technicianID,
	})
	if err != nil {
		t.Fatalf("CreateIncident: %v", err)
	}

	if err := UpdateWorkOrderProgress(ctx, database, workOrderID, WorkOrderProgressInput{
		Status:         WorkOrderStatusDone,
		ExecutionNotes: "Intento con incidente abierto",
		CloseSummary:   "Cierre no válido",
		TotalMinutes:   55,
	}); err == nil {
		t.Fatal("expected close with open incident to fail")
	}

	if err := UpdateIncident(ctx, database, incidentID, IncidentUpdateInput{
		Status:          IncidentStatusResolved,
		EscalationNotes: "Se ajustó soporte y desapareció la vibración",
		ResolvedBy:      supervisorID,
	}); err != nil {
		t.Fatalf("UpdateIncident resolved: %v", err)
	}

	if err := UpdateWorkOrderProgress(ctx, database, workOrderID, WorkOrderProgressInput{
		Status:         WorkOrderStatusDone,
		ExecutionNotes: "Inspección completada",
		CloseSummary:   "Equipo liberado para operación",
		TotalMinutes:   65,
	}); err != nil {
		t.Fatalf("UpdateWorkOrderProgress done: %v", err)
	}

	detail, err = WorkOrderByID(ctx, database, workOrderID)
	if err != nil {
		t.Fatalf("WorkOrderByID final: %v", err)
	}
	if detail.ExecutionStatus != WorkOrderStatusDone || detail.OpenIncidentCount != 0 || detail.EndTime == "" {
		t.Fatalf("final detail = %#v", detail)
	}

	incidents, err := ListIncidents(ctx, database, IncidentFilters{
		WorkOrderID: workOrderID,
		Status:      IncidentStatusResolved,
	})
	if err != nil {
		t.Fatalf("ListIncidents: %v", err)
	}
	if len(incidents) != 1 || incidents[0].ResolvedByName != "Supervisor Uno" {
		t.Fatalf("incidents = %#v", incidents)
	}

	tasks, err := ListScheduledTasks(ctx, database, ScheduleFilters{AssetID: assetID})
	if err != nil {
		t.Fatalf("ListScheduledTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Status != PlannedStatusCompleted {
		t.Fatalf("scheduled tasks = %#v", tasks)
	}
}

func TestExecutionWorkOrderRequiresPublishedTaskAndValidAssignee(t *testing.T) {
	ctx := context.Background()
	database := openModelTestDB(t)

	plannerID, err := CreateUser(ctx, database, "planner2", "Planner Dos", RolePlanner, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser planner: %v", err)
	}
	viewerID, err := CreateUser(ctx, database, "viewer1", "Viewer Uno", RoleViewer, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser viewer: %v", err)
	}

	areaID, err := CreateArea(ctx, database, "Soporte", "Área")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := CreateAssetCategory(ctx, database, "Panel", "Clase")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	documentID, err := CreateTechnicalDocument(ctx, database, "Local Control Cabinet", "docs/Local Control Cabinet 2.0 Streamline (Maintenance and Repair).pdf", DocumentTypeManual, "Cap 2", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument: %v", err)
	}
	assetID, err := CreateAsset(ctx, database, AssetInput{
		Code:             "EQ-300",
		Name:             "Panel local",
		AreaID:           areaID,
		CategoryID:       categoryID,
		OperationalState: AssetStateActive,
		Criticality:      AssetCriticalityMedium,
		Active:           true,
		DocumentIDs:      []int64{documentID},
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	templateID, err := CreateMaintenanceTemplate(ctx, database, MaintenanceTemplateInput{
		Name:               "Limpieza de gabinete",
		AssetID:            assetID,
		AssetCategoryID:    categoryID,
		SourceDocumentID:   documentID,
		FrequencyCode:      FrequencyMonthly,
		MaintenanceType:    MaintenanceTypeCleaning,
		ProcedureSummary:   "Limpiar exterior e interior",
		ValidationCriteria: "Sin polvo ni residuos",
		Priority:           AssetCriticalityMedium,
		EstimatedMinutes:   45,
		Active:             true,
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceTemplate: %v", err)
	}

	draftTaskID, err := CreateScheduledTaskFromTemplate(ctx, database, templateID, "2026-05-20", plannerID, PlannedStatusScheduled, "")
	if err != nil {
		t.Fatalf("CreateScheduledTaskFromTemplate draft: %v", err)
	}
	if _, err := CreateWorkOrderFromScheduledTask(ctx, database, draftTaskID, plannerID); err == nil {
		t.Fatal("expected work order creation from non-published task to fail")
	}

	publishedTaskID, err := CreateScheduledTaskFromTemplate(ctx, database, templateID, "2026-05-21", plannerID, PlannedStatusPublished, "")
	if err != nil {
		t.Fatalf("CreateScheduledTaskFromTemplate published: %v", err)
	}
	workOrderID, err := CreateWorkOrderFromScheduledTask(ctx, database, publishedTaskID, plannerID)
	if err != nil {
		t.Fatalf("CreateWorkOrderFromScheduledTask: %v", err)
	}
	if err := SetWorkOrderAssignment(ctx, database, workOrderID, viewerID); err == nil {
		t.Fatal("expected viewer assignment to fail")
	}
}
