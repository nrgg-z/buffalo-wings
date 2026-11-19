package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buffalo-pos/internal/catalog/domain"
)

type CatalogRepository struct {
	db *sql.DB
}

func NewCatalogRepository(db *sql.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) GetFullCatalog(ctx context.Context) ([]domain.Category, error) {
	// 1. Categorías activas
	catRows, err := r.db.QueryContext(ctx, `
		SELECT id, name, station, sort_order
		FROM categories
		WHERE is_active = 1
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando categorías: %w", err)
	}
	defer catRows.Close()

	categories := []domain.Category{}
	catIdx := map[int64]int{}
	for catRows.Next() {
		var c domain.Category
		if err := catRows.Scan(&c.ID, &c.Name, &c.Station, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("escaneando categoría: %w", err)
		}
		c.Products = []domain.Product{}
		catIdx[c.ID] = len(categories)
		categories = append(categories, c)
	}
	if err := catRows.Err(); err != nil {
		return nil, err
	}

	// 2. Grupos de modificadores activos
	grpRows, err := r.db.QueryContext(ctx, `
		SELECT id, name, is_required, min_selections, max_selections
		FROM modifier_groups
		WHERE is_active = 1
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando grupos de modificadores: %w", err)
	}
	defer grpRows.Close()

	groupsByID := map[int64]*domain.ModifierGroup{}
	for grpRows.Next() {
		g := &domain.ModifierGroup{Options: []domain.ModifierOption{}}
		if err := grpRows.Scan(&g.ID, &g.Name, &g.Required, &g.MinSelections, &g.MaxSelections); err != nil {
			return nil, fmt.Errorf("escaneando grupo de modificadores: %w", err)
		}
		groupsByID[g.ID] = g
	}
	if err := grpRows.Err(); err != nil {
		return nil, err
	}

	// 3. Opciones de modificadores activas
	optRows, err := r.db.QueryContext(ctx, `
		SELECT id, group_id, name, extra_price, is_available
		FROM modifier_options
		WHERE is_active = 1
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando opciones de modificadores: %w", err)
	}
	defer optRows.Close()

	for optRows.Next() {
		var o domain.ModifierOption
		if err := optRows.Scan(&o.ID, &o.GroupID, &o.Name, &o.ExtraPrice, &o.Available); err != nil {
			return nil, fmt.Errorf("escaneando opción de modificador: %w", err)
		}
		if g, ok := groupsByID[o.GroupID]; ok {
			g.Options = append(g.Options, o)
		}
	}
	if err := optRows.Err(); err != nil {
		return nil, err
	}

	// 4. Relación producto -> grupos
	linkRows, err := r.db.QueryContext(ctx, `
		SELECT product_id, group_id
		FROM product_modifier_groups
		ORDER BY product_id ASC, group_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando relación producto-modificadores: %w", err)
	}
	defer linkRows.Close()

	prodGroups := map[int64][]domain.ModifierGroup{}
	for linkRows.Next() {
		var prodID, grpID int64
		if err := linkRows.Scan(&prodID, &grpID); err != nil {
			return nil, fmt.Errorf("escaneando relación producto-modificadores: %w", err)
		}
		if g, ok := groupsByID[grpID]; ok {
			prodGroups[prodID] = append(prodGroups[prodID], *g)
		}
	}
	if err := linkRows.Err(); err != nil {
		return nil, err
	}

	// 5. Productos activos
	prodRows, err := r.db.QueryContext(ctx, `
		SELECT id, category_id, name, base_price, is_available
		FROM products
		WHERE is_active = 1
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando productos: %w", err)
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var p domain.Product
		if err := prodRows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.BasePrice, &p.Available); err != nil {
			return nil, fmt.Errorf("escaneando producto: %w", err)
		}
		if gs, ok := prodGroups[p.ID]; ok {
			p.ModifierGroups = gs
		} else {
			p.ModifierGroups = []domain.ModifierGroup{}
		}
		if idx, ok := catIdx[p.CategoryID]; ok {
			categories[idx].Products = append(categories[idx].Products, p)
		}
	}
	if err := prodRows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *CatalogRepository) SetProductAvailability(ctx context.Context, productID int64, available bool) (*domain.Product, error) {
	var p domain.Product
	err := r.db.QueryRowContext(ctx, `
		SELECT id, category_id, name, base_price, is_available
		FROM products
		WHERE id = ? AND is_active = 1
	`, productID).Scan(&p.ID, &p.CategoryID, &p.Name, &p.BasePrice, &p.Available)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("buscando producto %d: %w", productID, err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE products SET is_available = ? WHERE id = ? AND is_active = 1
	`, available, productID)
	if err != nil {
		return nil, fmt.Errorf("actualizando disponibilidad del producto %d: %w", productID, err)
	}

	p.Available = available
	p.ModifierGroups = []domain.ModifierGroup{}
	return &p, nil
}

func (r *CatalogRepository) SetModifierAvailability(ctx context.Context, optionID int64, available bool) (*domain.ModifierOption, error) {
	var o domain.ModifierOption
	err := r.db.QueryRowContext(ctx, `
		SELECT id, group_id, name, extra_price, is_available
		FROM modifier_options
		WHERE id = ? AND is_active = 1
	`, optionID).Scan(&o.ID, &o.GroupID, &o.Name, &o.ExtraPrice, &o.Available)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrModifierNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("buscando modificador %d: %w", optionID, err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE modifier_options SET is_available = ? WHERE id = ? AND is_active = 1
	`, available, optionID)
	if err != nil {
		return nil, fmt.Errorf("actualizando disponibilidad del modificador %d: %w", optionID, err)
	}

	o.Available = available
	return &o, nil
}
