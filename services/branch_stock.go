package services

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	menu_model "pos-go/models/menu_model"
)

var ErrInsufficientStock = errors.New("Stok menu tidak mencukupi atau menu tidak tersedia di cabang yang dipilih")

var ErrBranchStocks = errors.New("Data stok cabang tidak valid")

func parseBranchStocks(raw string) ([]menu_model.BranchStock, error) {
	if raw == "" {
		return nil, nil
	}
	var stocks []menu_model.BranchStock
	if json.Unmarshal([]byte(raw), &stocks) != nil || len(stocks) != 3 {
		return nil, ErrBranchStocks
	}
	seen := map[string]bool{}
	for _, stock := range stocks {
		if !menu_model.ValidBranch(stock.Branch) || seen[stock.Branch] || stock.Stock < 0 {
			return nil, ErrBranchStocks
		}
		seen[stock.Branch] = true
	}
	return stocks, nil
}

// Conditional decrement is atomic; concurrent orders cannot make stock negative.
func consumeBranchStock(tx *gorm.DB, menuID uuid.UUID, branch string, quantity int) error {
	if !menu_model.ValidBranch(branch) || quantity <= 0 {
		return errors.New("Cabang atau jumlah pesanan tidak valid")
	}
	result := tx.Model(&menu_model.BranchStock{}).Where("menu_id = ? AND branch = ? AND is_available = ? AND stock >= ?", menuID, branch, true, quantity).UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
	if result.Error != nil {
		return ErrDatabaseError
	}
	if result.RowsAffected != 1 {
		return ErrInsufficientStock
	}
	return nil
}
