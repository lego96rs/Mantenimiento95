package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mantenimiento/internal/middleware"
	"mantenimiento/internal/models"
)

type assetFilterValues struct {
	Query        string
	AreaID       int64
	CategoryID   int64
	State        string
	Criticality  string
	ActiveFilter string
}

type assetsPageData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	CSRF        string
	Assets      []models.Asset
	Areas       []models.Area
	Categories  []models.AssetCategory
	Filters     assetFilterValues
	CanManage   bool
}

type assetInputValues struct {
	Code              string
	Name              string
	Family            string
	AreaID            int64
	CategoryID        int64
	Subarea           string
	Location          string
	Manufacturer      string
	Model             string
	SerialNumber      string
	OperationalState  string
	Criticality       string
	Notes             string
	Active            bool
	SelectedDocuments map[int64]bool
}

type assetFormData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	CSRF        string
	Error       string
	FormTitle   string
	Action      string
	SubmitLabel string
	Asset       assetInputValues
	Areas       []models.Area
	Categories  []models.AssetCategory
	Documents   []models.TechnicalDocument
}

type simpleCatalogForm struct {
	Name        string
	Description string
}

type documentCatalogForm struct {
	Title        string
	FilePath     string
	DocumentType string
	SourceRef    string
	Notes        string
}

type catalogsPageData struct {
	Title        string
	AppName      string
	Environment  string
	UserName     string
	RoleLabel    string
	CSRF         string
	Error        string
	Areas        []models.Area
	Categories   []models.AssetCategory
	Documents    []models.TechnicalDocument
	AreaForm     simpleCatalogForm
	CategoryForm simpleCatalogForm
	DocumentForm documentCatalogForm
}

func (s *Server) requireAssetManager(handler http.HandlerFunc) http.Handler {
	return middleware.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := middleware.SessionFrom(r)
		if !session.User.CanManageAssets() {
			http.Error(w, "acceso denegado", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	}))
}

