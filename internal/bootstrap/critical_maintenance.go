package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mantenimiento/internal/db"
	"mantenimiento/internal/models"
)

type CriticalMaintenanceSummary struct {
	Documents    int
	Assets       int
	Templates    int
	Schedules    int
	WorkOrders   int
	Assignments  int
	Checklists   int
}

type maintenanceSeed struct {
	DocumentTitle               string
	DocumentPath                string
	DocumentType                string
	SourceRef                   string
	DocumentNotes               string
	AreaName                    string
	AreaDescription             string
	CategoryName                string
	CategoryDescription         string
	AssetCode                   string
	AssetName                   string
	AssetFamily                 string
	AssetLocation               string
	AssetState                  string
	AssetCriticality            string
	TemplateName                string
	FrequencyCode               string
	MaintenanceType             string
	ProcedureSummary            string
	ValidationCriteria          string
	Priority                    string
	EstimatedMinutes            int
	RequiresChecklist           bool
	RequiresSupervisor          bool
	RequiresQualifiedPersonnel  bool
	RequiredImplements          []string
	ChecklistItems              []string
	AutoSchedule                bool
	ScheduleOffsetDays          int
}

func SeedCriticalMaintenance(ctx context.Context, database *db.DB, baseDate time.Time) (CriticalMaintenanceSummary, error) {
	if baseDate.IsZero() {
		baseDate = time.Now().UTC()
	}

	manager, assignees, err := loadSeedUsers(ctx, database)
	if err != nil {
		return CriticalMaintenanceSummary{}, err
	}

	summary := CriticalMaintenanceSummary{}
	assetsByCode := make(map[string]int64)
	templatesByKey := make(map[string]int64)
	assigneeIndex := 0

	for _, seed := range criticalMaintenanceSeeds() {
		docID, created, err := ensureTechnicalDocument(ctx, database, seed)
		if err != nil {
			return summary, err
		}
		if created {
			summary.Documents++
		}

		areaID, err := ensureArea(ctx, database, seed.AreaName, seed.AreaDescription)
		if err != nil {
			return summary, err
		}
		categoryID, err := ensureAssetCategory(ctx, database, seed.CategoryName, seed.CategoryDescription)
		if err != nil {
			return summary, err
		}

		assetID, ok := assetsByCode[seed.AssetCode]
		if !ok {
			var assetCreated bool
			assetID, assetCreated, err = ensureAsset(ctx, database, seed, areaID, categoryID, docID)
			if err != nil {
				return summary, err
			}
			if assetCreated {
				summary.Assets++
			}
			assetsByCode[seed.AssetCode] = assetID
		}

		templateKey := seed.AssetCode + "::" + seed.TemplateName
		templateID, ok := templatesByKey[templateKey]
		if !ok {
			var templateCreated bool
			templateID, templateCreated, err = ensureTemplate(ctx, database, seed, assetID, categoryID, docID)
			if err != nil {
				return summary, err
			}
			if templateCreated {
				summary.Templates++
			}
			templatesByKey[templateKey] = templateID
		}

		if !seed.AutoSchedule {
			continue
		}

		scheduledFor := baseDate.AddDate(0, 0, seed.ScheduleOffsetDays).Format("2006-01-02")
		scheduleID, scheduleCreated, err := ensurePublishedSchedule(ctx, database, seed, templateID, assetID, docID, manager.ID, scheduledFor)
		if err != nil {
			return summary, err
		}
		if scheduleCreated {
			summary.Schedules++
		}

		workOrderID, workOrderCreated, err := ensureWorkOrder(ctx, database, scheduleID, manager.ID)
		if err != nil {
			return summary, err
		}
		if workOrderCreated {
			summary.WorkOrders++
		}

		assignee := assignees[assigneeIndex%len(assignees)]
		assigneeIndex++
		if err := models.SetWorkOrderAssignment(ctx, database, workOrderID, assignee.ID); err != nil {
			return summary, fmt.Errorf("assign work order %d to %s: %w", workOrderID, assignee.Username, err)
		}
		summary.Assignments++

		if len(seed.ChecklistItems) > 0 {
			items := make([]models.WorkOrderChecklistItemInput, 0, len(seed.ChecklistItems))
			for i, item := range seed.ChecklistItems {
				items = append(items, models.WorkOrderChecklistItemInput{
					ItemText:  item,
					SortOrder: i + 1,
				})
			}
			if err := models.SetWorkOrderChecklist(ctx, database, workOrderID, items); err != nil {
				return summary, fmt.Errorf("set work order checklist %d: %w", workOrderID, err)
			}
			summary.Checklists++
		}
	}

	return summary, nil
}

