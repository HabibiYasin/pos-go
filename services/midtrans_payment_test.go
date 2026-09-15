package services

import (
	"crypto/sha512"
	"encoding/hex"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go/coreapi"
	"pos-go/dto"
	"testing"
)

func TestMidtransSignatureRejectsTampering(t *testing.T) {
	n := dto.MidtransNotification{OrderID: "order", StatusCode: "200", GrossAmount: "33000.00"}
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + "secret"))
	n.SignatureKey = hex.EncodeToString(sum[:])
	if !ValidMidtransSignature(n, "secret") {
		t.Fatal("valid signature rejected")
	}
	if ValidMidtransSignature(n, "") || ValidMidtransSignature(n, "wrong") {
		t.Fatal("invalid key accepted")
	}
	n.GrossAmount = "1.00"
	if ValidMidtransSignature(n, "secret") {
		t.Fatal("tampered amount accepted")
	}
}

func TestVerifiedMidtransStatusPreservesKitchenProgress(t *testing.T) {
	for _, tc := range []struct {
		name, status, fraud, current, order, amount string
		update, bad                                 bool
	}{
		{"paid while cooking", "settlement", "", "pending", "cooking", "33000.00", true, false},
		{"accepted card", "capture", "accept", "pending", "pending", "33000.00", true, false},
		{"challenged card", "capture", "challenge", "pending", "pending", "33000.00", false, false},
		{"duplicate settlement", "settlement", "", "paid", "completed", "33000.00", false, false},
		{"late expire", "expire", "", "paid", "ready", "33000.00", false, false},
		{"amount mismatch", "settlement", "", "pending", "pending", "1.00", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := settlementTestDB(t)
			id := uuid.New()
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT .*FROM "transactions".*FOR UPDATE`).WithArgs(id, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "total_amount", "payment_method", "payment_status", "order_status"}).AddRow(id, 33000, "e_wallet", tc.current, tc.order))
			if tc.update {
				mock.ExpectExec(`UPDATE "transactions" SET "payment_status"=\$1,"updated_at"=\$2 WHERE`).WithArgs("paid", sqlmock.AnyArg(), id).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			if tc.bad {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			err := NewTransactionService().ApplyMidtransStatus(&coreapi.TransactionStatusResponse{OrderID: id.String(), GrossAmount: tc.amount, TransactionStatus: tc.status, FraudStatus: tc.fraud})
			if (err != nil) != tc.bad {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
