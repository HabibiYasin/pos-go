package services

import (
	"encoding/json"
	"errors"
	"log"
	"pos-go/config"
	"time"

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
	for i, stock := range stocks {
		if !menu_model.ValidBranch(stock.Branch) || seen[stock.Branch] || stock.Stock < 0 {
			return nil, ErrBranchStocks
		}
		seen[stock.Branch] = true
		stocks[i].InitialStock = stock.Stock
		stocks[i].ResetDate = branchDate(stock.Branch, time.Now())
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

func branchDate(branch string, now time.Time) string {
	offset := 7
	if branch == "tokyo" {
		offset = 9
	}
	return now.In(time.FixedZone(branch, offset*3600)).Format("2006-01-02")
}

// A conditional update makes repeated calls and multiple server instances safe.
func ResetBranchStocks(db *gorm.DB, branch string, force bool) error {
	if branch != "" && !menu_model.ValidBranch(branch) {
		return ErrBranchStocks
	}
	query := db.Model(&menu_model.BranchStock{})
	if branch != "" {
		query = query.Where("branch = ?", branch)
	}
	dateSQL := "(CURRENT_TIMESTAMP AT TIME ZONE CASE WHEN branch = 'tokyo' THEN 'Asia/Tokyo' ELSE 'Asia/Jakarta' END)::date"
	if !force {
		query = query.Where("reset_date IS NULL OR reset_date < " + dateSQL)
	}
	return query.Updates(map[string]interface{}{"stock": gorm.Expr("initial_stock"), "reset_date": gorm.Expr(dateSQL)}).Error
}

func StartStockResetWorker() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if err := ResetBranchStocks(config.DB, "", false); err != nil {
				log.Println("Reset stok harian gagal")
			}
			<-ticker.C
		}
	}()
}
