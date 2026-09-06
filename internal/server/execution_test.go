package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"mantenimiento/internal/models"
)

func TestPlannerCanCreateAssignAndTrackWorkOrder(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "planner", "clave-segura-1", models.RolePlanner, false)
	ts.createUser(t, "tech", "clave-segura-1", models.RoleTechnician, false)

	scheduledTaskID, techID := setupExecutionFixture(t, ts)

	_, plannerCookie := ts.login(t, "planner", "clave-segura-1")

	executionPage := ts.do(t, http.MethodGet, "/execution", plannerCookie, nil)
	if executionPage.Code != http.StatusOK {
		t.Fatalf("GET /execution: status=%d body=%s", executionPage.Code, executionPage.Body.String())
	}
	if !strings.Contains(executionPage.Body.String(), "Inspección publicada") {
		t.Fatalf("execution page missing published task option: %s", executionPage.Body.String())
	}
	csrf := extractCSRF(t, executionPage.Body.String())

	createWO := ts.do(t, http.MethodPost, "/execution/from-schedule", plannerCookie, url.Values{
		"csrf":             {csrf},
		"scheduled_task_id": {strconv.FormatInt(scheduledTaskID, 10)},
	})
	if createWO.Code != http.StatusSeeOther || createWO.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/from-schedule: status=%d location=%q body=%s", createWO.Code, createWO.Header().Get("Location"), createWO.Body.String())
	}

	workOrders, err := models.ListWorkOrders(context.Background(), ts.db, models.WorkOrderFilters{})
	if err != nil {
		t.Fatalf("ListWorkOrders: %v", err)
	}
	if len(workOrders) != 1 {
		t.Fatalf("len(workOrders) = %d, want 1", len(workOrders))
	}
	workOrderID := workOrders[0].ID

	assign := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/assign", workOrderID), plannerCookie, url.Values{
		"csrf":        {csrf},
		"assigned_to": {strconv.FormatInt(techID, 10)},
	})
	if assign.Code != http.StatusSeeOther || assign.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/{id}/assign: status=%d location=%q body=%s", assign.Code, assign.Header().Get("Location"), assign.Body.String())
	}

	checklist := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/checklist", workOrderID), plannerCookie, url.Values{
		"csrf":            {csrf},
		"checklist_items": {"Verificar sensores\nValidar guardas"},
	})
	if checklist.Code != http.StatusSeeOther || checklist.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/{id}/checklist: status=%d location=%q body=%s", checklist.Code, checklist.Header().Get("Location"), checklist.Body.String())
	}

	items, err := models.ListWorkOrderChecklist(context.Background(), ts.db, workOrderID)
	if err != nil {
		t.Fatalf("ListWorkOrderChecklist: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	_, techCookie := ts.login(t, "tech", "clave-segura-1")
	techPage := ts.do(t, http.MethodGet, "/execution", techCookie, nil)
	if techPage.Code != http.StatusOK {
		t.Fatalf("GET /execution tech: status=%d body=%s", techPage.Code, techPage.Body.String())
	}
	techBody := techPage.Body.String()
	for _, want := range []string{"OT-000001", "Nombre tech", "Verificar sensores"} {
		if !strings.Contains(techBody, want) {
			t.Fatalf("execution page missing %q in body: %s", want, techBody)
		}
	}
	csrf = extractCSRF(t, techBody)

	updateItem := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/checklist/%d", workOrderID, items[0].ID), techCookie, url.Values{
		"csrf":    {csrf},
		"is_done": {"on"},
		"notes":   {"Sin novedades"},
	})
	if updateItem.Code != http.StatusSeeOther || updateItem.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/{id}/checklist/{item}: status=%d location=%q body=%s", updateItem.Code, updateItem.Header().Get("Location"), updateItem.Body.String())
	}

	progress := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/progress", workOrderID), techCookie, url.Values{
		"csrf":            {csrf},
		"status":          {models.WorkOrderStatusInProgress},
		"total_minutes":   {"20"},
		"execution_notes": {"Inicio de inspección"},
		"close_summary":   {""},
	})
	if progress.Code != http.StatusSeeOther || progress.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/{id}/progress: status=%d location=%q body=%s", progress.Code, progress.Header().Get("Location"), progress.Body.String())
	}

	incident := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/incidents", workOrderID), techCookie, url.Values{
		"csrf":             {csrf},
		"severity":         {models.IncidentSeverityHigh},
		"title":            {"Vibración detectada"},
		"description":      {"Se detecta oscilación en soporte lateral"},
		"escalation_notes": {"Requiere revisión de ajuste"},
	})
	if incident.Code != http.StatusSeeOther || incident.Header().Get("Location") != "/execution" {
		t.Fatalf("POST /execution/{id}/incidents: status=%d location=%q body=%s", incident.Code, incident.Header().Get("Location"), incident.Body.String())
	}

	plannerPage := ts.do(t, http.MethodGet, "/execution", plannerCookie, nil)
	if plannerPage.Code != http.StatusOK {
		t.Fatalf("GET /execution planner final: status=%d body=%s", plannerPage.Code, plannerPage.Body.String())
	}
	finalBody := plannerPage.Body.String()
	for _, want := range []string{"OT-000001", "Nombre tech", "Vibración detectada", "in_progress"} {
		if !strings.Contains(finalBody, want) {
			t.Fatalf("final execution page missing %q in body: %s", want, finalBody)
		}
	}
}

