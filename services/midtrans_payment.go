package services

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go/coreapi"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"pos-go/config"
	"pos-go/dto"
	transaction_model "pos-go/models/transaction_model"
	"strconv"
)

var ErrPaymentUnavailable = errors.New("Pembayaran non-tunai belum tersedia. Silakan pilih tunai atau coba lagi nanti")
var ErrInvalidPaymentNotification = errors.New("Notifikasi pembayaran tidak valid")

func ValidMidtransSignature(n dto.MidtransNotification, key string) bool {
	if key == "" || n.OrderID == "" || n.StatusCode == "" || n.GrossAmount == "" {
		return false
	}
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + key))
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(n.SignatureKey)) == 1
}

// Only verified provider statuses can change payment state. Repeated webhooks preserve kitchen progress.
func (s TransactionService) ApplyMidtransStatus(status *coreapi.TransactionStatusResponse) error {
	id, err := uuid.Parse(status.OrderID)
	if err != nil {
		return ErrInvalidPaymentNotification
	}
	amount, err := strconv.ParseFloat(status.GrossAmount, 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return ErrInvalidPaymentNotification
	}
	return config.DB.Transaction(func(db *gorm.DB) error {
		var tx transaction_model.Transaction
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tx, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTransactionNotFound
			}
			return ErrDatabaseError
		}
		if tx.PaymentMethod == "cash" || amount != float64(int64(tx.TotalAmount)) {
			return ErrInvalidPaymentNotification
		}
		payment := ""
		switch status.TransactionStatus {
		case "settlement":
			payment = "paid"
		case "capture":
			if status.FraudStatus == "accept" {
				payment = "paid"
			}
		case "expire":
			payment = "expired"
		case "cancel", "deny":
			payment = "cancelled"
		}
		if payment == "" || tx.PaymentStatus == "paid" || tx.PaymentStatus == payment {
			return nil
		}
		updates := map[string]interface{}{"payment_status": payment}
		if payment != "paid" && tx.OrderStatus == "pending" {
			updates["order_status"] = "cancelled"
		}
		if err := db.Model(&tx).Updates(updates).Error; err != nil {
			return ErrDatabaseError
		}
		return nil
	})
}
