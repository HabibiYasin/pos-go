package controllers

import (
	"errors"
	"pos-go/dto"
	"pos-go/services"
	"pos-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var settlementService services.SettlementService = services.NewSettlementService()

// GetSettlement GET /settlement?date=YYYY-MM-DD — expected_cash + settlement (jika sudah ada) untuk user yang login.
func GetSettlement(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		utils.ErrorResponseBadRequest(c, "Parameter date (YYYY-MM-DD) wajib diisi", nil)
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists || userIDVal == nil {
		utils.ErrorResponseUnauthorized(c, "User tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		utils.ErrorResponseUnauthorized(c, "User ID tidak valid")
		return
	}

	resp, err := settlementService.GetSettlementWithExpected(dateStr, userID)
	if err != nil {
		if errors.Is(err, services.ErrDatabaseError) {
			utils.ErrorResponseInternal(c, "Gagal mengambil data settlement")
			return
		}
		utils.ErrorResponseBadRequest(c, "Tanggal tidak valid. Gunakan format YYYY-MM-DD", nil)
		return
	}

	utils.SuccessResponseOK(c, "Data settlement berhasil diambil", resp)
}

// CreateSettlement POST /settlement — simpan settlement (tutup kasir). Body: { date, actual_cash }.
func CreateSettlement(c *gin.Context) {
	saveSettlement(c, false)
}

func UpdateSettlement(c *gin.Context) {
	saveSettlement(c, true)
}

func saveSettlement(c *gin.Context, update bool) {
	var req dto.CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseBadRequest(c, "Format data tidak valid. Perlu date (YYYY-MM-DD) dan actual_cash", nil)
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists || userIDVal == nil {
		utils.ErrorResponseUnauthorized(c, "User tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		utils.ErrorResponseUnauthorized(c, "User ID tidak valid")
		return
	}

	var settlement *dto.SettlementResponse
	if update {
		settlement, err = settlementService.UpdateSettlement(userID, req.Date, *req.ActualCash)
	} else {
		settlement, err = settlementService.CreateSettlement(userID, req.Date, *req.ActualCash)
	}
	if err != nil {
		if errors.Is(err, services.ErrSettlementNotFound) {
			utils.ErrorResponseNotFound(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrInvalidSettlementCash) {
			utils.ErrorResponseBadRequest(c, err.Error(), nil)
			return
		}
		if errors.Is(err, services.ErrSettlementAlreadyExists) {
			utils.ErrorResponseBadRequest(c, "Settlement untuk tanggal ini sudah ada", nil)
			return
		}
		if errors.Is(err, services.ErrDatabaseError) {
			utils.ErrorResponseInternal(c, "Gagal menyimpan settlement")
			return
		}
		utils.ErrorResponseBadRequest(c, "Tanggal tidak valid. Gunakan format YYYY-MM-DD", nil)
		return
	}

	if update {
		utils.SuccessResponseOK(c, "Settlement berhasil diperbarui", settlement)
	} else {
		utils.SuccessResponseCreated(c, "Settlement berhasil disimpan", settlement)
	}
}

func ResetSettlement(c *gin.Context) {
	if !services.SettlementDebugResetEnabled() {
		utils.ErrorResponseForbidden(c, services.ErrSettlementResetDisabled.Error())
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		utils.ErrorResponseUnauthorized(c, "User ID tidak valid")
		return
	}
	if err := settlementService.ResetSettlement(userID, c.Query("date")); err != nil {
		if errors.Is(err, services.ErrDatabaseError) {
			utils.ErrorResponseInternal(c, "Gagal mereset settlement")
		} else {
			utils.ErrorResponseBadRequest(c, "Tanggal tidak valid. Gunakan format YYYY-MM-DD", nil)
		}
		return
	}
	utils.SuccessResponseOK(c, "Settlement berhasil direset", nil)
}

// GetSettlementStatusByDate GET /settlement/status-by-date?date=YYYY-MM-DD — admin only. Daftar kasir + expected cash + status settlement.
func GetSettlementStatusByDate(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		utils.ErrorResponseBadRequest(c, "Parameter date (YYYY-MM-DD) wajib diisi", nil)
		return
	}

	resp, err := settlementService.GetSettlementStatusByDate(dateStr)
	if err != nil {
		if errors.Is(err, services.ErrDatabaseError) {
			utils.ErrorResponseInternal(c, "Gagal mengambil status settlement")
			return
		}
		utils.ErrorResponseBadRequest(c, "Tanggal tidak valid. Gunakan format YYYY-MM-DD", nil)
		return
	}

	utils.SuccessResponseOK(c, "Status settlement per kasir berhasil diambil", resp)
}
