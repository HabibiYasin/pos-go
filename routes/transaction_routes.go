package routes

import (
	"pos-go/controllers"
	"pos-go/middleware"

	"github.com/gin-gonic/gin"
)

func TransactionRoutes(r *gin.Engine) {
	transaction := r.Group("/transaction")
	{
		// Public - create transaction (checkout untuk customer)
		transaction.POST("", controllers.CreateTransaction)

		// Public - webhook dari Midtrans (PENTING!)
		transaction.POST("/notification", controllers.HandleMidtransNotification)

		// Admin & Kasir - lihat semua transaksi
		transaction.GET("", middleware.AuthMiddleware(), controllers.GetAllTransactions)

		// Admin & Kasir - lihat detail transaksi
		transaction.GET("/:id", middleware.AuthMiddleware(), controllers.GetTransactionByID)

		// Admin, Kasir & Koki - proses status pesanan
		transaction.PATCH("/:id/status", middleware.AuthMiddleware(), middleware.RequireRole("admin", "kasir", "koki"), controllers.UpdateTransactionStatus)
		// Compatibility for older frontend bundles.
		transaction.PATCH("/:id/order-status", middleware.AuthMiddleware(), middleware.RequireRole("admin", "kasir", "koki"), controllers.UpdateLegacyOrderStatus)
	}
}