func (s *Server) handleAssetsList(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	areas, categories, _, err := s.loadAssetCatalogs(r.Context())
	if err != nil {
		s.log.Error("load asset catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	filters := assetFilterValues{
		Query:        strings.TrimSpace(r.URL.Query().Get("q")),
		State:        strings.TrimSpace(r.URL.Query().Get("state")),
		Criticality:  strings.TrimSpace(r.URL.Query().Get("criticality")),
		ActiveFilter: strings.TrimSpace(r.URL.Query().Get("active")),
	}
	if filters.ActiveFilter == "" {
		filters.ActiveFilter = "active"
	}
	if filters.AreaID, err = parseOptionalInt64(r.URL.Query().Get("area_id")); err != nil {
		http.Error(w, "filtro de área inválido", http.StatusBadRequest)
		return
	}
	if filters.CategoryID, err = parseOptionalInt64(r.URL.Query().Get("category_id")); err != nil {
		http.Error(w, "filtro de categoría inválido", http.StatusBadRequest)
		return
	}

	assets, err := models.ListAssets(r.Context(), s.db, models.AssetFilters{
		Query:        filters.Query,
		AreaID:       filters.AreaID,
		CategoryID:   filters.CategoryID,
		State:        filters.State,
		Criticality:  filters.Criticality,
		ActiveFilter: filters.ActiveFilter,
	})
	if err != nil {
		s.log.Error("list assets", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	s.render(w, http.StatusOK, "assets", assetsPageData{
		Title:       "Activos",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		Assets:      assets,
		Areas:       areas,
		Categories:  categories,
		Filters:     filters,
		CanManage:   session.User.CanManageAssets(),
	})
}

func (s *Server) handleAssetNewPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	s.renderAssetForm(w, r.Context(), http.StatusOK, assetFormData{
		Title:       "Nuevo activo",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		FormTitle:   "Crear activo",
		Action:      "/assets",
		SubmitLabel: "Guardar activo",
		Asset: assetInputValues{
			OperationalState:  models.AssetStateActive,
			Criticality:       models.AssetCriticalityMedium,
			Active:            true,
			SelectedDocuments: map[int64]bool{},
		},
	})
}

func (s *Server) handleAssetCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	form, input, errMessage := parseAssetForm(r)
	if errMessage != "" {
		s.renderAssetForm(w, r.Context(), http.StatusUnprocessableEntity, assetFormData{
			Title:       "Nuevo activo",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       errMessage,
			FormTitle:   "Crear activo",
			Action:      "/assets",
			SubmitLabel: "Guardar activo",
			Asset:       form,
		})
		return
	}

	if _, err := models.CreateAsset(r.Context(), s.db, input); err != nil {
		s.log.Error("create asset", "err", err)
		s.renderAssetForm(w, r.Context(), http.StatusUnprocessableEntity, assetFormData{
			Title:       "Nuevo activo",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyAssetError(err),
			FormTitle:   "Crear activo",
			Action:      "/assets",
			SubmitLabel: "Guardar activo",
			Asset:       form,
		})
		return
	}

	http.Redirect(w, r, "/assets", http.StatusSeeOther)
}

func (s *Server) handleAssetEditPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	assetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || assetID <= 0 {
		http.NotFound(w, r)
		return
	}

	input, err := models.AssetByID(r.Context(), s.db, assetID)
	if err != nil {
		s.log.Error("asset by id", "id", assetID, "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if input == nil {
		http.NotFound(w, r)
		return
	}

	s.renderAssetForm(w, r.Context(), http.StatusOK, assetFormData{
		Title:       "Editar activo",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		FormTitle:   "Editar activo",
		Action:      fmt.Sprintf("/assets/%d", assetID),
		SubmitLabel: "Guardar cambios",
		Asset: assetInputValues{
			Code:              input.Code,
			Name:              input.Name,
			Family:            input.Family,
			AreaID:            input.AreaID,
			CategoryID:        input.CategoryID,
			Subarea:           input.Subarea,
			Location:          input.Location,
			Manufacturer:      input.Manufacturer,
			Model:             input.Model,
			SerialNumber:      input.SerialNumber,
			OperationalState:  input.OperationalState,
			Criticality:       input.Criticality,
			Notes:             input.Notes,
			Active:            input.Active,
			SelectedDocuments: selectedIDs(input.DocumentIDs),
		},
	})
}

func (s *Server) handleAssetUpdate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	assetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || assetID <= 0 {
		http.NotFound(w, r)
		return
	}

	form, input, errMessage := parseAssetForm(r)
	if errMessage != "" {
		s.renderAssetForm(w, r.Context(), http.StatusUnprocessableEntity, assetFormData{
			Title:       "Editar activo",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       errMessage,
			FormTitle:   "Editar activo",
			Action:      fmt.Sprintf("/assets/%d", assetID),
			SubmitLabel: "Guardar cambios",
			Asset:       form,
		})
		return
	}

	if err := models.UpdateAsset(r.Context(), s.db, assetID, input); err != nil {
		s.log.Error("update asset", "id", assetID, "err", err)
		s.renderAssetForm(w, r.Context(), http.StatusUnprocessableEntity, assetFormData{
			Title:       "Editar activo",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyAssetError(err),
			FormTitle:   "Editar activo",
			Action:      fmt.Sprintf("/assets/%d", assetID),
			SubmitLabel: "Guardar cambios",
			Asset:       form,
		})
		return
	}

	http.Redirect(w, r, "/assets", http.StatusSeeOther)
}

func (s *Server) handleCatalogsPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	s.renderCatalogsPage(w, r.Context(), http.StatusOK, catalogsPageData{
		Title:       "Catálogos",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		DocumentForm: documentCatalogForm{
			DocumentType: models.DocumentTypeManual,
		},
	})
}

func (s *Server) handleAreaCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:       "Catálogos",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       "El área debe tener nombre.",
			AreaForm:    simpleCatalogForm{Name: name, Description: description},
			DocumentForm: documentCatalogForm{
				DocumentType: models.DocumentTypeManual,
			},
		})
		return
	}

	if _, err := models.CreateArea(r.Context(), s.db, name, description); err != nil {
		s.log.Error("create area", "err", err)
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:       "Catálogos",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyAssetError(err),
			AreaForm:    simpleCatalogForm{Name: name, Description: description},
			DocumentForm: documentCatalogForm{
				DocumentType: models.DocumentTypeManual,
			},
		})
		return
	}
	http.Redirect(w, r, "/catalogs", http.StatusSeeOther)
}

