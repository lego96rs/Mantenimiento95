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

func TestPlannerCanCreateAssetAndListIt(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "planificador", "clave-segura-1", models.RolePlanner, false)

	areaID, err := models.CreateArea(context.Background(), ts.db, "Operaciones", "Área principal")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := models.CreateAssetCategory(context.Background(), ts.db, "Transportador", "Categoría")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	docID, err := models.CreateTechnicalDocument(context.Background(), ts.db, "System Maintenance Plan", "docs/System Maintenance Plan.pdf", models.DocumentTypePlan, "Tabla 2", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument: %v", err)
	}

	_, cookie := ts.login(t, "planificador", "clave-segura-1")
	page := ts.do(t, http.MethodGet, "/assets/new", cookie, nil)
	if page.Code != http.StatusOK {
		t.Fatalf("GET /assets/new: status=%d body=%s", page.Code, page.Body.String())
	}
	csrf := extractCSRF(t, page.Body.String())

	rec := ts.do(t, http.MethodPost, "/assets", cookie, url.Values{
		"csrf":              {csrf},
		"code":              {"EQ-001"},
		"name":              {"Conveyor principal"},
		"family":            {"Transportador"},
		"area_id":           {strconv.FormatInt(areaID, 10)},
		"category_id":       {strconv.FormatInt(categoryID, 10)},
		"location":          {"Pasillo 3"},
		"manufacturer":      {"KNAPP"},
		"model":             {"SRC"},
		"serial_number":     {"SN-001"},
		"operational_state": {models.AssetStateActive},
		"criticality":       {models.AssetCriticalityHigh},
		"document_ids":      {strconv.FormatInt(docID, 10)},
		"active":            {"on"},
	})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/assets" {
		t.Fatalf("POST /assets: status=%d location=%q body=%s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}

	list := ts.do(t, http.MethodGet, "/assets?q=EQ-001", cookie, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("GET /assets: status=%d body=%s", list.Code, list.Body.String())
	}
	body := list.Body.String()
	for _, want := range []string{"Conveyor principal", "Operaciones", "Transportador", "System Maintenance Plan"} {
		if !strings.Contains(body, want) {
			t.Fatalf("assets list missing %q in body: %s", want, body)
		}
	}
}

func TestTechnicianCanViewAssetsButCannotManageCatalogs(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "tecnico", "clave-segura-1", models.RoleTechnician, false)

	_, cookie := ts.login(t, "tecnico", "clave-segura-1")

	if rec := ts.do(t, http.MethodGet, "/assets", cookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /assets technician: status=%d", rec.Code)
	}
	if rec := ts.do(t, http.MethodGet, "/assets/new", cookie, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("GET /assets/new technician: status=%d, want 403", rec.Code)
	}
	if rec := ts.do(t, http.MethodGet, "/catalogs", cookie, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("GET /catalogs technician: status=%d, want 403", rec.Code)
	}
}
