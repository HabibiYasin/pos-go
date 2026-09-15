package services

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestBranchStockInput(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`[{"branch":"unknown","stock":1},{"branch":"depok","stock":1},{"branch":"tokyo","stock":1}]`,
		`[{"branch":"depok","stock":1},{"branch":"depok","stock":1},{"branch":"tokyo","stock":1}]`,
		`[{"branch":"jakarta-selatan","stock":-1},{"branch":"depok","stock":1},{"branch":"tokyo","stock":1}]`,
		`[{"branch":"jakarta-selatan","stock":1.5},{"branch":"depok","stock":1},{"branch":"tokyo","stock":1}]`,
	} {
		if _, err := parseBranchStocks(raw); err == nil {
			t.Errorf("accepted invalid stocks: %s", raw)
		}
	}
	stocks, err := parseBranchStocks(`[{"branch":"jakarta-selatan","is_available":true,"stock":0},{"branch":"depok","is_available":true,"stock":10},{"branch":"tokyo","is_available":false,"stock":3}]`)
	if err != nil || len(stocks) != 3 || stocks[0].Stock != 0 || stocks[2].IsAvailable {
		t.Fatal("valid branch stocks rejected")
	}
}
func TestConsumeBranchStock(t *testing.T) {
	for _, affected := range []int64{0, 1} {
		sqlDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer sqlDB.Close()
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{SkipDefaultTransaction: true})
		if err != nil {
			t.Fatal(err)
		}
		id := uuid.New()
		mock.ExpectExec(`UPDATE "branch_stocks" SET "stock"=stock - \$1 WHERE menu_id = \$2 AND branch = \$3 AND is_available = \$4 AND stock >= \$5`).WithArgs(3, id, "depok", true, 3).WillReturnResult(sqlmock.NewResult(0, affected))
		err = consumeBranchStock(db, id, "depok", 3)
		if affected == 0 && !errors.Is(err, ErrInsufficientStock) {
			t.Fatalf("expected stock error, got %v", err)
		}
		if affected == 1 && err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBranchResetDates(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	if branchDate("tokyo", now) != "2026-09-16" || branchDate("depok", now) != "2026-09-15" {
		t.Fatal("incorrect local reset date")
	}
}
func TestResetStockScope(t *testing.T) {
	for _, force := range []bool{false, true} {
		sqlDB, mock, _ := sqlmock.New()
		defer sqlDB.Close()
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{SkipDefaultTransaction: true})
		if err != nil {
			t.Fatal(err)
		}
		pattern := `UPDATE "branch_stocks" SET .* WHERE branch = \$1`
		if !force {
			pattern += ` AND \(reset_date IS NULL OR reset_date < .*`
		}
		mock.ExpectExec(pattern).WithArgs("depok").WillReturnResult(sqlmock.NewResult(0, 15))
		if err := ResetBranchStocks(db, "depok", force); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
