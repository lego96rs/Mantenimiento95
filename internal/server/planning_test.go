package server

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"mantenimiento/internal/models"
)

func TestPlannerCanCreateTemplateAndScheduleFromIt(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "planner", "clave-segura-1", models.RolePlanner, false)

	areaID, err := models.CreateArea(context.Background(), ts.db, "Operaciones", "Área principal")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := models.CreateAssetCategory(context.Background(), ts.db, "Transportador", "Categoría")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	documentID, err := models.CreateTechnicalDocument(context.Background(), ts.db, "System Maintenance Plan", "docs/System Maintenance Plan.pdf", models.DocumentTypePlan, "Cap 4", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument: %v", err)
	}
	assetID, err := models.CreateAsset(context.Background(), ts.db, models.AssetInput{
		Code:             "EQ-200",
		Name:             "Conveyor agenda",
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

	_, cookie := ts.login(t, "planner", "clave-segura-1")

	templatePage := ts.do(t, http.MethodGet, "/templates/new", cookie, nil)
	if templatePage.Code != http.StatusOK {
		t.Fatalf("GET /templates/new: status=%d body=%s", templatePage.Code, templatePage.Body.String())
	}
	csrf := extractCSRF(t, templatePage.Body.String())

	createTemplate := ts.do(t, http.MethodPost, "/templates", cookie, url.Values{
		"csrf":                         {csrf},
		"name":                         {"Inspección mensual conveyor"},
		"asset_id":                     {strconv.FormatInt(assetID, 10)},
		"asset_category_id":            {strconv.FormatInt(categoryID, 10)},
		"source_document_id":           {strconv.FormatInt(documentID, 10)},
		"source_ref":                   {"Cap 4"},
		"frequency_code":               {models.FrequencyMonthly},
		"maintenance_type":             {models.MaintenanceTypeInspection},
		"procedure_summary":            {"Verificar sensores, rodillos y limpieza"},
		"validation_criteria":          {"Sin alarmas y con sensor alineado"},
		"priority":                     {models.AssetCriticalityHigh},
		"estimated_minutes":            {"75"},
		"requires_checklist":           {"on"},
		"requires_supervisor":          {"on"},
		"requires_qualified_personnel": {"on"},
		"active":                       {"on"},
	})
	if createTemplate.Code != http.StatusSeeOther || createTemplate.Header().Get("Location") != "/templates" {
		t.Fatalf("POST /templates: status=%d location=%q body=%s", createTemplate.Code, createTemplate.Header().Get("Location"), createTemplate.Body.String())
	}

	listTemplates, err := models.ListMaintenanceTemplates(context.Background(), ts.db, models.TemplateFilters{Query: "conveyor", ActiveFilter: "active"})
	if err != nil {
		t.Fatalf("ListMaintenanceTemplates: %v", err)
	}
	if len(listTemplates) != 1 {
		t.Fatalf("templates found = %d, want 1", len(listTemplates))
	}

	planningPage := ts.do(t, http.MethodGet, "/planning", cookie, nil)
	if planningPage.Code != http.StatusOK {
		t.Fatalf("GET /planning: status=%d body=%s", planningPage.Code, planningPage.Body.String())
	}
	csrf = extractCSRF(t, planningPage.Body.String())

	createSchedule := ts.do(t, http.MethodPost, "/planning/from-template", cookie, url.Values{
		"csrf":              {csrf},
		"template_id":       {strconv.FormatInt(listTemplates[0].ID, 10)},
		"scheduled_for":     {"2026-10-15"},
		"status":            {models.PlannedStatusPublished},
		"publication_notes": {"Primera publicación de prueba"},
	})
	if createSchedule.Code != http.StatusSeeOther || createSchedule.Header().Get("Location") != "/planning" {
		t.Fatalf("POST /planning/from-template: status=%d location=%q body=%s", createSchedule.Code, createSchedule.Header().Get("Location"), createSchedule.Body.String())
	}

	agenda := ts.do(t, http.MethodGet, "/planning?from=2026-10-01&to=2026-10-31", cookie, nil)
	if agenda.Code != http.StatusOK {
		t.Fatalf("GET /planning filtered: status=%d body=%s", agenda.Code, agenda.Body.String())
	}
	body := agenda.Body.String()
	for _, want := range []string{"Inspección mensual conveyor", "2026-10-15", "published"} {
		if !strings.Contains(body, want) {
			t.Fatalf("planning page missing %q in body: %s", want, body)
		}
	}
}

func TestTechnicianCanViewPlanningButCannotManageIt(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "tech", "clave-segura-1", models.RoleTechnician, false)

	_, cookie := ts.login(t, "tech", "clave-segura-1")
	if rec := ts.do(t, http.MethodGet, "/planning", cookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /planning technician: status=%d", rec.Code)
	}
	if rec := ts.do(t, http.MethodGet, "/templates", cookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /templates technician: status=%d", rec.Code)
	}
	if rec := ts.do(t, http.MethodGet, "/templates/new", cookie, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("GET /templates/new technician: status=%d, want 403", rec.Code)
	}
	if rec := ts.do(t, http.MethodPost, "/planning/tasks", cookie, url.Values{}); rec.Code != http.StatusForbidden {
		t.Fatalf("POST /planning/tasks technician: status=%d, want 403", rec.Code)
	}
}
