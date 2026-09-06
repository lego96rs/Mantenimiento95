package bootstrap

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mantenimiento/internal/db"
	"mantenimiento/internal/models"
)

func openBootstrapTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return database
}

func createSeedUser(t *testing.T, database *db.DB, username, role string) {
	t.Helper()
	if _, err := models.CreateUser(context.Background(), database, username, username, role, "hash", false); err != nil {
		t.Fatalf("CreateUser(%s): %v", username, err)
	}
}

func TestSeedCriticalMaintenanceCreatesTemplatesSchedulesAndAssignments(t *testing.T) {
	ctx := context.Background()
	database := openBootstrapTestDB(t)

	createSeedUser(t, database, "planner1", models.RolePlanner)
	createSeedUser(t, database, "tech1", models.RoleTechnician)
	createSeedUser(t, database, "tech2", models.RoleTechnician)

	summary, err := SeedCriticalMaintenance(ctx, database, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("SeedCriticalMaintenance: %v", err)
	}
	if summary.Documents == 0 || summary.Templates == 0 || summary.WorkOrders == 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}

	templates, err := models.ListMaintenanceTemplates(ctx, database, models.TemplateFilters{ActiveFilter: "active"})
	if err != nil {
		t.Fatalf("ListMaintenanceTemplates: %v", err)
	}
	if len(templates) < 8 {
		t.Fatalf("len(templates) = %d, want at least 8", len(templates))
	}

	tasks, err := models.ListScheduledTasks(ctx, database, models.ScheduleFilters{Status: models.PlannedStatusPublished})
	if err != nil {
		t.Fatalf("ListScheduledTasks: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected published schedules to be created")
	}

	workOrders, err := models.ListWorkOrders(ctx, database, models.WorkOrderFilters{})
	if err != nil {
		t.Fatalf("ListWorkOrders: %v", err)
	}
	if len(workOrders) == 0 {
		t.Fatal("expected work orders to be created")
	}
	for _, item := range workOrders {
		if item.AssignedTo == 0 || item.AssignedToName == "" {
			t.Fatalf("work order not assigned: %#v", item)
		}
		if !strings.Contains(item.PublicationNotes, "Implementos:") {
			t.Fatalf("publication notes missing implements: %#v", item)
		}
	}

	detail, err := models.WorkOrderByID(ctx, database, workOrders[0].ID)
	if err != nil {
		t.Fatalf("WorkOrderByID: %v", err)
	}
	if detail == nil || len(detail.Checklist) == 0 {
		t.Fatalf("expected checklist in work order detail, got %#v", detail)
	}
}

func TestSeedCriticalMaintenanceIsIdempotent(t *testing.T) {
	ctx := context.Background()
	database := openBootstrapTestDB(t)

	createSeedUser(t, database, "planner1", models.RolePlanner)
	createSeedUser(t, database, "tech1", models.RoleTechnician)

	baseDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if _, err := SeedCriticalMaintenance(ctx, database, baseDate); err != nil {
		t.Fatalf("first SeedCriticalMaintenance: %v", err)
	}
	if _, err := SeedCriticalMaintenance(ctx, database, baseDate); err != nil {
		t.Fatalf("second SeedCriticalMaintenance: %v", err)
	}

	var templateCount int
	if err := database.Read.QueryRowContext(ctx, `SELECT COUNT(*) FROM maintenance_templates`).Scan(&templateCount); err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if templateCount != len(criticalMaintenanceSeeds()) {
		t.Fatalf("template count = %d, want %d", templateCount, len(criticalMaintenanceSeeds()))
	}

	var scheduleCount int
	if err := database.Read.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduled_tasks`).Scan(&scheduleCount); err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	expectedSchedules := 0
	for _, seed := range criticalMaintenanceSeeds() {
		if seed.AutoSchedule {
			expectedSchedules++
		}
	}
	if scheduleCount != expectedSchedules {
		t.Fatalf("schedule count = %d, want %d", scheduleCount, expectedSchedules)
	}

	var workOrderCount int
	if err := database.Read.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_orders`).Scan(&workOrderCount); err != nil {
		t.Fatalf("count work orders: %v", err)
	}
	if workOrderCount != expectedSchedules {
		t.Fatalf("work order count = %d, want %d", workOrderCount, expectedSchedules)
	}
}

func TestSeedCriticalMaintenanceRequiresAssignableUsers(t *testing.T) {
	ctx := context.Background()
	database := openBootstrapTestDB(t)

	createSeedUser(t, database, "viewer1", models.RoleViewer)

	if _, err := SeedCriticalMaintenance(ctx, database, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("expected SeedCriticalMaintenance to fail without managers/operators")
	}
}
