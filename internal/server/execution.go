package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/middleware"
	"mantenimiento/internal/models"
)

type executionFilterValues struct {
	Status          string
	AssetID         int64
	AssignedTo      int64
	FromDate        string
	ToDate          string
	MaintenanceType string
}

type executionFormValues struct {
	ScheduledTaskID    int64
	AssignedTo         int64
	ChecklistItemsText string
	ProgressStatus     string
	ExecutionNotes     string
	CloseSummary       string
	TotalMinutes       int
	IncidentSeverity   string
	IncidentTitle      string
	IncidentDescription string
	IncidentNotes      string
}

type executionWorkOrderView struct {
	models.WorkOrderDetail
	CanOperate bool
}

type executionPageData struct {
	Title          string
	AppName        string
	Environment    string
	UserName       string
	RoleLabel      string
	CSRF           string
	Error          string
	WorkOrders     []executionWorkOrderView
	PublishedTasks []models.ScheduledTask
	Assets         []models.Asset
	AssignableUsers []models.User
	Filters        executionFilterValues
	Form           executionFormValues
	CanManage      bool
}

func (s *Server) requireExecutionManager(handler http.HandlerFunc) http.Handler {
	return middleware.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := middleware.SessionFrom(r)
		if !ok || !session.User.CanManageExecution() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		handler(w, r)
	}))
}

func (s *Server) handleExecutionPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	filters := executionFilterValues{
		Status:          strings.TrimSpace(r.URL.Query().Get("status")),
		FromDate:        strings.TrimSpace(r.URL.Query().Get("from")),
		ToDate:          strings.TrimSpace(r.URL.Query().Get("to")),
		MaintenanceType: strings.TrimSpace(r.URL.Query().Get("maintenance_type")),
	}
	var err error
	if filters.AssetID, err = parseOptionalInt64(r.URL.Query().Get("asset_id")); err != nil {
		http.Error(w, "filtro de activo inválido", http.StatusBadRequest)
		return
	}
	if filters.AssignedTo, err = parseOptionalInt64(r.URL.Query().Get("assigned_to")); err != nil {
		http.Error(w, "filtro de responsable inválido", http.StatusBadRequest)
		return
	}

	s.renderExecutionPage(w, r.Context(), http.StatusOK, session, executionPageData{
		Title:       "Ejecución",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
		Filters:     filters,
		CanManage:   session.User.CanManageExecution(),
		Form: executionFormValues{
			ProgressStatus:   models.WorkOrderStatusInProgress,
			IncidentSeverity: models.IncidentSeverityMedium,
		},
	})
}