func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:        "Catálogos",
			AppName:      "Sistema de Mantenimiento",
			Environment:  s.cfg.Env,
			UserName:     session.User.DisplayName,
			RoleLabel:    session.User.Role,
			CSRF:         session.CSRF,
			Error:        "La categoría debe tener nombre.",
			CategoryForm: simpleCatalogForm{Name: name, Description: description},
			DocumentForm: documentCatalogForm{
				DocumentType: models.DocumentTypeManual,
			},
		})
		return
	}

	if _, err := models.CreateAssetCategory(r.Context(), s.db, name, description); err != nil {
		s.log.Error("create asset category", "err", err)
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:        "Catálogos",
			AppName:      "Sistema de Mantenimiento",
			Environment:  s.cfg.Env,
			UserName:     session.User.DisplayName,
			RoleLabel:    session.User.Role,
			CSRF:         session.CSRF,
			Error:        friendlyAssetError(err),
			CategoryForm: simpleCatalogForm{Name: name, Description: description},
			DocumentForm: documentCatalogForm{
				DocumentType: models.DocumentTypeManual,
			},
		})
		return
	}
	http.Redirect(w, r, "/catalogs", http.StatusSeeOther)
}

func (s *Server) handleDocumentCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	form := documentCatalogForm{
		Title:        strings.TrimSpace(r.FormValue("title")),
		FilePath:     strings.TrimSpace(r.FormValue("file_path")),
		DocumentType: strings.TrimSpace(r.FormValue("document_type")),
		SourceRef:    strings.TrimSpace(r.FormValue("source_ref")),
		Notes:        strings.TrimSpace(r.FormValue("notes")),
	}
	if form.DocumentType == "" {
		form.DocumentType = models.DocumentTypeManual
	}

	if form.Title == "" || form.FilePath == "" {
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:        "Catálogos",
			AppName:      "Sistema de Mantenimiento",
			Environment:  s.cfg.Env,
			UserName:     session.User.DisplayName,
			RoleLabel:    session.User.Role,
			CSRF:         session.CSRF,
			Error:        "El documento técnico requiere título y ruta de archivo.",
			DocumentForm: form,
		})
		return
	}
	if !validDocumentType(form.DocumentType) {
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:        "Catálogos",
			AppName:      "Sistema de Mantenimiento",
			Environment:  s.cfg.Env,
			UserName:     session.User.DisplayName,
			RoleLabel:    session.User.Role,
			CSRF:         session.CSRF,
			Error:        "El tipo de documento no es válido.",
			DocumentForm: form,
		})
		return
	}

	if _, err := models.CreateTechnicalDocument(r.Context(), s.db, form.Title, form.FilePath, form.DocumentType, form.SourceRef, form.Notes); err != nil {
		s.log.Error("create technical document", "err", err)
		s.renderCatalogsPage(w, r.Context(), http.StatusUnprocessableEntity, catalogsPageData{
			Title:        "Catálogos",
			AppName:      "Sistema de Mantenimiento",
			Environment:  s.cfg.Env,
			UserName:     session.User.DisplayName,
			RoleLabel:    session.User.Role,
			CSRF:         session.CSRF,
			Error:        friendlyAssetError(err),
			DocumentForm: form,
		})
		return
	}
	http.Redirect(w, r, "/catalogs", http.StatusSeeOther)
}

