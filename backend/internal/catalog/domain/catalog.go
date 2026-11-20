package domain

import "errors"

var (
	ErrProductNotFound  = errors.New("producto no encontrado")
	ErrModifierNotFound = errors.New("modificador no encontrado")
)

type ModifierOption struct {
	ID         int64   `json:"id"`
	GroupID    int64   `json:"group_id"`
	Name       string  `json:"name"`
	ExtraPrice float64 `json:"extra_price"`
	Available  bool    `json:"is_available"`
}

type ModifierGroup struct {
	ID            int64            `json:"id"`
	Name          string           `json:"name"`
	Required      bool             `json:"is_required"`
	MinSelections int              `json:"min_selections"`
	MaxSelections int              `json:"max_selections"`
	Options       []ModifierOption `json:"options"`
}

type Product struct {
	ID             int64           `json:"id"`
	CategoryID     int64           `json:"category_id"`
	Name           string          `json:"name"`
	BasePrice      float64         `json:"base_price"`
	Available      bool            `json:"is_available"`
	ModifierGroups []ModifierGroup `json:"modifier_groups"`
}

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Station   string    `json:"station"`
	SortOrder int       `json:"sort_order"`
	Products  []Product `json:"products"`
}