func criticalMaintenanceSeeds() []maintenanceSeed {
	return []maintenanceSeed{
		{
			DocumentTitle:              "System Maintenance Plan",
			DocumentPath:               "docs/System Maintenance Plan.pdf",
			DocumentType:               models.DocumentTypePlan,
			SourceRef:                  "Tabla 2",
			DocumentNotes:              "Plan maestro de mantenimiento usado como fuente primaria de frecuencias.",
			AreaName:                   "Operaciones",
			AreaDescription:            "Equipos principales de operación y transporte interno.",
			CategoryName:               "Estación de trabajo",
			CategoryDescription:        "Puestos operativos y estaciones Pick-it-Easy.",
			AssetCode:                  "AST-WS-001",
			AssetName:                  "Pick-it-Easy Evo 01",
			AssetFamily:                "Workstation",
			AssetLocation:              "Zona picking",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Limpieza semanal de scanner",
			FrequencyCode:              models.FrequencyWeekly,
			MaintenanceType:            models.MaintenanceTypeCleaning,
			ProcedureSummary:           "Limpiar scanner y ventana óptica siguiendo la instrucción de mantenimiento del puesto.",
			ValidationCriteria:         "Scanner limpio, sin residuos visibles y con lectura estable. Implementos: paño sin pelusa, limpiador óptico, aire seco, guantes.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           30,
			RequiresChecklist:          true,
			RequiredImplements:         []string{"Paño sin pelusa", "Limpiador óptico", "Aire seco", "Guantes"},
			ChecklistItems:             []string{"Aplicar bloqueo seguro si corresponde", "Limpiar superficie del scanner", "Verificar lectura y ausencia de alarmas", "Registrar observaciones"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         1,
		},
		{
			DocumentTitle:              "Open Shuttle 100b (Maintenance and Repair)",
			DocumentPath:               "docs/Open Shuttle 100b (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Sección limpieza y sensores",
			DocumentNotes:              "Manual crítico para limpieza y ajustes condicionales del shuttle.",
			AreaName:                   "Automatización",
			AreaDescription:            "Equipos shuttle y automatización intralogística.",
			CategoryName:               "Shuttle",
			CategoryDescription:        "Vehículos y carros automatizados.",
			AssetCode:                  "AST-SHUT-001",
			AssetName:                  "Open Shuttle 100b 01",
			AssetFamily:                "Shuttle",
			AssetLocation:              "Circuito Open Shuttle",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Limpieza mensual de sensores ópticos y reflectores",
			FrequencyCode:              models.FrequencyMonthly,
			MaintenanceType:            models.MaintenanceTypeInspection,
			ProcedureSummary:           "Limpiar sensores ópticos y reflectores del shuttle para asegurar detección estable.",
			ValidationCriteria:         "Sensores y reflectores limpios, alineados y sin alarmas. Implementos: paño microfibra, limpiador óptico, linterna, guantes.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           45,
			RequiresChecklist:          true,
			RequiredImplements:         []string{"Paño microfibra", "Limpiador óptico", "Linterna", "Guantes"},
			ChecklistItems:             []string{"Asegurar acceso seguro al shuttle", "Limpiar sensores ópticos", "Limpiar reflectores", "Confirmar alineación y lectura"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         2,
		},
		{
			DocumentTitle:              "Roller Conveyor System STREAMLINE (Maintenance and Repair)",
			DocumentPath:               "docs/Roller Conveyor System STREAMLINE (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Capítulo limpieza semestral",
			DocumentNotes:              "Manual principal de mantenimiento periódico del transportador de rodillos.",
			AreaName:                   "Operaciones",
			AreaDescription:            "Equipos principales de operación y transporte interno.",
			CategoryName:               "Transportador",
			CategoryDescription:        "Transportadores, curvas y transferencias.",
			AssetCode:                  "AST-CONV-001",
			AssetName:                  "Roller Conveyor principal",
			AssetFamily:                "Conveyor",
			AssetLocation:              "Línea transporte 1",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityCritical,
			TemplateName:               "Limpieza semestral de correas, rodillos y rieles",
			FrequencyCode:              models.FrequencySemiAnnual,
			MaintenanceType:            models.MaintenanceTypeCleaning,
			ProcedureSummary:           "Limpiar correas, rodillos y rieles del transportador siguiendo la rutina semestral.",
			ValidationCriteria:         "Sistema libre de polvo y residuos, giro sin rozamientos anómalos. Implementos: aspiradora industrial, cepillo, paño, kit LOTO.",
			Priority:                   models.AssetCriticalityCritical,
			EstimatedMinutes:           120,
			RequiresChecklist:          true,
			RequiresSupervisor:         true,
			RequiredImplements:         []string{"Aspiradora industrial", "Cepillo", "Paño", "Kit LOTO"},
			ChecklistItems:             []string{"Aplicar LOTO y detener equipo", "Limpiar rodillos", "Limpiar rieles y correas", "Verificar libre giro y registrar condición"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         3,
		},
		{
			DocumentTitle:              "Local Control Cabinet 2.0 Streamline (Maintenance and Repair)",
			DocumentPath:               "docs/Local Control Cabinet 2.0 Streamline (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Pruebas de seguridad",
			DocumentNotes:              "Manual de gabinete local y pruebas periódicas de seguridad.",
			AreaName:                   "Electricidad",
			AreaDescription:            "Gabinetes eléctricos, HMI y seguridad.",
			CategoryName:               "Gabinete eléctrico",
			CategoryDescription:        "Gabinetes de control y tableros locales.",
			AssetCode:                  "AST-CAB-001",
			AssetName:                  "Local Control Cabinet 2.0",
			AssetFamily:                "Electrical Cabinet",
			AssetLocation:              "Zona técnica",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityCritical,
			TemplateName:               "Verificación anual de botón de paro de emergencia",
			FrequencyCode:              models.FrequencyYearly,
			MaintenanceType:            models.MaintenanceTypeSafety,
			ProcedureSummary:           "Verificar correcto funcionamiento del botón de paro de emergencia del gabinete local.",
			ValidationCriteria:         "Paro probado y restablecido sin fallas. Implementos: checklist de seguridad, multímetro, kit LOTO, EPP dieléctrico.",
			Priority:                   models.AssetCriticalityCritical,
			EstimatedMinutes:           60,
			RequiresChecklist:          true,
			RequiresSupervisor:         true,
			RequiresQualifiedPersonnel: true,
			RequiredImplements:         []string{"Checklist de seguridad", "Multímetro", "Kit LOTO", "EPP dieléctrico"},
			ChecklistItems:             []string{"Aislar equipo y validar condición segura", "Probar botón de paro de emergencia", "Confirmar restablecimiento del circuito", "Registrar resultado de seguridad"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         4,
		},
		{
			DocumentTitle:              "Pneumatic Control Panel for Warehouse Areas (Maintenance and Repair)",
			DocumentPath:               "docs/Pneumatic Control Panel for Warehouse Areas (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Mantenimiento mensual",
			DocumentNotes:              "Rutinas periódicas del panel neumático y sus condiciones operativas.",
			AreaName:                   "Servicios",
			AreaDescription:            "Servicios auxiliares y sistemas neumáticos.",
			CategoryName:               "Panel neumático",
			CategoryDescription:        "Paneles de control y distribución neumática.",
			AssetCode:                  "AST-PNEU-001",
			AssetName:                  "Panel neumático área warehouse",
			AssetFamily:                "Pneumatic Panel",
			AssetLocation:              "Sala neumática",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Drenaje mensual de separador de condensados",
			FrequencyCode:              models.FrequencyMonthly,
			MaintenanceType:            models.MaintenanceTypeInspection,
			ProcedureSummary:           "Drenar el separador de condensados y validar que no queden acumulaciones de agua.",
			ValidationCriteria:         "Separador drenado y sin fugas posteriores. Implementos: recipiente de drenaje, guantes, gafas, paño absorbente.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           20,
			RequiresChecklist:          true,
			RequiredImplements:         []string{"Recipiente de drenaje", "Guantes", "Gafas", "Paño absorbente"},
			ChecklistItems:             []string{"Asegurar condición de trabajo segura", "Drenar condensado", "Verificar fugas o humedad residual", "Registrar volumen y observaciones"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         5,
		},
		{
			DocumentTitle:              "Pneumatic Control Panel for Warehouse Areas (Maintenance and Repair)",
			DocumentPath:               "docs/Pneumatic Control Panel for Warehouse Areas (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Control de presión de operación",
			DocumentNotes:              "Rutinas periódicas del panel neumático y sus condiciones operativas.",
			AreaName:                   "Servicios",
			AreaDescription:            "Servicios auxiliares y sistemas neumáticos.",
			CategoryName:               "Panel neumático",
			CategoryDescription:        "Paneles de control y distribución neumática.",
			AssetCode:                  "AST-PNEU-001",
			AssetName:                  "Panel neumático área warehouse",
			AssetFamily:                "Pneumatic Panel",
			AssetLocation:              "Sala neumática",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Verificación mensual de presión de operación a 6 bar",
			FrequencyCode:              models.FrequencyMonthly,
			MaintenanceType:            models.MaintenanceTypeInspection,
			ProcedureSummary:           "Verificar y ajustar la presión de operación del panel neumático a 6 bar según la documentación.",
			ValidationCriteria:         "Manómetro en 6 bar y sin fluctuaciones anómalas. Implementos: manómetro, llave de regulación, EPP, checklist neumático.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           25,
			RequiresChecklist:          true,
			RequiredImplements:         []string{"Manómetro", "Llave de regulación", "EPP", "Checklist neumático"},
			ChecklistItems:             []string{"Leer presión actual", "Ajustar regulación si aplica", "Confirmar estabilidad en 6 bar", "Registrar medición final"},
			AutoSchedule:               true,
			ScheduleOffsetDays:         6,
		},
		{
			DocumentTitle:              "Open Shuttle 100b (Maintenance and Repair)",
			DocumentPath:               "docs/Open Shuttle 100b (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Ajuste condicional de alineación",
			DocumentNotes:              "Manual crítico para limpieza y ajustes condicionales del shuttle.",
			AreaName:                   "Automatización",
			AreaDescription:            "Equipos shuttle y automatización intralogística.",
			CategoryName:               "Shuttle",
			CategoryDescription:        "Vehículos y carros automatizados.",
			AssetCode:                  "AST-SHUT-001",
			AssetName:                  "Open Shuttle 100b 01",
			AssetFamily:                "Shuttle",
			AssetLocation:              "Circuito Open Shuttle",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Ajuste condicional de alineación de laser scanner",
			FrequencyCode:              models.FrequencyConditional,
			MaintenanceType:            models.MaintenanceTypeCorrective,
			ProcedureSummary:           "Ajustar la alineación del laser scanner cuando se detecte orientación incorrecta.",
			ValidationCriteria:         "Orientación corregida y lectura validada. Implementos: juego Allen, plantilla de alineación, nivel, EPP.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           45,
			RequiresChecklist:          true,
			RequiresQualifiedPersonnel: true,
			RequiredImplements:         []string{"Juego Allen", "Plantilla de alineación", "Nivel", "EPP"},
			ChecklistItems:             []string{"Validar condición que dispara la intervención", "Ajustar alineación del scanner", "Probar lectura y posición", "Documentar causa y corrección"},
			AutoSchedule:               false,
		},
		{
			DocumentTitle:              "Belt Conveyor Streamline (Maintenance and Repair)",
			DocumentPath:               "docs/Belt Conveyor Streamline (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Ajustes por soltura detectada",
			DocumentNotes:              "Manual crítico del transportador de banda para ajustes y reaprietes.",
			AreaName:                   "Operaciones",
			AreaDescription:            "Equipos principales de operación y transporte interno.",
			CategoryName:               "Transportador",
			CategoryDescription:        "Transportadores, curvas y transferencias.",
			AssetCode:                  "AST-BELT-001",
			AssetName:                  "Belt Conveyor 01",
			AssetFamily:                "Belt Conveyor",
			AssetLocation:              "Línea transporte 2",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityHigh,
			TemplateName:               "Reapriete condicional de fijaciones de ruedas y motores",
			FrequencyCode:              models.FrequencyConditional,
			MaintenanceType:            models.MaintenanceTypeCorrective,
			ProcedureSummary:           "Reapretar fijaciones de ruedas, motores o componentes cuando se detecte soltura.",
			ValidationCriteria:         "Fijaciones reapretadas al par definido y sin vibración anómala. Implementos: torquímetro, juego de llaves, marcador de torque, EPP.",
			Priority:                   models.AssetCriticalityHigh,
			EstimatedMinutes:           60,
			RequiresChecklist:          true,
			RequiredImplements:         []string{"Torquímetro", "Juego de llaves", "Marcador de torque", "EPP"},
			ChecklistItems:             []string{"Identificar fijación suelta", "Reapretar al par requerido", "Marcar punto ajustado", "Verificar ausencia de juego o vibración"},
			AutoSchedule:               false,
		},
		{
			DocumentTitle:              "OSR Shuttle™ Evo 1D Elementary (Maintenance and Repair)",
			DocumentPath:               "docs/OSR Shuttle™ Evo 1D Elementary (Maintenance and Repair).pdf",
			DocumentType:               models.DocumentTypeManual,
			SourceRef:                  "Reemplazo por condición",
			DocumentNotes:              "Manual del OSR Shuttle Evo para componentes críticos y correctivos por condición.",
			AreaName:                   "Automatización",
			AreaDescription:            "Equipos shuttle y automatización intralogística.",
			CategoryName:               "Shuttle",
			CategoryDescription:        "Vehículos y carros automatizados.",
			AssetCode:                  "AST-OSR-001",
			AssetName:                  "OSR Shuttle Evo 1D 01",
			AssetFamily:                "OSR Shuttle",
			AssetLocation:              "Módulo OSR",
			AssetState:                 models.AssetStateActive,
			AssetCriticality:           models.AssetCriticalityCritical,
			TemplateName:               "Reemplazo condicional de sensores, reflectores o componentes defectuosos",
			FrequencyCode:              models.FrequencyConditional,
			MaintenanceType:            models.MaintenanceTypeCorrective,
			ProcedureSummary:           "Reemplazar sensores, reflectores, correas, filtros o componentes defectuosos cuando la condición lo indique.",
			ValidationCriteria:         "Componente reemplazado, ajuste confirmado y equipo habilitado. Implementos: repuesto homologado, herramientas de montaje, checklist de prueba, EPP.",
			Priority:                   models.AssetCriticalityCritical,
			EstimatedMinutes:           90,
			RequiresChecklist:          true,
			RequiresSupervisor:         true,
			RequiresQualifiedPersonnel: true,
			RequiredImplements:         []string{"Repuesto homologado", "Herramientas de montaje", "Checklist de prueba", "EPP"},
			ChecklistItems:             []string{"Identificar componente defectuoso", "Bloquear equipo y retirar componente", "Instalar repuesto homologado", "Probar funcionamiento y documentar cambio"},
			AutoSchedule:               false,
		},
	}
}

func loadSeedUsers(ctx context.Context, database *db.DB) (models.User, []models.User, error) {
	managers, err := models.ListActiveUsers(ctx, database, models.RolePlanner, models.RoleSupervisor, models.RoleAdmin)
	if err != nil {
		return models.User{}, nil, err
	}
	if len(managers) == 0 {
		return models.User{}, nil, fmt.Errorf("bootstrap: no hay usuarios activos con rol planner, supervisor o admin para publicar las tareas")
	}

	technicians, err := models.ListActiveUsers(ctx, database, models.RoleTechnician)
	if err != nil {
		return models.User{}, nil, err
	}
	if len(technicians) == 0 {
		technicians = append(technicians, managers...)
	}
	if len(technicians) == 0 {
		return models.User{}, nil, fmt.Errorf("bootstrap: no hay operadores activos para asignar tareas")
	}

	return managers[0], technicians, nil
}

func ensureArea(ctx context.Context, database *db.DB, name, description string) (int64, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM areas WHERE name = ?`,
		name,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("bootstrap: lookup area %q: %w", name, err)
	}
	return models.CreateArea(ctx, database, name, description)
}

func ensureAssetCategory(ctx context.Context, database *db.DB, name, description string) (int64, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM asset_categories WHERE name = ?`,
		name,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("bootstrap: lookup asset category %q: %w", name, err)
	}
	return models.CreateAssetCategory(ctx, database, name, description)
}

func ensureTechnicalDocument(ctx context.Context, database *db.DB, seed maintenanceSeed) (int64, bool, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM technical_documents WHERE file_path = ?`,
		seed.DocumentPath,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("bootstrap: lookup technical document %q: %w", seed.DocumentPath, err)
	}
	id, err = models.CreateTechnicalDocument(ctx, database, seed.DocumentTitle, seed.DocumentPath, seed.DocumentType, seed.SourceRef, seed.DocumentNotes)
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func ensureAsset(ctx context.Context, database *db.DB, seed maintenanceSeed, areaID, categoryID, documentID int64) (int64, bool, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM assets WHERE code = ?`,
		seed.AssetCode,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("bootstrap: lookup asset %q: %w", seed.AssetCode, err)
	}
	id, err = models.CreateAsset(ctx, database, models.AssetInput{
		Code:             seed.AssetCode,
		Name:             seed.AssetName,
		Family:           seed.AssetFamily,
		AreaID:           areaID,
		CategoryID:       categoryID,
		Location:         seed.AssetLocation,
		OperationalState: seed.AssetState,
		Criticality:      seed.AssetCriticality,
		Notes:            "Activo base cargado desde documentación crítica.",
		Active:           true,
		DocumentIDs:      []int64{documentID},
	})
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func ensureTemplate(ctx context.Context, database *db.DB, seed maintenanceSeed, assetID, categoryID, documentID int64) (int64, bool, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM maintenance_templates WHERE name = ? AND asset_id = ?`,
		seed.TemplateName,
		assetID,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("bootstrap: lookup template %q: %w", seed.TemplateName, err)
	}

	validation := seed.ValidationCriteria
	if implements := strings.Join(seed.RequiredImplements, ", "); implements != "" && !strings.Contains(validation, "Implementos:") {
		validation = strings.TrimSpace(validation + " Implementos: " + implements + ".")
	}

	id, err = models.CreateMaintenanceTemplate(ctx, database, models.MaintenanceTemplateInput{
		Name:                       seed.TemplateName,
		AssetID:                    assetID,
		AssetCategoryID:            categoryID,
		SourceDocumentID:           documentID,
		SourceRef:                  seed.SourceRef,
		FrequencyCode:              seed.FrequencyCode,
		MaintenanceType:            seed.MaintenanceType,
		ProcedureSummary:           seed.ProcedureSummary,
		ValidationCriteria:         validation,
		RequiresChecklist:          seed.RequiresChecklist,
		RequiresSupervisor:         seed.RequiresSupervisor,
		RequiresQualifiedPersonnel: seed.RequiresQualifiedPersonnel,
		Priority:                   seed.Priority,
		EstimatedMinutes:           seed.EstimatedMinutes,
		Active:                     true,
	})
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func ensurePublishedSchedule(ctx context.Context, database *db.DB, seed maintenanceSeed, templateID, assetID, documentID, publisherID int64, scheduledFor string) (int64, bool, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id
		 FROM scheduled_tasks
		 WHERE template_id = ?
		   AND scheduled_for = ?
		   AND title = ?`,
		templateID,
		scheduledFor,
		seed.TemplateName,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("bootstrap: lookup schedule %q: %w", seed.TemplateName, err)
	}

	id, err = models.CreateScheduledTask(ctx, database, models.ScheduledTaskInput{
		TemplateID:       templateID,
		AssetID:          assetID,
		SourceDocumentID: documentID,
		Title:            seed.TemplateName,
		FrequencyCode:    seed.FrequencyCode,
		MaintenanceType:  seed.MaintenanceType,
		Status:           models.PlannedStatusPublished,
		ScheduledFor:     scheduledFor,
		PublicationNotes: buildPublicationNotes(seed),
		CreatedBy:        publisherID,
		PublishedBy:      publisherID,
		PublishedAt:      time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func ensureWorkOrder(ctx context.Context, database *db.DB, scheduledTaskID, publishedBy int64) (int64, bool, error) {
	var id int64
	err := database.Read.QueryRowContext(ctx,
		`SELECT id FROM work_orders WHERE scheduled_task_id = ?`,
		scheduledTaskID,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, fmt.Errorf("bootstrap: lookup work order for schedule %d: %w", scheduledTaskID, err)
	}

	id, err = models.CreateWorkOrderFromScheduledTask(ctx, database, scheduledTaskID, publishedBy)
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func buildPublicationNotes(seed maintenanceSeed) string {
	parts := []string{}
	if len(seed.RequiredImplements) > 0 {
		parts = append(parts, "Implementos: "+strings.Join(seed.RequiredImplements, ", "))
	}
	if seed.RequiresQualifiedPersonnel {
		parts = append(parts, "Requiere personal calificado")
	}
	if seed.RequiresSupervisor {
		parts = append(parts, "Validación de supervisor")
	}
	return strings.Join(parts, " | ")
}
