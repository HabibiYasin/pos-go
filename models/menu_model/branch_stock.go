package menu_model

import "github.com/google/uuid"

type BranchStock struct {
	InitialStock int       `gorm:"not null;default:0;check:branch_initial_stock_nonnegative,initial_stock >= 0" json:"initial_stock"`
	ResetDate    string    `gorm:"type:date" json:"-"`
	MenuID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	Branch       string    `gorm:"type:varchar(32);primaryKey;check:branch_stock_branch_check,branch IN ('jakarta-selatan','depok','tokyo')" json:"branch"`
	IsAvailable  bool      `gorm:"not null" json:"is_available"`
	Stock        int       `gorm:"not null;check:branch_stock_nonnegative,stock >= 0" json:"stock"`
}

func ValidBranch(branch string) bool {
	return branch == "jakarta-selatan" || branch == "depok" || branch == "tokyo"
}