func TestTechnicianCannotManageWorkOrderCreationOrAssignment(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "planner", "clave-segura-1", models.RolePlanner, false)
	ts.createUser(t, "tech", "clave-segura-1", models.RoleTechnician, false)

	scheduledTaskID, techID := setupExecutionFixture(t, ts)

	_, plannerCookie := ts.login(t, "planner", "clave-segura-1")
	page := ts.do(t, http.MethodGet, "/execution", plannerCookie, nil)
	csrf := extractCSRF(t, page.Body.String())
	if rec := ts.do(t, http.MethodPost, "/execution/from-schedule", plannerCookie, url.Values{
		"csrf":             {csrf},
		"scheduled_task_id": {strconv.FormatInt(scheduledTaskID, 10)},
	}); rec.Code != http.StatusSeeOther {
		t.Fatalf("planner create work order: status=%d body=%s", rec.Code, rec.Body.String())
	}

	workOrders, err := models.ListWorkOrders(context.Background(), ts.db, models.WorkOrderFilters{})
	if err != nil {
		t.Fatalf("ListWorkOrders: %v", err)
	}
	if len(workOrders) != 1 {
		t.Fatalf("len(workOrders) = %d, want 1", len(workOrders))
	}

	_, techCookie := ts.login(t, "tech", "clave-segura-1")
	techPage := ts.do(t, http.MethodGet, "/execution", techCookie, nil)
	if techPage.Code != http.StatusOK {
		t.Fatalf("GET /execution tech: status=%d body=%s", techPage.Code, techPage.Body.String())
	}
	homePage := ts.do(t, http.MethodGet, "/", techCookie, nil)
	if homePage.Code != http.StatusOK {
		t.Fatalf("GET / home tech: status=%d body=%s", homePage.Code, homePage.Body.String())
	}
	csrf = extractCSRF(t, homePage.Body.String())

	if rec := ts.do(t, http.MethodPost, "/execution/from-schedule", techCookie, url.Values{
		"csrf":             {csrf},
		"scheduled_task_id": {strconv.FormatInt(scheduledTaskID, 10)},
	}); rec.Code != http.StatusForbidden {
		t.Fatalf("POST /execution/from-schedule tech: status=%d, want 403", rec.Code)
	}

	if rec := ts.do(t, http.MethodPost, fmt.Sprintf("/execution/%d/assign", workOrders[0].ID), techCookie, url.Values{
		"csrf":        {csrf},
		"assigned_to": {strconv.FormatInt(techID, 10)},
	}); rec.Code != http.StatusForbidden {
		t.Fatalf("POST /execution/{id}/assign tech: status=%d, want 403", rec.Code)
	}
}

func setupExecutionFixture(t *testing.T, ts *testAuthServer) (int64, int64) {
	t.Helper()

	ctx := context.Background()
	areaID, err := models.CreateArea(ctx, ts.db, "Operaciones", "Área principal")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := models.CreateAssetCategory(ctx, ts.db, "Transportador", "Categoría")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	documentID, err := models.CreateTechnicalDocument(ctx, ts.db, "System Maintenance Plan", "docs/System Maintenance Plan.pdf", models.DocumentTypePlan, "Cap 4", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument: %v", err)
	}
	assetID, err := models.CreateAsset(ctx, ts.db, models.AssetInput{
		Code:             "EQ-500",
		Name:             "Conveyor ejecución",
		AreaID:           areaID,
		CategoryID:       categoryID,
		OperationalState: models.AssetStateActive,
		Criticality:      models.AssetCriticalityHigh,
		Active:           true,
		DocumentIDs:      []int64{documentID},
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	templateID, err := models.CreateMaintenanceTemplate(ctx, ts.db, models.MaintenanceTemplateInput{
		Name:               "Inspección publicada",
		AssetID:            assetID,
		AssetCategoryID:    categoryID,
		SourceDocumentID:   documentID,
		SourceRef:          "Cap 4",
		FrequencyCode:      models.FrequencyMonthly,
		MaintenanceType:    models.MaintenanceTypeInspection,
		ProcedureSummary:   "Revisar sensores y rodillos",
		ValidationCriteria: "Sin desalineación",
		Priority:           models.AssetCriticalityHigh,
		EstimatedMinutes:   60,
		Active:             true,
		RequiresChecklist:  true,
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceTemplate: %v", err)
	}

	planner, _, err := models.UserByUsername(ctx, ts.db, "planner")
	if err != nil || planner == nil {
		t.Fatalf("UserByUsername planner: user=%v err=%v", planner, err)
	}
	tech, _, err := models.UserByUsername(ctx, ts.db, "tech")
	if err != nil || tech == nil {
		t.Fatalf("UserByUsername tech: user=%v err=%v", tech, err)
	}

	scheduledTaskID, err := models.CreateScheduledTaskFromTemplate(ctx, ts.db, templateID, "2026-11-15", planner.ID, models.PlannedStatusPublished, "Ventana liberada")
	if err != nil {
		t.Fatalf("CreateScheduledTaskFromTemplate: %v", err)
	}
	return scheduledTaskID, tech.ID
}
