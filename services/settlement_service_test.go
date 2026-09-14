package services

import (
	"errors"
	"fmt"
	"math"
	"pos-go/config"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func settlementTestDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = gdb
	t.Cleanup(func() {
		config.DB = previous
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		db.Close()
	})
	return mock
}

func TestUpdateSettlementRecalculatesCashForAuthenticatedUser(t *testing.T) {
	for _, actual := range []float64{11111, 33000, 0, 33000.50} {
		t.Run(fmt.Sprintf("cash_%.2f", actual), func(t *testing.T) {
			mock := settlementTestDB(t)
			user, id := uuid.New(), uuid.New()
			date := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
			mock.ExpectQuery(`SELECT COALESCE\(SUM\(total_amount\), 0\).*closed_by_user_id = \$6`).
				WithArgs(date, date.AddDate(0, 0, 1), "completed", "paid", "cash", user).
				WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(33000))
			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "settlements" SET .*WHERE \(date = \$5 AND user_id = \$6\)`).
				WithArgs(actual, actual-33000, float64(33000), sqlmock.AnyArg(), date, user).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(`SELECT .*FROM "settlements" WHERE \(date = \$1 AND user_id = \$2\)`).
				WithArgs(date, user, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "date", "user_id", "actual_cash", "expected_cash", "discrepancy"}).AddRow(id, date, user, actual, 33000, actual-33000))
			mock.ExpectCommit()
			result, err := NewSettlementService().UpdateSettlement(user, "2026-09-14", actual)
			if err != nil {
				t.Fatal(err)
			}
			if result.ActualCash != actual || result.Discrepancy != actual-33000 || result.UserID != user.String() {
				t.Fatalf("unexpected settlement: %+v", result)
			}
		})
	}
}

func TestSettlementRejectsInvalidAmountsBeforeDatabase(t *testing.T) {
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1), 10000000000000} {
		if _, err := NewSettlementService().UpdateSettlement(uuid.New(), "2026-09-14", amount); !errors.Is(err, ErrInvalidSettlementCash) {
			t.Fatalf("amount %v: %v", amount, err)
		}
		if _, err := NewSettlementService().CreateSettlement(uuid.New(), "2026-09-14", amount); !errors.Is(err, ErrInvalidSettlementCash) {
			t.Fatalf("amount %v: %v", amount, err)
		}
	}
}

func TestUpdateSettlementCannotCreateOrModifyAnotherCashiersRecord(t *testing.T) {
	mock := settlementTestDB(t)
	user := uuid.New()
	date := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(total_amount\), 0\).*closed_by_user_id = \$6`).
		WithArgs(date, date.AddDate(0, 0, 1), "completed", "paid", "cash", user).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "settlements" SET .*WHERE \(date = \$5 AND user_id = \$6\)`).
		WithArgs(float64(0), float64(0), float64(0), sqlmock.AnyArg(), date, user).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	if _, err := NewSettlementService().UpdateSettlement(user, "2026-09-14", 0); !errors.Is(err, ErrSettlementNotFound) {
		t.Fatalf("expected own settlement not found, got %v", err)
	}
}

func TestResetSettlementDebugGuardAndScope(t *testing.T) {
	t.Setenv("SETTLEMENT_DEBUG_RESET", "")
	user := uuid.New()
	if err := NewSettlementService().ResetSettlement(user, "2026-09-14"); !errors.Is(err, ErrSettlementResetDisabled) {
		t.Fatal(err)
	}
	t.Setenv("SETTLEMENT_DEBUG_RESET", "true")
	if err := NewSettlementService().ResetSettlement(user, "invalid"); err == nil {
		t.Fatal("accepted invalid date")
	}
	mock := settlementTestDB(t)
	date := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	// Both an existing and already-reset settlement are safe for repeated automation.
	for _, rows := range []int64{1, 0} {
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "settlements" WHERE date = \$1 AND user_id = \$2`).WithArgs(date, user).WillReturnResult(sqlmock.NewResult(0, rows))
		mock.ExpectCommit()
		if err := NewSettlementService().ResetSettlement(user, "2026-09-14"); err != nil {
			t.Fatal(err)
		}
	}
}
