package domain

import (
	"errors"
	"time"
)

var (
	ErrItemNotFound      = errors.New("ítem no encontrado")
	ErrInvalidStation    = errors.New("estación inválida (usa KITCHEN o BAR)")
	ErrInvalidStatus     = errors.New("estado inválido (usa IN_PROGRESS, READY o DELIVERED)")
	ErrInvalidTransition = errors.New("transición de estado no permitida")
)

type KDSItem struct {
	ID             int64      `json:"id"`
	OrderID        int64      `json:"order_id"`
	TableID        int64      `json:"table_id"`
	TableCode      string     `json:"table_code"`
	ProductName    string     `json:"product_name"`
	Station        string     `json:"station"`
	Quantity       int        `json:"quantity"`
	Notes          string     `json:"notes,omitempty"`
	Status         string     `json:"status"`
	Modifiers      []string   `json:"modifiers"`
	CreatedAt      time.Time  `json:"created_at"`
	ReadyAt        *time.Time `json:"ready_at,omitempty"`
	ElapsedMinutes int        `json:"elapsed_minutes"`
	AlertLevel     string     `json:"alert_level"`
}

// ComputeTiming calcula minutos transcurridos y color del semáforo KDS.
func (k *KDSItem) ComputeTiming(now time.Time) {
	mins := int(now.Sub(k.CreatedAt).Minutes())
	if mins < 0 {
		mins = 0
	}
	k.ElapsedMinutes = mins
	switch {
	case mins >= 15:
		k.AlertLevel = "RED"
	case mins >= 10:
		k.AlertLevel = "YELLOW"
	default:
		k.AlertLevel = "GREEN"
	}
}

func IsValidTransition(from, to string) bool {
	switch from {
	case "PENDING":
		return to == "IN_PROGRESS" || to == "READY"
	case "IN_PROGRESS":
		return to == "READY"
	case "READY":
		return to == "DELIVERED"
	default:
		return false
	}
}