func (s *Server) renderAssetForm(w http.ResponseWriter, ctx context.Context, status int, data assetFormData) {
	areas, categories, documents, err := s.loadAssetCatalogs(ctx)
	if err != nil {
		s.log.Error("load asset form catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	data.Areas = areas
	data.Categories = categories
	data.Documents = documents
	if data.Asset.SelectedDocuments == nil {
		data.Asset.SelectedDocuments = map[int64]bool{}
	}
	s.render(w, status, "asset_form", data)
}

func (s *Server) renderCatalogsPage(w http.ResponseWriter, ctx context.Context, status int, data catalogsPageData) {
	areas, categories, documents, err := s.loadAssetCatalogs(ctx)
	if err != nil {
		s.log.Error("load catalogs page", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	data.Areas = areas
	data.Categories = categories
	data.Documents = documents
	if data.DocumentForm.DocumentType == "" {
		data.DocumentForm.DocumentType = models.DocumentTypeManual
	}
	s.render(w, status, "catalogs", data)
}

func (s *Server) loadAssetCatalogs(ctx context.Context) ([]models.Area, []models.AssetCategory, []models.TechnicalDocument, error) {
	areas, err := models.ListAreas(ctx, s.db)
	if err != nil {
		return nil, nil, nil, err
	}
	categories, err := models.ListAssetCategories(ctx, s.db)
	if err != nil {
		return nil, nil, nil, err
	}
	documents, err := models.ListTechnicalDocuments(ctx, s.db)
	if err != nil {
		return nil, nil, nil, err
	}
	return areas, categories, documents, nil
}

func parseAssetForm(r *http.Request) (assetInputValues, models.AssetInput, string) {
	form := assetInputValues{
		Code:              strings.TrimSpace(r.FormValue("code")),
		Name:              strings.TrimSpace(r.FormValue("name")),
		Family:            strings.TrimSpace(r.FormValue("family")),
		Subarea:           strings.TrimSpace(r.FormValue("subarea")),
		Location:          strings.TrimSpace(r.FormValue("location")),
		Manufacturer:      strings.TrimSpace(r.FormValue("manufacturer")),
		Model:             strings.TrimSpace(r.FormValue("model")),
		SerialNumber:      strings.TrimSpace(r.FormValue("serial_number")),
		OperationalState:  strings.TrimSpace(r.FormValue("operational_state")),
		Criticality:       strings.TrimSpace(r.FormValue("criticality")),
		Notes:             strings.TrimSpace(r.FormValue("notes")),
		Active:            r.FormValue("active") == "on",
		SelectedDocuments: map[int64]bool{},
	}
	if form.OperationalState == "" {
		form.OperationalState = models.AssetStateActive
	}
	if form.Criticality == "" {
		form.Criticality = models.AssetCriticalityMedium
	}

	var err error
	if form.AreaID, err = parseOptionalInt64(r.FormValue("area_id")); err != nil {
		return form, models.AssetInput{}, "El área seleccionada no es válida."
	}
	if form.CategoryID, err = parseOptionalInt64(r.FormValue("category_id")); err != nil {
		return form, models.AssetInput{}, "La categoría seleccionada no es válida."
	}

	documentIDs := make([]int64, 0, len(r.Form["document_ids"]))
	for _, raw := range r.Form["document_ids"] {
		id, err := parseOptionalInt64(raw)
		if err != nil {
			return form, models.AssetInput{}, "Uno de los documentos seleccionados no es válido."
		}
		if id > 0 {
			documentIDs = append(documentIDs, id)
			form.SelectedDocuments[id] = true
		}
	}

	switch {
	case form.Code == "":
		return form, models.AssetInput{}, "El activo debe tener un código interno."
	case form.Name == "":
		return form, models.AssetInput{}, "El activo debe tener un nombre."
	case !validAssetState(form.OperationalState):
		return form, models.AssetInput{}, "El estado operativo no es válido."
	case !validCriticality(form.Criticality):
		return form, models.AssetInput{}, "La criticidad no es válida."
	}

	return form, models.AssetInput{
		Code:             form.Code,
		Name:             form.Name,
		Family:           form.Family,
		AreaID:           form.AreaID,
		CategoryID:       form.CategoryID,
		Subarea:          form.Subarea,
		Location:         form.Location,
		Manufacturer:     form.Manufacturer,
		Model:            form.Model,
		SerialNumber:     form.SerialNumber,
		OperationalState: form.OperationalState,
		Criticality:      form.Criticality,
		Notes:            form.Notes,
		Active:           form.Active,
		DocumentIDs:      documentIDs,
	}, ""
}

func parseOptionalInt64(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid int64")
	}
	return value, nil
}

func selectedIDs(ids []int64) map[int64]bool {
	result := make(map[int64]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func validDocumentType(documentType string) bool {
	switch documentType {
	case models.DocumentTypeManual, models.DocumentTypePlan, models.DocumentTypeDatasheet, models.DocumentTypeInstruction, models.DocumentTypeOther:
		return true
	default:
		return false
	}
}

func validAssetState(state string) bool {
	switch state {
	case models.AssetStateActive, models.AssetStateMaintenance, models.AssetStateFault, models.AssetStateInactive, models.AssetStateRetired:
		return true
	default:
		return false
	}
}

func validCriticality(criticality string) bool {
	switch criticality {
	case models.AssetCriticalityLow, models.AssetCriticalityMedium, models.AssetCriticalityHigh, models.AssetCriticalityCritical:
		return true
	default:
		return false
	}
}

func friendlyAssetError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique"), strings.Contains(message, "constraint"):
		return "Ya existe un registro con esos datos. Revisa código, nombre o ruta."
	case strings.Contains(message, "not found"):
		return "Uno de los catálogos seleccionados ya no existe."
	default:
		return "No se pudo guardar el registro. Revisa los datos e intenta de nuevo."
	}
}
