package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/middleware"
	"mantenimiento/internal/models"
)

type planningNavData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	CSRF        string
}

type templateFilterValues struct {
	Query           string
	AssetID         int64
	CategoryID      int64
	FrequencyCode   string
	MaintenanceType string
	ActiveFilter    string
}

type templatesPageData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	CSRF        string
	Error       string
	Templates   []models.MaintenanceTemplate
	Assets      []models.Asset
	Categories  []models.AssetCategory
	Filters     templateFilterValues
	CanManage   bool
}

type templateInputValues struct {
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

type templateFormData struct {
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
	Template    templateInputValues
	Assets      []models.Asset
	Categories  []models.AssetCategory
	Documents   []models.TechnicalDocument
}

type scheduleFilterValues struct {
	Status          string
	AssetID         int64
	TemplateID      int64
	FromDate        string
	ToDate          string
	MaintenanceType string
}

type scheduleInputValues struct {
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
}

type planningPageData struct {
	Title          string
	AppName        string
	Environment    string
	UserName       string
	RoleLabel      string
	CSRF           string
	Error          string
	ScheduledTasks []models.ScheduledTask
	Templates      []models.MaintenanceTemplate
	Assets         []models.Asset
	Filters        scheduleFilterValues
	Form           scheduleInputValues
	CanManage      bool
}

func (s *Server) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	assets, categories, documents, err := s.loadPlanningCatalogs(r.Context())
	if err != nil {
		s.log.Error("load planning catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	_ = documents

	filters := templateFilterValues{
		Query:           strings.TrimSpace(r.URL.Query().Get("q")),
		FrequencyCode:   strings.TrimSpace(r.URL.Query().Get("frequency")),
		MaintenanceType: strings.TrimSpace(r.URL.Query().Get("maintenance_type")),
		ActiveFilter:    strings.TrimSpace(r.URL.Query().Get("active")),
	}
	if filters.ActiveFilter == "" {
		filters.ActiveFilter = "active"
	}
	if filters.AssetID, err = parseOptionalInt64(r.URL.Query().Get("asset_id")); err != nil {
		http.Error(w, "filtro de activo inválido", http.StatusBadRequest)
		return
	}
	if filters.CategoryID, err = parseOptionalInt64(r.URL.Query().Get("category_id")); err != nil {
		http.Error(w, "filtro de categoría inválido", http.StatusBadRequest)
		return
	}

	templates, err := models.ListMaintenanceTemplates(r.Context(), s.db, models.TemplateFilters{
		Query:           filters.Query,
		AssetID:         filters.AssetID,
		AssetCategoryID: filters.CategoryID,
		FrequencyCode:   filters.FrequencyCode,
		MaintenanceType: filters.MaintenanceType,
		ActiveFilter:    filters.ActiveFilter,
	})
	if err != nil {
		s.log.Error("list maintenance templates", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	s.render(w, http.StatusOK, "templates", templatesPageData{
		Title:       "Plantillas",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		Templates:   templates,
		Assets:      assets,
		Categories:  categories,
		Filters:     filters,
		CanManage:   session.User.CanManageAssets(),
	})
}

func (s *Server) handleTemplateNewPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	s.renderTemplateForm(w, r.Context(), http.StatusOK, templateFormData{
		Title:       "Nueva plantilla",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		FormTitle:   "Crear plantilla de mantenimiento",
		Action:      "/templates",
		SubmitLabel: "Guardar plantilla",
		Template: templateInputValues{
			FrequencyCode:    models.FrequencyMonthly,
			MaintenanceType:  models.MaintenanceTypePreventive,
			Priority:         models.AssetCriticalityMedium,
			EstimatedMinutes: 60,
			Active:           true,
		},
	})
}

func (s *Server) handleTemplateCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	form, input, errMessage := parseTemplateForm(r)
	if errMessage != "" {
		s.renderTemplateForm(w, r.Context(), http.StatusUnprocessableEntity, templateFormData{
			Title:       "Nueva plantilla",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       errMessage,
			FormTitle:   "Crear plantilla de mantenimiento",
			Action:      "/templates",
			SubmitLabel: "Guardar plantilla",
			Template:    form,
		})
		return
	}

	if _, err := models.CreateMaintenanceTemplate(r.Context(), s.db, input); err != nil {
		s.log.Error("create maintenance template", "err", err)
		s.renderTemplateForm(w, r.Context(), http.StatusUnprocessableEntity, templateFormData{
			Title:       "Nueva plantilla",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyPlanningError(err),
			FormTitle:   "Crear plantilla de mantenimiento",
			Action:      "/templates",
			SubmitLabel: "Guardar plantilla",
			Template:    form,
		})
		return
	}

	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

func (s *Server) handleTemplateEditPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	templateID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || templateID <= 0 {
		http.NotFound(w, r)
		return
	}

	item, err := models.MaintenanceTemplateByID(r.Context(), s.db, templateID)
	if err != nil {
		s.log.Error("maintenance template by id", "id", templateID, "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if item == nil {
		http.NotFound(w, r)
		return
	}

	s.renderTemplateForm(w, r.Context(), http.StatusOK, templateFormData{
		Title:       "Editar plantilla",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		FormTitle:   "Editar plantilla de mantenimiento",
		Action:      fmt.Sprintf("/templates/%d", templateID),
		SubmitLabel: "Guardar cambios",
		Template: templateInputValues{
			Name:                       item.Name,
			AssetID:                    item.AssetID,
			AssetCategoryID:            item.AssetCategoryID,
			SourceDocumentID:           item.SourceDocumentID,
			SourceRef:                  item.SourceRef,
			FrequencyCode:              item.FrequencyCode,
			MaintenanceType:            item.MaintenanceType,
			ProcedureSummary:           item.ProcedureSummary,
			ValidationCriteria:         item.ValidationCriteria,
			RequiresChecklist:          item.RequiresChecklist,
			RequiresSupervisor:         item.RequiresSupervisor,
			RequiresQualifiedPersonnel: item.RequiresQualifiedPersonnel,
			Priority:                   item.Priority,
			EstimatedMinutes:           item.EstimatedMinutes,
			Active:                     item.Active,
		},
	})
}

func (s *Server) handleTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	templateID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || templateID <= 0 {
		http.NotFound(w, r)
		return
	}

	form, input, errMessage := parseTemplateForm(r)
	if errMessage != "" {
		s.renderTemplateForm(w, r.Context(), http.StatusUnprocessableEntity, templateFormData{
			Title:       "Editar plantilla",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       errMessage,
			FormTitle:   "Editar plantilla de mantenimiento",
			Action:      fmt.Sprintf("/templates/%d", templateID),
			SubmitLabel: "Guardar cambios",
			Template:    form,
		})
		return
	}

	if err := models.UpdateMaintenanceTemplate(r.Context(), s.db, templateID, input); err != nil {
		s.log.Error("update maintenance template", "id", templateID, "err", err)
		s.renderTemplateForm(w, r.Context(), http.StatusUnprocessableEntity, templateFormData{
			Title:       "Editar plantilla",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyPlanningError(err),
			FormTitle:   "Editar plantilla de mantenimiento",
			Action:      fmt.Sprintf("/templates/%d", templateID),
			SubmitLabel: "Guardar cambios",
			Template:    form,
		})
		return
	}

	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

func (s *Server) handlePlanningPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	assets, _, _, err := s.loadPlanningCatalogs(r.Context())
	if err != nil {
		s.log.Error("load planning catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	templates, err := models.ListMaintenanceTemplates(r.Context(), s.db, models.TemplateFilters{ActiveFilter: "active"})
	if err != nil {
		s.log.Error("list templates for planning", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	filters := scheduleFilterValues{
		Status:          strings.TrimSpace(r.URL.Query().Get("status")),
		FromDate:        strings.TrimSpace(r.URL.Query().Get("from")),
		ToDate:          strings.TrimSpace(r.URL.Query().Get("to")),
		MaintenanceType: strings.TrimSpace(r.URL.Query().Get("maintenance_type")),
	}
	if filters.FromDate == "" {
		filters.FromDate = time.Now().UTC().Format("2006-01-02")
	}
	if filters.ToDate == "" {
		filters.ToDate = time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02")
	}
	if filters.AssetID, err = parseOptionalInt64(r.URL.Query().Get("asset_id")); err != nil {
		http.Error(w, "filtro de activo inválido", http.StatusBadRequest)
		return
	}
	if filters.TemplateID, err = parseOptionalInt64(r.URL.Query().Get("template_id")); err != nil {
		http.Error(w, "filtro de plantilla inválido", http.StatusBadRequest)
		return
	}

	tasks, err := models.ListScheduledTasks(r.Context(), s.db, models.ScheduleFilters{
		Status:          filters.Status,
		AssetID:         filters.AssetID,
		TemplateID:      filters.TemplateID,
		FromDate:        filters.FromDate,
		ToDate:          filters.ToDate,
		MaintenanceType: filters.MaintenanceType,
	})
	if err != nil {
		s.log.Error("list scheduled tasks", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	s.render(w, http.StatusOK, "planning", planningPageData{
		Title:          "Planificación",
		AppName:        "Sistema de Mantenimiento",
		Environment:    s.cfg.Env,
		UserName:       session.User.DisplayName,
		RoleLabel:      session.User.Role,
		CSRF:           session.CSRF,
		ScheduledTasks: tasks,
		Templates:      templates,
		Assets:         assets,
		Filters:        filters,
		Form: scheduleInputValues{
			Status:          models.PlannedStatusScheduled,
			FrequencyCode:   models.FrequencyMonthly,
			MaintenanceType: models.MaintenanceTypePreventive,
			ScheduledFor:    time.Now().UTC().Format("2006-01-02"),
		},
		CanManage: session.User.CanManageAssets(),
	})
}

func (s *Server) handleScheduleCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	form, input, errMessage := parseScheduleForm(r, session.User.ID)
	if errMessage != "" {
		s.renderPlanningPage(w, r.Context(), http.StatusUnprocessableEntity, session, planningPageData{
			Title:       "Planificación",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       errMessage,
			Form:        form,
			CanManage:   session.User.CanManageAssets(),
		})
		return
	}

	if _, err := models.CreateScheduledTask(r.Context(), s.db, input); err != nil {
		s.log.Error("create scheduled task", "err", err)
		s.renderPlanningPage(w, r.Context(), http.StatusUnprocessableEntity, session, planningPageData{
			Title:       "Planificación",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyPlanningError(err),
			Form:        form,
			CanManage:   session.User.CanManageAssets(),
		})
		return
	}

	http.Redirect(w, r, "/planning", http.StatusSeeOther)
}

func (s *Server) handleScheduleFromTemplateCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	templateID, err := parseOptionalInt64(r.FormValue("template_id"))
	if err != nil || templateID <= 0 {
		http.Error(w, "plantilla inválida", http.StatusBadRequest)
		return
	}
	scheduledFor := strings.TrimSpace(r.FormValue("scheduled_for"))
	if scheduledFor == "" {
		scheduledFor = time.Now().UTC().Format("2006-01-02")
	}
	status := strings.TrimSpace(r.FormValue("status"))
	if status == "" {
		status = models.PlannedStatusScheduled
	}

	if _, err := models.CreateScheduledTaskFromTemplate(r.Context(), s.db, templateID, scheduledFor, session.User.ID, status, strings.TrimSpace(r.FormValue("publication_notes"))); err != nil {
		s.log.Error("create schedule from template", "err", err)
		s.renderPlanningPage(w, r.Context(), http.StatusUnprocessableEntity, session, planningPageData{
			Title:       "Planificación",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyPlanningError(err),
			Form: scheduleInputValues{
				TemplateID:   templateID,
				ScheduledFor: scheduledFor,
				Status:       status,
			},
			CanManage: session.User.CanManageAssets(),
		})
		return
	}

	http.Redirect(w, r, "/planning", http.StatusSeeOther)
}

func (s *Server) renderTemplateForm(w http.ResponseWriter, ctx context.Context, status int, data templateFormData) {
	assets, categories, documents, err := s.loadPlanningCatalogs(ctx)
	if err != nil {
		s.log.Error("load template form catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	data.Assets = assets
	data.Categories = categories
	data.Documents = documents
	if data.Template.FrequencyCode == "" {
		data.Template.FrequencyCode = models.FrequencyMonthly
	}
	if data.Template.MaintenanceType == "" {
		data.Template.MaintenanceType = models.MaintenanceTypePreventive
	}
	if data.Template.Priority == "" {
		data.Template.Priority = models.AssetCriticalityMedium
	}
	if data.Template.EstimatedMinutes <= 0 {
		data.Template.EstimatedMinutes = 60
	}
	s.render(w, status, "template_form", data)
}

func (s *Server) renderPlanningPage(w http.ResponseWriter, ctx context.Context, status int, session *auth.Session, data planningPageData) {
	assets, _, _, err := s.loadPlanningCatalogs(ctx)
	if err != nil {
		s.log.Error("load planning page catalogs", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	templates, err := models.ListMaintenanceTemplates(ctx, s.db, models.TemplateFilters{ActiveFilter: "active"})
	if err != nil {
		s.log.Error("load planning page templates", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	tasks, err := models.ListScheduledTasks(ctx, s.db, models.ScheduleFilters{
		Status:          data.Filters.Status,
		AssetID:         data.Filters.AssetID,
		TemplateID:      data.Filters.TemplateID,
		FromDate:        data.Filters.FromDate,
		ToDate:          data.Filters.ToDate,
		MaintenanceType: data.Filters.MaintenanceType,
	})
	if err != nil {
		s.log.Error("load planning page tasks", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	data.Assets = assets
	data.Templates = templates
	data.ScheduledTasks = tasks
	data.UserName = session.User.DisplayName
	data.RoleLabel = session.User.Role
	data.CSRF = session.CSRF
	if data.Form.Status == "" {
		data.Form.Status = models.PlannedStatusScheduled
	}
	if data.Form.FrequencyCode == "" {
		data.Form.FrequencyCode = models.FrequencyMonthly
	}
	if data.Form.MaintenanceType == "" {
		data.Form.MaintenanceType = models.MaintenanceTypePreventive
	}
	if data.Form.ScheduledFor == "" {
		data.Form.ScheduledFor = time.Now().UTC().Format("2006-01-02")
	}
	s.render(w, status, "planning", data)
}

func (s *Server) loadPlanningCatalogs(ctx context.Context) ([]models.Asset, []models.AssetCategory, []models.TechnicalDocument, error) {
	assets, err := models.ListAssets(ctx, s.db, models.AssetFilters{ActiveFilter: "active"})
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
	return assets, categories, documents, nil
}

func parseTemplateForm(r *http.Request) (templateInputValues, models.MaintenanceTemplateInput, string) {
	form := templateInputValues{
		Name:                       strings.TrimSpace(r.FormValue("name")),
		SourceRef:                  strings.TrimSpace(r.FormValue("source_ref")),
		FrequencyCode:              strings.TrimSpace(r.FormValue("frequency_code")),
		MaintenanceType:            strings.TrimSpace(r.FormValue("maintenance_type")),
		ProcedureSummary:           strings.TrimSpace(r.FormValue("procedure_summary")),
		ValidationCriteria:         strings.TrimSpace(r.FormValue("validation_criteria")),
		RequiresChecklist:          r.FormValue("requires_checklist") == "on",
		RequiresSupervisor:         r.FormValue("requires_supervisor") == "on",
		RequiresQualifiedPersonnel: r.FormValue("requires_qualified_personnel") == "on",
		Priority:                   strings.TrimSpace(r.FormValue("priority")),
		Active:                     r.FormValue("active") == "on",
	}
	var err error
	if form.AssetID, err = parseOptionalInt64(r.FormValue("asset_id")); err != nil {
		return form, models.MaintenanceTemplateInput{}, "El activo seleccionado no es válido."
	}
	if form.AssetCategoryID, err = parseOptionalInt64(r.FormValue("asset_category_id")); err != nil {
		return form, models.MaintenanceTemplateInput{}, "La categoría seleccionada no es válida."
	}
	if form.SourceDocumentID, err = parseOptionalInt64(r.FormValue("source_document_id")); err != nil {
		return form, models.MaintenanceTemplateInput{}, "El documento fuente seleccionado no es válido."
	}
	estimated, err := strconv.Atoi(strings.TrimSpace(r.FormValue("estimated_minutes")))
	if err != nil || estimated <= 0 {
		return form, models.MaintenanceTemplateInput{}, "La duración estimada debe ser un número positivo."
	}
	form.EstimatedMinutes = estimated

	switch {
	case form.Name == "":
		return form, models.MaintenanceTemplateInput{}, "La plantilla debe tener un nombre."
	case form.ProcedureSummary == "":
		return form, models.MaintenanceTemplateInput{}, "La plantilla debe incluir el procedimiento resumido."
	}
	if !validPlanningFrequency(form.FrequencyCode) {
		return form, models.MaintenanceTemplateInput{}, "La frecuencia no es válida."
	}
	if !validPlanningMaintenanceType(form.MaintenanceType) {
		return form, models.MaintenanceTemplateInput{}, "El tipo de mantenimiento no es válido."
	}
	if !validPlanningPriorityValue(form.Priority) {
		return form, models.MaintenanceTemplateInput{}, "La prioridad no es válida."
	}

	return form, models.MaintenanceTemplateInput{
		Name:                       form.Name,
		AssetID:                    form.AssetID,
		AssetCategoryID:            form.AssetCategoryID,
		SourceDocumentID:           form.SourceDocumentID,
		SourceRef:                  form.SourceRef,
		FrequencyCode:              form.FrequencyCode,
		MaintenanceType:            form.MaintenanceType,
		ProcedureSummary:           form.ProcedureSummary,
		ValidationCriteria:         form.ValidationCriteria,
		RequiresChecklist:          form.RequiresChecklist,
		RequiresSupervisor:         form.RequiresSupervisor,
		RequiresQualifiedPersonnel: form.RequiresQualifiedPersonnel,
		Priority:                   form.Priority,
		EstimatedMinutes:           form.EstimatedMinutes,
		Active:                     form.Active,
	}, ""
}

func parseScheduleForm(r *http.Request, userID int64) (scheduleInputValues, models.ScheduledTaskInput, string) {
	form := scheduleInputValues{
		Title:            strings.TrimSpace(r.FormValue("title")),
		FrequencyCode:    strings.TrimSpace(r.FormValue("frequency_code")),
		MaintenanceType:  strings.TrimSpace(r.FormValue("maintenance_type")),
		Status:           strings.TrimSpace(r.FormValue("status")),
		ScheduledFor:     strings.TrimSpace(r.FormValue("scheduled_for")),
		WindowStart:      strings.TrimSpace(r.FormValue("window_start")),
		WindowEnd:        strings.TrimSpace(r.FormValue("window_end")),
		PublicationNotes: strings.TrimSpace(r.FormValue("publication_notes")),
	}
	var err error
	if form.TemplateID, err = parseOptionalInt64(r.FormValue("template_id")); err != nil {
		return form, models.ScheduledTaskInput{}, "La plantilla seleccionada no es válida."
	}
	if form.AssetID, err = parseOptionalInt64(r.FormValue("asset_id")); err != nil {
		return form, models.ScheduledTaskInput{}, "El activo seleccionado no es válido."
	}
	if form.SourceDocumentID, err = parseOptionalInt64(r.FormValue("source_document_id")); err != nil {
		return form, models.ScheduledTaskInput{}, "El documento fuente seleccionado no es válido."
	}

	if form.Title == "" {
		return form, models.ScheduledTaskInput{}, "La programación debe tener un título."
	}
	if !validPlanningFrequency(form.FrequencyCode) {
		return form, models.ScheduledTaskInput{}, "La frecuencia no es válida."
	}
	if !validPlanningMaintenanceType(form.MaintenanceType) {
		return form, models.ScheduledTaskInput{}, "El tipo de mantenimiento no es válido."
	}
	if !validPlanningStatusValue(form.Status) {
		return form, models.ScheduledTaskInput{}, "El estado de planificación no es válido."
	}
	if _, err := time.Parse("2006-01-02", form.ScheduledFor); err != nil {
		return form, models.ScheduledTaskInput{}, "La fecha programada debe usar el formato YYYY-MM-DD."
	}

	input := models.ScheduledTaskInput{
		TemplateID:       form.TemplateID,
		AssetID:          form.AssetID,
		SourceDocumentID: form.SourceDocumentID,
		Title:            form.Title,
		FrequencyCode:    form.FrequencyCode,
		MaintenanceType:  form.MaintenanceType,
		Status:           form.Status,
		ScheduledFor:     form.ScheduledFor,
		WindowStart:      form.WindowStart,
		WindowEnd:        form.WindowEnd,
		PublicationNotes: form.PublicationNotes,
		CreatedBy:        userID,
	}
	if form.Status == models.PlannedStatusPublished {
		input.PublishedBy = userID
		input.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return form, input, ""
}

func friendlyPlanningError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"):
		return "Uno de los registros relacionados ya no existe."
	case strings.Contains(message, "required"):
		return "Faltan datos obligatorios para guardar el registro."
	case strings.Contains(message, "date"):
		return "La fecha programada no tiene un formato válido."
	case strings.Contains(message, "frequency"), strings.Contains(message, "maintenance type"), strings.Contains(message, "priority"), strings.Contains(message, "status"):
		return "Uno de los valores de planificación no es válido."
	default:
		return "No se pudo guardar la planificación. Revisa los datos e intenta de nuevo."
	}
}

func validPlanningFrequency(code string) bool {
	switch code {
	case models.FrequencyDaily, models.FrequencyWeekly, models.FrequencyMonthly, models.FrequencyQuarterly, models.FrequencySemiAnnual, models.FrequencyYearly, models.FrequencyConditional:
		return true
	default:
		return false
	}
}

func validPlanningMaintenanceType(value string) bool {
	switch value {
	case models.MaintenanceTypePreventive, models.MaintenanceTypeInspection, models.MaintenanceTypeCleaning, models.MaintenanceTypeSafety, models.MaintenanceTypeCorrective:
		return true
	default:
		return false
	}
}

func validPlanningPriorityValue(value string) bool {
	switch value {
	case models.AssetCriticalityLow, models.AssetCriticalityMedium, models.AssetCriticalityHigh, models.AssetCriticalityCritical:
		return true
	default:
		return false
	}
}

func validPlanningStatusValue(value string) bool {
	switch value {
	case models.PlannedStatusDraft, models.PlannedStatusScheduled, models.PlannedStatusPublished, models.PlannedStatusCancelled, models.PlannedStatusReprogrammed, models.PlannedStatusCompleted:
		return true
	default:
		return false
	}
}
