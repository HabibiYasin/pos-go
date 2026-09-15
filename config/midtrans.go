package config

import (
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"os"
	"strings"
)

var MidtransClient snap.Client
var MidtransStatusClient coreapi.Client

func MidtransServerKey() string {
	key := strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY"))
	if key == "" {
		key = strings.TrimSpace(os.Getenv("server_key_mid"))
	}
	return key
}

func MidtransClientKey() string { return strings.TrimSpace(os.Getenv("MIDTRANS_CLIENT_KEY")) }

func MidtransReady() bool {
	return strings.HasPrefix(MidtransServerKey(), "SB-Mid-server-") && strings.HasPrefix(MidtransClientKey(), "SB-Mid-client-")
}

func InitMidtrans() {
	serverKey := MidtransServerKey()

	MidtransClient.New(serverKey, midtrans.Sandbox)
	MidtransStatusClient.New(serverKey, midtrans.Sandbox)
	// Untuk production: midtrans.Production
}
