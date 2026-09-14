package services

import (
	"errors"
	"math"
	"os"
	"pos-go/config"
	"pos-go/dto"
	settlement_model "pos-go/models/settlement_model"
	transaction_model "pos-go/models/transaction_model"
	user_model "pos-go/models/user_model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrSettlementAlreadyExists = errors.New("Settlement untuk tanggal ini sudah ada")
var ErrSettlementNotFound = errors.New("Settlement untuk tanggal ini belum ada")
var ErrInvalidSettlementCash = errors.New("Jumlah uang tunai harus berupa angka valid, minimal 0")
var ErrSettlementResetDisabled = errors.New("Reset settlement hanya tersedia dalam mode debugging")

func SettlementDebugResetEnabled() bool {
	// Enabled for this debugging deployment at the owner's request.
	// Render can disable it immediately with SETTLEMENT_DEBUG_RESET=false.
	value, configured := os.LookupEnv("SETTLEMENT_DEBUG_RESET")
	if !configured {
		return true
	}
	return value == "true"
}

func validateSettlementCash(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 9999999999999.99 {
		return ErrInvalidSettlementCash
	}
	return nil
}

type SettlementService struct{}

func NewSettlementService() SettlementService {
	return SettlementService{}
}

// expectedCashFromTransactionsByUser menghitung total tunai yang seharusnya untuk kasir tersebut (transaksi cash, paid, completed, closed_by_user_id = userID).
func expectedCashFromTransactionsByUser(startOfDay, endOfDay time.Time, userID uuid.UUID) (float64, error) {
	var total float64
	err := config.DB.Model(&transaction_model.Transaction{}).
		Select("COALESCE(SUM(total_amount), 0)").
		Where("created_at >= ? AND created_at < ? AND order_status = ? AND payment_status = ? AND payment_method = ? AND closed_by_user_id = ?",
			startOfDay, endOfDay, "completed", "paid", "cash", userID).
		Scan(&total).Error
	if err != nil {
		return 0, ErrDatabaseError
	}
	return total, nil
}

// CreateSettlement menyimpan settlement (tutup kasir). Satu settlement per (date, user_id).
func (s SettlementService) CreateSettlement(userID uuid.UUID, dateStr string, actualCash float64) (*dto.SettlementResponse, error) {
	if err := validateSettlementCash(actualCash); err != nil {
		return nil, err
	}
	actualCash = math.Round(actualCash*100) / 100
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	expectedCash, err := expectedCashFromTransactionsByUser(startOfDay, endOfDay, userID)
	if err != nil {
		return nil, err
	}

	discrepancy := actualCash - expectedCash

	// Cek sudah ada settlement untuk (date, user_id)?
	var existing settlement_model.Settlement
	if err := config.DB.Where("date = ? AND user_id = ?", startOfDay, userID).First(&existing).Error; err == nil {
		return nil, ErrSettlementAlreadyExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDatabaseError
	}

	settlement := settlement_model.Settlement{
		Date:         startOfDay,
		UserID:       userID,
		ExpectedCash: expectedCash,
		ActualCash:   actualCash,
		Discrepancy:  discrepancy,
	}
	if err := config.DB.Create(&settlement).Error; err != nil {
		return nil, ErrDatabaseError
	}

	return toSettlementResponse(&settlement), nil
}

// UpdateSettlement replaces the total handed-in cash, scoped to the authenticated cashier.
func (s SettlementService) UpdateSettlement(userID uuid.UUID, dateStr string, actualCash float64) (*dto.SettlementResponse, error) {
	if err := validateSettlementCash(actualCash); err != nil {
		return nil, err
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	expected, err := s.GetExpectedCashForDateAndUser(dateStr, userID)
	if err != nil {
		return nil, err
	}
	actualCash = math.Round(actualCash*100) / 100
	var settlement settlement_model.Settlement
	err = config.DB.Transaction(func(db *gorm.DB) error {
		result := db.Model(&settlement_model.Settlement{}).
			Where("date = ? AND user_id = ?", date, userID).
			Updates(map[string]interface{}{"actual_cash": actualCash, "expected_cash": expected, "discrepancy": math.Round((actualCash-expected)*100) / 100})
		if result.Error != nil {
			return ErrDatabaseError
		}
		if result.RowsAffected == 0 {
			return ErrSettlementNotFound
		}
		if err := db.Where("date = ? AND user_id = ?", date, userID).First(&settlement).Error; err != nil {
			return ErrDatabaseError
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toSettlementResponse(&settlement), nil
}

// ResetSettlement removes only this user's settlement, allowing automation to create it again.
// Transactions and other cashiers' settlements are unaffected.
func (s SettlementService) ResetSettlement(userID uuid.UUID, dateStr string) error {
	if !SettlementDebugResetEnabled() {
		return ErrSettlementResetDisabled
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return err
	}
	// Hard delete also releases the unique (date, user_id) key for the next test run.
	if err := config.DB.Unscoped().Where("date = ? AND user_id = ?", date, userID).Delete(&settlement_model.Settlement{}).Error; err != nil {
		return ErrDatabaseError
	}
	return nil
}

// GetSettlementByDateAndUser mengembalikan settlement untuk tanggal dan user (kasir) tertentu. Nil jika belum ada.
func (s SettlementService) GetSettlementByDateAndUser(dateStr string, userID uuid.UUID) (*dto.SettlementResponse, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	var settlement settlement_model.Settlement
	if err := config.DB.Where("date = ? AND user_id = ?", startOfDay, userID).First(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // belum ada settlement
		}
		return nil, ErrDatabaseError
	}
	return toSettlementResponse(&settlement), nil
}

// GetSettlementWithExpected mengembalikan expected_cash untuk tanggal + kasir tersebut + settlement jika sudah ada (untuk GET /settlement?date=)
func (s SettlementService) GetSettlementWithExpected(dateStr string, userID uuid.UUID) (*dto.GetSettlementResponse, error) {
	expectedCash, err := s.GetExpectedCashForDateAndUser(dateStr, userID)
	if err != nil {
		return nil, err
	}
	settlement, err := s.GetSettlementByDateAndUser(dateStr, userID)
	if err != nil {
		return nil, err
	}
	return &dto.GetSettlementResponse{
		DebugResetEnabled: SettlementDebugResetEnabled(),
		ExpectedCash:      expectedCash,
		Settlement:        settlement,
	}, nil
}

// GetExpectedCashForDateAndUser mengembalikan expected cash untuk tanggal dan kasir (tunai dari transaksi yang ditutup kasir tersebut).
func (s SettlementService) GetExpectedCashForDateAndUser(dateStr string, userID uuid.UUID) (float64, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, err
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)
	return expectedCashFromTransactionsByUser(startOfDay, endOfDay, userID)
}

// GetSettlementStatusByDate untuk admin: daftar kasir yang punya transaksi di tanggal tersebut + status settlement (sudah/belum).
func (s SettlementService) GetSettlementStatusByDate(dateStr string) (*dto.GetSettlementStatusByDateResponse, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var userIDs []uuid.UUID
	err = config.DB.Model(&transaction_model.Transaction{}).
		Distinct("closed_by_user_id").
		Where("created_at >= ? AND created_at < ? AND order_status = ? AND payment_status = ? AND closed_by_user_id IS NOT NULL",
			startOfDay, endOfDay, "completed", "paid").
		Pluck("closed_by_user_id", &userIDs).Error
	if err != nil {
		return nil, ErrDatabaseError
	}

	items := make([]dto.SettlementStatusItem, 0, len(userIDs))
	for _, uid := range userIDs {
		expected, _ := expectedCashFromTransactionsByUser(startOfDay, endOfDay, uid)
		settlement, _ := s.GetSettlementByDateAndUser(dateStr, uid)
		var name string
		var u user_model.User
		if err := config.DB.Select("name").First(&u, "id = ?", uid).Error; err == nil {
			name = u.Name
		} else {
			name = "-"
		}
		items = append(items, dto.SettlementStatusItem{
			UserID:       uid.String(),
			UserName:     name,
			ExpectedCash: expected,
			Settlement:   settlement,
		})
	}

	return &dto.GetSettlementStatusByDateResponse{
		Date:  dateStr,
		Items: items,
	}, nil
}

func toSettlementResponse(s *settlement_model.Settlement) *dto.SettlementResponse {
	dateStr := s.Date.Format("2006-01-02")
	return &dto.SettlementResponse{
		ID:           s.ID.String(),
		Date:         dateStr,
		UserID:       s.UserID.String(),
		ExpectedCash: s.ExpectedCash,
		ActualCash:   s.ActualCash,
		Discrepancy:  s.Discrepancy,
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
	}
}
