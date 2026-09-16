package promo_model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Promo struct {
	ValidBranches int            `gorm:"type:smallint;not null;default:7;check:promos_valid_branches_check,valid_branches BETWEEN 1 AND 7" json:"valid_branches"`
	ValidDays     int            `gorm:"type:smallint;not null;default:127;check:promos_valid_days_check,valid_days BETWEEN 1 AND 127" json:"valid_days"` // Bitmask: Sunday=1, Monday=2, ..., Saturday=64.
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Code          string         `gorm:"type:varchar(50);unique;not null" json:"code"`
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	Description   string         `gorm:"type:text" json:"description"`
	Type          string         `gorm:"type:varchar(20);not null" json:"type"` // "percentage" atau "fixed"
	Value         float64        `gorm:"type:decimal(15,2);not null" json:"value"`
	MinPurchase   float64        `gorm:"type:decimal(15,2);default:0" json:"min_purchase"`
	MaxDiscount   float64        `gorm:"type:decimal(15,2);default:0" json:"max_discount"` // 0 = unlimited
	UsageLimit    int            `gorm:"type:int;default:0" json:"usage_limit"`            // 0 = unlimited
	UsageCount    int            `gorm:"type:int;default:0" json:"usage_count"`
	StartDate     time.Time      `gorm:"type:timestamp;not null" json:"start_date"`
	EndDate       time.Time      `gorm:"type:timestamp;not null" json:"end_date"`
	IsActive      bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Hook untuk generate UUID sebelum insert
func (promo *Promo) BeforeCreate(tx *gorm.DB) (err error) {
	promo.ID = uuid.New()
	return
}
