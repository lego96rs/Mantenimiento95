package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mantenimiento/internal/db"
)

const (
	DocumentTypeManual      = "manual"
	DocumentTypePlan        = "plan"
	DocumentTypeDatasheet   = "datasheet"
	DocumentTypeInstruction = "instruction"
	DocumentTypeOther       = "other"

	AssetStateActive      = "active"
	AssetStateMaintenance = "maintenance"
	AssetStateFault       = "fault"
	AssetStateInactive    = "inactive"
	AssetStateRetired     = "retired"

	AssetCriticalityLow      = "low"
	AssetCriticalityMedium   = "medium"
	AssetCriticalityHigh     = "high"
	AssetCriticalityCritical = "critical"
)

type Area struct {
	ID          int64
	Name        string
	Description string
	Active      bool
}

type AssetCategory struct {
	ID          int64
	Name        string
	Description string
	Active      bool
}

type TechnicalDocument struct {
	ID           int64
	Title        string
	FilePath     string
	DocumentType string
	SourceRef    string
	Notes        string
	Active       bool
}

type Asset struct {
	ID               int64
	Code             string
	Name             string
	Family           string
	AreaID           int64
	CategoryID       int64
	AreaName         string
	CategoryName     string
	Subarea          string
	Location         string
	Manufacturer     string
	Model            string
	SerialNumber     string
	OperationalState string
	Criticality      string
	Notes            string
	Active           bool
	DocumentTitles   []string
}

type AssetFilters struct {
	Query        string
	AreaID       int64
	CategoryID   int64
	State        string
	Criticality  string
	ActiveFilter string
}

type AssetInput struct {
	Code             string
	Name             string
	Family           string
	AreaID           int64
	CategoryID       int64
	Subarea          string
	Location         string
	Manufacturer     string
	Model            string
	SerialNumber     string
	OperationalState string
	Criticality      string
	Notes            string
	Active           bool
	DocumentIDs      []int64
}

func ListAreas(ctx context.Context, d *db.DB) ([]Area, error) {
	rows, err := d.Read.QueryContext(ctx,
		`SELECT id, name, description, active
		 FROM areas
		 WHERE active = 1
		 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("models: list areas: %w", err)
	}
	defer rows.Close()

	var items []Area
	for rows.Next() {
		var item Area
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Active); err != nil {
			return nil, fmt.Errorf("models: scan area: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func CreateArea(ctx context.Context, d *db.DB, name, description string) (int64, error) {
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO areas (name, description) VALUES (?, ?)`,
		strings.TrimSpace(name), strings.TrimSpace(description),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create area: %w", err)
	}
	return res.LastInsertId()
}

func ListAssetCategories(ctx context.Context, d *db.DB) ([]AssetCategory, error) {
	rows, err := d.Read.QueryContext(ctx,
		`SELECT id, name, description, active
		 FROM asset_categories
		 WHERE active = 1
		 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("models: list asset categories: %w", err)
	}
	defer rows.Close()

	var items []AssetCategory
	for rows.Next() {
		var item AssetCategory
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Active); err != nil {
			return nil, fmt.Errorf("models: scan asset category: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func CreateAssetCategory(ctx context.Context, d *db.DB, name, description string) (int64, error) {
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO asset_categories (name, description) VALUES (?, ?)`,
		strings.TrimSpace(name), strings.TrimSpace(description),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create asset category: %w", err)
	}
	return res.LastInsertId()
}

