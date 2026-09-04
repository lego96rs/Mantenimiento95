package models

import (
	"context"
	"path/filepath"
	"testing"

	"mantenimiento/internal/db"
)

func openModelTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "models.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return database
}

func TestAssetCatalogsAndRelations(t *testing.T) {
	ctx := context.Background()
	database := openModelTestDB(t)

	areaID, err := CreateArea(ctx, database, "Operaciones", "Área principal")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	categoryID, err := CreateAssetCategory(ctx, database, "Transportador", "Equipos de transporte")
	if err != nil {
		t.Fatalf("CreateAssetCategory: %v", err)
	}
	docManualID, err := CreateTechnicalDocument(ctx, database, "System Maintenance Plan", "docs/System Maintenance Plan.pdf", DocumentTypePlan, "Tabla 2", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument plan: %v", err)
	}
	docDataID, err := CreateTechnicalDocument(ctx, database, "Roller Conveyor Manual", "docs/Roller Conveyor System STREAMLINE (Maintenance and Repair).pdf", DocumentTypeManual, "Cap 4", "")
	if err != nil {
		t.Fatalf("CreateTechnicalDocument manual: %v", err)
	}

	assetID, err := CreateAsset(ctx, database, AssetInput{
		Code:             "EQ-001",
		Name:             "Conveyor principal",
		Family:           "Transportador",
		AreaID:           areaID,
		CategoryID:       categoryID,
		Location:         "Pasillo 3",
		Manufacturer:     "KNAPP",
		Model:            "SRC",
		SerialNumber:     "SN-001",
		OperationalState: AssetStateActive,
		Criticality:      AssetCriticalityHigh,
		Active:           true,
		DocumentIDs:      []int64{docManualID, docDataID},
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}

	assets, err := ListAssets(ctx, database, AssetFilters{Query: "EQ-001", ActiveFilter: "active"})
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("len(assets) = %d, want 1", len(assets))
	}
	if assets[0].AreaName != "Operaciones" || assets[0].CategoryName != "Transportador" {
		t.Fatalf("asset classification = %#v", assets[0])
	}
	if len(assets[0].DocumentTitles) != 2 {
		t.Fatalf("document titles = %v, want 2 docs", assets[0].DocumentTitles)
	}

	stored, err := AssetByID(ctx, database, assetID)
	if err != nil {
		t.Fatalf("AssetByID: %v", err)
	}
	if stored == nil || stored.Code != "EQ-001" || len(stored.DocumentIDs) != 2 {
		t.Fatalf("stored asset = %#v", stored)
	}

	if err := UpdateAsset(ctx, database, assetID, AssetInput{
		Code:             "EQ-001",
		Name:             "Conveyor principal actualizado",
		Family:           "Transportador",
		AreaID:           areaID,
		CategoryID:       categoryID,
		Location:         "Pasillo 4",
		Manufacturer:     "KNAPP",
		Model:            "SRC",
		SerialNumber:     "SN-001",
		OperationalState: AssetStateMaintenance,
		Criticality:      AssetCriticalityCritical,
		Active:           false,
		DocumentIDs:      []int64{docManualID},
	}); err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}

	activeAssets, err := ListAssets(ctx, database, AssetFilters{ActiveFilter: "active"})
	if err != nil {
		t.Fatalf("ListAssets active: %v", err)
	}
	if len(activeAssets) != 0 {
		t.Fatalf("active assets after deactivation = %d, want 0", len(activeAssets))
	}

	allAssets, err := ListAssets(ctx, database, AssetFilters{ActiveFilter: "all", State: AssetStateMaintenance})
	if err != nil {
		t.Fatalf("ListAssets all: %v", err)
	}
	if len(allAssets) != 1 {
		t.Fatalf("len(allAssets) = %d, want 1", len(allAssets))
	}
	if allAssets[0].Criticality != AssetCriticalityCritical || len(allAssets[0].DocumentTitles) != 1 {
		t.Fatalf("updated asset = %#v", allAssets[0])
	}
}