func (s *Server) handleWorkOrderCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	scheduledTaskID, err := parseOptionalInt64(r.FormValue("scheduled_task_id"))
	if err != nil || scheduledTaskID <= 0 {
		http.Error(w, "tarea publicada inválida", http.StatusBadRequest)
		return
	}

	if _, err := models.CreateWorkOrderFromScheduledTask(r.Context(), s.db, scheduledTaskID, session.User.ID); err != nil {
		s.log.Error("create work order", "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ScheduledTaskID:  scheduledTaskID,
				ProgressStatus:   models.WorkOrderStatusInProgress,
				IncidentSeverity: models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleWorkOrderAssign(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	workOrderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workOrderID <= 0 {
		http.NotFound(w, r)
		return
	}

	assignedTo, err := parseOptionalInt64(r.FormValue("assigned_to"))
	if err != nil {
		http.Error(w, "responsable inválido", http.StatusBadRequest)
		return
	}

	if err := models.SetWorkOrderAssignment(r.Context(), s.db, workOrderID, assignedTo); err != nil {
		s.log.Error("assign work order", "id", workOrderID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				AssignedTo:       assignedTo,
				ProgressStatus:   models.WorkOrderStatusInProgress,
				IncidentSeverity: models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleWorkOrderChecklistSet(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	workOrderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workOrderID <= 0 {
		http.NotFound(w, r)
		return
	}

	items := parseChecklistItems(r.FormValue("checklist_items"))
	if len(items) == 0 {
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       "La checklist debe tener al menos un ítem.",
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ChecklistItemsText: r.FormValue("checklist_items"),
				ProgressStatus:     models.WorkOrderStatusInProgress,
				IncidentSeverity:   models.IncidentSeverityMedium,
			},
		})
		return
	}

	if err := models.SetWorkOrderChecklist(r.Context(), s.db, workOrderID, items); err != nil {
		s.log.Error("set work order checklist", "id", workOrderID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ChecklistItemsText: r.FormValue("checklist_items"),
				ProgressStatus:     models.WorkOrderStatusInProgress,
				IncidentSeverity:   models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleChecklistItemUpdate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	workOrderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workOrderID <= 0 {
		http.NotFound(w, r)
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("item_id"), 10, 64)
	if err != nil || itemID <= 0 {
		http.NotFound(w, r)
		return
	}

	workOrder, err := models.WorkOrderByID(r.Context(), s.db, workOrderID)
	if err != nil {
		s.log.Error("work order by id for checklist update", "id", workOrderID, "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if workOrder == nil {
		http.NotFound(w, r)
		return
	}
	if !canOperateExecutionWorkOrder(session, workOrder) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := models.UpdateChecklistItem(r.Context(), s.db, itemID, r.FormValue("is_done") == "on", r.FormValue("notes")); err != nil {
		s.log.Error("update checklist item", "id", itemID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ProgressStatus:   models.WorkOrderStatusInProgress,
				IncidentSeverity: models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleWorkOrderProgress(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	workOrderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workOrderID <= 0 {
		http.NotFound(w, r)
		return
	}

	workOrder, err := models.WorkOrderByID(r.Context(), s.db, workOrderID)
	if err != nil {
		s.log.Error("work order by id for progress", "id", workOrderID, "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if workOrder == nil {
		http.NotFound(w, r)
		return
	}
	if !canOperateExecutionWorkOrder(session, workOrder) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	totalMinutes, err := strconv.Atoi(strings.TrimSpace(r.FormValue("total_minutes")))
	if err != nil || totalMinutes < 0 {
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       "Los minutos reales deben ser un número válido.",
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ProgressStatus:     strings.TrimSpace(r.FormValue("status")),
				ExecutionNotes:     strings.TrimSpace(r.FormValue("execution_notes")),
				CloseSummary:       strings.TrimSpace(r.FormValue("close_summary")),
				TotalMinutes:       totalMinutes,
				IncidentSeverity:   models.IncidentSeverityMedium,
			},
		})
		return
	}

	if err := models.UpdateWorkOrderProgress(r.Context(), s.db, workOrderID, models.WorkOrderProgressInput{
		Status:         strings.TrimSpace(r.FormValue("status")),
		ExecutionNotes: strings.TrimSpace(r.FormValue("execution_notes")),
		CloseSummary:   strings.TrimSpace(r.FormValue("close_summary")),
		TotalMinutes:   totalMinutes,
	}); err != nil {
		s.log.Error("update work order progress", "id", workOrderID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ProgressStatus:     strings.TrimSpace(r.FormValue("status")),
				ExecutionNotes:     strings.TrimSpace(r.FormValue("execution_notes")),
				CloseSummary:       strings.TrimSpace(r.FormValue("close_summary")),
				TotalMinutes:       totalMinutes,
				IncidentSeverity:   models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleWorkOrderIncidentCreate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	workOrderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workOrderID <= 0 {
		http.NotFound(w, r)
		return
	}

	workOrder, err := models.WorkOrderByID(r.Context(), s.db, workOrderID)
	if err != nil {
		s.log.Error("work order by id for incident", "id", workOrderID, "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if workOrder == nil {
		http.NotFound(w, r)
		return
	}
	if !canOperateExecutionWorkOrder(session, workOrder) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if _, err := models.CreateIncident(r.Context(), s.db, models.IncidentInput{
		WorkOrderID:     workOrderID,
		Severity:        strings.TrimSpace(r.FormValue("severity")),
		Title:           strings.TrimSpace(r.FormValue("title")),
		Description:     strings.TrimSpace(r.FormValue("description")),
		EscalationNotes: strings.TrimSpace(r.FormValue("escalation_notes")),
		ReportedBy:      session.User.ID,
	}); err != nil {
		s.log.Error("create incident", "work_order_id", workOrderID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				IncidentSeverity:    strings.TrimSpace(r.FormValue("severity")),
				IncidentTitle:       strings.TrimSpace(r.FormValue("title")),
				IncidentDescription: strings.TrimSpace(r.FormValue("description")),
				IncidentNotes:       strings.TrimSpace(r.FormValue("escalation_notes")),
				ProgressStatus:      models.WorkOrderStatusInProgress,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) handleIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	incidentID, err := strconv.ParseInt(r.PathValue("incident_id"), 10, 64)
	if err != nil || incidentID <= 0 {
		http.NotFound(w, r)
		return
	}

	status := strings.TrimSpace(r.FormValue("status"))
	update := models.IncidentUpdateInput{
		Status:          status,
		EscalationNotes: strings.TrimSpace(r.FormValue("escalation_notes")),
	}
	if status == models.IncidentStatusResolved || status == models.IncidentStatusClosed {
		update.ResolvedBy = session.User.ID
	}

	if err := models.UpdateIncident(r.Context(), s.db, incidentID, update); err != nil {
		s.log.Error("update incident", "id", incidentID, "err", err)
		s.renderExecutionPage(w, r.Context(), http.StatusUnprocessableEntity, session, executionPageData{
			Title:       "Ejecución",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			CSRF:        session.CSRF,
			Error:       friendlyExecutionError(err),
			CanManage:   session.User.CanManageExecution(),
			Form: executionFormValues{
				ProgressStatus:   models.WorkOrderStatusInProgress,
				IncidentSeverity: models.IncidentSeverityMedium,
			},
		})
		return
	}

	http.Redirect(w, r, "/execution", http.StatusSeeOther)
}

func (s *Server) renderExecutionPage(w http.ResponseWriter, ctx context.Context, status int, session *auth.Session, data executionPageData) {
	workOrders, publishedTasks, assets, users, err := s.loadExecutionPageData(ctx, data.Filters)
	if err != nil {
		s.log.Error("load execution page data", "err", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	views := make([]executionWorkOrderView, 0, len(workOrders))
	for _, item := range workOrders {
		views = append(views, executionWorkOrderView{
			WorkOrderDetail: item,
			CanOperate:      canOperateExecutionWorkOrder(session, &item),
		})
	}

	data.WorkOrders = views
	data.PublishedTasks = publishedTasks
	data.Assets = assets
	data.AssignableUsers = users
	data.UserName = session.User.DisplayName
	data.RoleLabel = session.User.Role
	data.CSRF = session.CSRF
	data.CanManage = session.User.CanManageExecution()
	if data.Form.ProgressStatus == "" {
		data.Form.ProgressStatus = models.WorkOrderStatusInProgress
	}
	if data.Form.IncidentSeverity == "" {
		data.Form.IncidentSeverity = models.IncidentSeverityMedium
	}
	s.render(w, status, "execution", data)
}

func (s *Server) loadExecutionPageData(ctx context.Context, filters executionFilterValues) ([]models.WorkOrderDetail, []models.ScheduledTask, []models.Asset, []models.User, error) {
	workOrders, err := models.ListWorkOrders(ctx, s.db, models.WorkOrderFilters{
		Status:          filters.Status,
		AssetID:         filters.AssetID,
		AssignedTo:      filters.AssignedTo,
		FromDate:        filters.FromDate,
		ToDate:          filters.ToDate,
		MaintenanceType: filters.MaintenanceType,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	details := make([]models.WorkOrderDetail, 0, len(workOrders))
	usedTasks := make(map[int64]struct{}, len(workOrders))
	for _, item := range workOrders {
		usedTasks[item.ScheduledTaskID] = struct{}{}
		detail, err := models.WorkOrderByID(ctx, s.db, item.ID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if detail != nil {
			details = append(details, *detail)
		}
	}

	publishedTasks, err := models.ListScheduledTasks(ctx, s.db, models.ScheduleFilters{Status: models.PlannedStatusPublished})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	availableTasks := make([]models.ScheduledTask, 0, len(publishedTasks))
	for _, item := range publishedTasks {
		if _, ok := usedTasks[item.ID]; ok {
			continue
		}
		availableTasks = append(availableTasks, item)
	}

	assets, err := models.ListAssets(ctx, s.db, models.AssetFilters{ActiveFilter: "active"})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	users, err := models.ListActiveUsers(ctx, s.db, models.RoleAdmin, models.RolePlanner, models.RoleSupervisor, models.RoleTechnician)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return details, availableTasks, assets, users, nil
}

func canOperateExecutionWorkOrder(session *auth.Session, workOrder *models.WorkOrderDetail) bool {
	if session == nil || workOrder == nil {
		return false
	}
	if session.User.CanManageExecution() {
		return true
	}
	return session.User.Role == models.RoleTechnician && workOrder.AssignedTo == session.User.ID
}

func parseChecklistItems(raw string) []models.WorkOrderChecklistItemInput {
	lines := strings.Split(raw, "\n")
	items := make([]models.WorkOrderChecklistItemInput, 0, len(lines))
	for i, line := range lines {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		items = append(items, models.WorkOrderChecklistItemInput{
			ItemText:  text,
			SortOrder: i + 1,
		})
	}
	return items
}

func friendlyExecutionError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "published before creating"):
		return "Solo las tareas publicadas pueden convertirse en orden de trabajo."
	case strings.Contains(message, "already exists"):
		return "La orden de trabajo ya existe para esa tarea publicada."
	case strings.Contains(message, "must be assigned before starting"):
		return "Asigna un responsable antes de iniciar la orden."
	case strings.Contains(message, "all checklist items"), strings.Contains(message, "requires a checklist"):
		return "La checklist debe estar completa antes de cerrar la orden."
	case strings.Contains(message, "open incidents"):
		return "No puedes cerrar la orden mientras existan incidentes abiertos."
	case strings.Contains(message, "close summary"):
		return "Debes completar el resumen de cierre."
	case strings.Contains(message, "viewer"):
		return "Ese usuario no puede recibir órdenes de trabajo."
	case strings.Contains(message, "not found"):
		return "Uno de los registros relacionados ya no existe."
	case strings.Contains(message, "invalid"):
		return "Uno de los valores cargados no es válido."
	default:
		return "No se pudo guardar la operación de ejecución. Revisa los datos e intenta de nuevo."
	}
}

func executionWorkOrderStatusOptions() []string {
	return []string{
		models.WorkOrderStatusAssigned,
		models.WorkOrderStatusInProgress,
		models.WorkOrderStatusPaused,
		models.WorkOrderStatusBlocked,
		models.WorkOrderStatusDone,
		models.WorkOrderStatusCancelled,
	}
}

func executionIncidentStatusOptions() []string {
	return []string{
		models.IncidentStatusOpen,
		models.IncidentStatusInvestigating,
		models.IncidentStatusResolved,
		models.IncidentStatusClosed,
	}
}

func (s *Server) executionStatusOptions() []string {
	return executionWorkOrderStatusOptions()
}

func (s *Server) incidentStatusOptions() []string {
	return executionIncidentStatusOptions()
}

func (s *Server) executionHeading(workOrder models.WorkOrderDetail) string {
	if workOrder.AssetName == "" {
		return workOrder.WorkOrderCode
	}
	return fmt.Sprintf("%s · %s", workOrder.WorkOrderCode, workOrder.AssetName)
}