func ListTechnicalDocuments(ctx context.Context, d *db.DB) ([]TechnicalDocument, error) {
	rows, err := d.Read.QueryContext(ctx,
		`SELECT id, title, file_path, document_type, source_ref, notes, active
		 FROM technical_documents
		 WHERE active = 1
		 ORDER BY title`,
	)
	if err != nil {
		return nil, fmt.Errorf("models: list technical documents: %w", err)
	}
	defer rows.Close()

	var items []TechnicalDocument
	for rows.Next() {
		var item TechnicalDocument
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.FilePath,
			&item.DocumentType,
			&item.SourceRef,
			&item.Notes,
			&item.Active,
		); err != nil {
			return nil, fmt.Errorf("models: scan technical document: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func CreateTechnicalDocument(ctx context.Context, d *db.DB, title, filePath, documentType, sourceRef, notes string) (int64, error) {
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO technical_documents (title, file_path, document_type, source_ref, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		strings.TrimSpace(title),
		strings.TrimSpace(filePath),
		strings.TrimSpace(documentType),
		strings.TrimSpace(sourceRef),
		strings.TrimSpace(notes),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create technical document: %w", err)
	}
	return res.LastInsertId()
}

func ListAssets(ctx context.Context, d *db.DB, filters AssetFilters) ([]Asset, error) {
	var (
		args []any
		sqlb strings.Builder
	)

	sqlb.WriteString(`
SELECT
    a.id,
    a.code,
    a.name,
    a.family,
    COALESCE(a.area_id, 0),
    COALESCE(a.category_id, 0),
    COALESCE(ar.name, ''),
    COALESCE(c.name, ''),
    a.subarea,
    a.location,
    a.manufacturer,
    a.model,
    a.serial_number,
    a.operational_state,
    a.criticality,
    a.notes,
    a.active,
    COALESCE(group_concat(td.title, ' || '), '')
FROM assets a
LEFT JOIN areas ar ON ar.id = a.area_id
LEFT JOIN asset_categories c ON c.id = a.category_id
LEFT JOIN asset_documents ad ON ad.asset_id = a.id
LEFT JOIN technical_documents td ON td.id = ad.document_id
WHERE 1 = 1`)

	if q := strings.TrimSpace(filters.Query); q != "" {
		like := "%" + q + "%"
		sqlb.WriteString(`
  AND (
        a.code LIKE ?
     OR a.name LIKE ?
     OR a.family LIKE ?
     OR a.manufacturer LIKE ?
     OR a.model LIKE ?
     OR a.serial_number LIKE ?
     OR a.location LIKE ?
  )`)
		for i := 0; i < 7; i++ {
			args = append(args, like)
		}
	}
	if filters.AreaID > 0 {
		sqlb.WriteString(` AND a.area_id = ?`)
		args = append(args, filters.AreaID)
	}
	if filters.CategoryID > 0 {
		sqlb.WriteString(` AND a.category_id = ?`)
		args = append(args, filters.CategoryID)
	}
	if state := strings.TrimSpace(filters.State); state != "" {
		sqlb.WriteString(` AND a.operational_state = ?`)
		args = append(args, state)
	}
	if criticality := strings.TrimSpace(filters.Criticality); criticality != "" {
		sqlb.WriteString(` AND a.criticality = ?`)
		args = append(args, criticality)
	}
	switch filters.ActiveFilter {
	case "inactive":
		sqlb.WriteString(` AND a.active = 0`)
	case "", "active":
		sqlb.WriteString(` AND a.active = 1`)
	case "all":
	default:
		sqlb.WriteString(` AND a.active = 1`)
	}

	sqlb.WriteString(`
GROUP BY
    a.id,
    a.code,
    a.name,
    a.family,
    a.area_id,
    a.category_id,
    ar.name,
    c.name,
    a.subarea,
    a.location,
    a.manufacturer,
    a.model,
    a.serial_number,
    a.operational_state,
    a.criticality,
    a.notes,
    a.active
ORDER BY a.name, a.code`)

	rows, err := d.Read.QueryContext(ctx, sqlb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("models: list assets: %w", err)
	}
	defer rows.Close()

	var items []Asset
	for rows.Next() {
		var (
			item         Asset
			documentList string
		)
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Name,
			&item.Family,
			&item.AreaID,
			&item.CategoryID,
			&item.AreaName,
			&item.CategoryName,
			&item.Subarea,
			&item.Location,
			&item.Manufacturer,
			&item.Model,
			&item.SerialNumber,
			&item.OperationalState,
			&item.Criticality,
			&item.Notes,
			&item.Active,
			&documentList,
		); err != nil {
			return nil, fmt.Errorf("models: scan asset: %w", err)
		}
		if documentList != "" {
			item.DocumentTitles = strings.Split(documentList, " || ")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func AssetByID(ctx context.Context, d *db.DB, id int64) (*AssetInput, error) {
	var (
		item       AssetInput
		areaID     sql.NullInt64
		categoryID sql.NullInt64
	)

	err := d.Read.QueryRowContext(ctx,
		`SELECT
		    code,
		    name,
		    family,
		    area_id,
		    category_id,
		    subarea,
		    location,
		    manufacturer,
		    model,
		    serial_number,
		    operational_state,
		    criticality,
		    notes,
		    active
		 FROM assets
		 WHERE id = ?`,
		id,
	).Scan(
		&item.Code,
		&item.Name,
		&item.Family,
		&areaID,
		&categoryID,
		&item.Subarea,
		&item.Location,
		&item.Manufacturer,
		&item.Model,
		&item.SerialNumber,
		&item.OperationalState,
		&item.Criticality,
		&item.Notes,
		&item.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: asset by id: %w", err)
	}
	if areaID.Valid {
		item.AreaID = areaID.Int64
	}
	if categoryID.Valid {
		item.CategoryID = categoryID.Int64
	}

	rows, err := d.Read.QueryContext(ctx,
		`SELECT document_id
		 FROM asset_documents
		 WHERE asset_id = ?
		 ORDER BY document_id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("models: asset document ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var documentID int64
		if err := rows.Scan(&documentID); err != nil {
			return nil, fmt.Errorf("models: scan asset document id: %w", err)
		}
		item.DocumentIDs = append(item.DocumentIDs, documentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &item, nil
}

func CreateAsset(ctx context.Context, d *db.DB, input AssetInput) (int64, error) {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("models: begin create asset tx: %w", err)
	}
	defer tx.Rollback()

	if err := validateAssetReferences(ctx, tx, input); err != nil {
		return 0, err
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO assets (
		    code,
		    name,
		    family,
		    area_id,
		    category_id,
		    subarea,
		    location,
		    manufacturer,
		    model,
		    serial_number,
		    operational_state,
		    criticality,
		    notes,
		    active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Code),
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Family),
		nullableID(input.AreaID),
		nullableID(input.CategoryID),
		strings.TrimSpace(input.Subarea),
		strings.TrimSpace(input.Location),
		strings.TrimSpace(input.Manufacturer),
		strings.TrimSpace(input.Model),
		strings.TrimSpace(input.SerialNumber),
		strings.TrimSpace(input.OperationalState),
		strings.TrimSpace(input.Criticality),
		strings.TrimSpace(input.Notes),
		boolToInt(input.Active),
	)
	if err != nil {
		return 0, fmt.Errorf("models: insert asset: %w", err)
	}

	assetID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("models: asset id: %w", err)
	}

	if err := replaceAssetDocuments(ctx, tx, assetID, input.DocumentIDs); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("models: commit create asset: %w", err)
	}
	return assetID, nil
}

func UpdateAsset(ctx context.Context, d *db.DB, assetID int64, input AssetInput) error {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("models: begin update asset tx: %w", err)
	}
	defer tx.Rollback()

	if err := validateAssetReferences(ctx, tx, input); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE assets
		 SET code = ?,
		     name = ?,
		     family = ?,
		     area_id = ?,
		     category_id = ?,
		     subarea = ?,
		     location = ?,
		     manufacturer = ?,
		     model = ?,
		     serial_number = ?,
		     operational_state = ?,
		     criticality = ?,
		     notes = ?,
		     active = ?,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`,
		strings.TrimSpace(input.Code),
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Family),
		nullableID(input.AreaID),
		nullableID(input.CategoryID),
		strings.TrimSpace(input.Subarea),
		strings.TrimSpace(input.Location),
		strings.TrimSpace(input.Manufacturer),
		strings.TrimSpace(input.Model),
		strings.TrimSpace(input.SerialNumber),
		strings.TrimSpace(input.OperationalState),
		strings.TrimSpace(input.Criticality),
		strings.TrimSpace(input.Notes),
		boolToInt(input.Active),
		assetID,
	)
	if err != nil {
		return fmt.Errorf("models: update asset: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("models: update asset: asset %d not found", assetID)
	}

	if err := replaceAssetDocuments(ctx, tx, assetID, input.DocumentIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("models: commit update asset: %w", err)
	}
	return nil
}

func validateAssetReferences(ctx context.Context, tx *sql.Tx, input AssetInput) error {
	if input.AreaID > 0 {
		ok, err := existsByID(ctx, tx, "areas", input.AreaID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("models: area %d not found", input.AreaID)
		}
	}
	if input.CategoryID > 0 {
		ok, err := existsByID(ctx, tx, "asset_categories", input.CategoryID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("models: category %d not found", input.CategoryID)
		}
	}
	if len(input.DocumentIDs) == 0 {
		return nil
	}

	ids := uniqueInt64s(input.DocumentIDs)
	var sqlb strings.Builder
	sqlb.WriteString(`SELECT COUNT(*) FROM technical_documents WHERE id IN (`)
	for i := range ids {
		if i > 0 {
			sqlb.WriteString(",")
		}
		sqlb.WriteString("?")
	}
	sqlb.WriteString(`)`)

	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	var count int
	if err := tx.QueryRowContext(ctx, sqlb.String(), args...).Scan(&count); err != nil {
		return fmt.Errorf("models: validate technical documents: %w", err)
	}
	if count != len(ids) {
		return fmt.Errorf("models: one or more technical documents do not exist")
	}
	return nil
}

func replaceAssetDocuments(ctx context.Context, tx *sql.Tx, assetID int64, documentIDs []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM asset_documents WHERE asset_id = ?`, assetID); err != nil {
		return fmt.Errorf("models: delete asset documents: %w", err)
	}
	for _, documentID := range uniqueInt64s(documentIDs) {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO asset_documents (asset_id, document_id, relation_kind)
			 VALUES (?, ?, 'source')`,
			assetID, documentID,
		); err != nil {
			return fmt.Errorf("models: insert asset document: %w", err)
		}
	}
	return nil
}

func existsByID(ctx context.Context, tx *sql.Tx, table string, id int64) (bool, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE id = ?`, table)
	var count int
	if err := tx.QueryRowContext(ctx, query, id).Scan(&count); err != nil {
		return false, fmt.Errorf("models: exists by id in %s: %w", table, err)
	}
	return count == 1, nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}
