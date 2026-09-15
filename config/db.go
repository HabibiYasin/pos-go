package config

import (
	"fmt"
	"log"
	"os"

	category_model "pos-go/models/category_model"
	menu_model "pos-go/models/menu_model"
	promo_model "pos-go/models/promo_model"
	settlement_model "pos-go/models/settlement_model"
	transaction_model "pos-go/models/transaction_model"
	user_model "pos-go/models/user_model"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {

	// koneksi .env file
	// Render supplies environment variables without a local .env file.
	_ = godotenv.Load()

	// Ambil variabel dari .env
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Format DSN
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "require"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", host, user, password, dbname, port, sslmode)
	}

	// koneksi ke database
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke database:", err)
	}

	log.Println("Berhasil terhubung ke database PostgreSQL")

	// AutoMigrate tabel
	err = database.AutoMigrate(
		&user_model.User{},
		&category_model.Category{},
		&menu_model.Menu{},
		&menu_model.BranchStock{},
		&transaction_model.Transaction{},
		&transaction_model.TransactionItem{},
		&promo_model.Promo{},
		&settlement_model.Settlement{},
	)
	if err != nil {
		log.Fatal("Migrasi gagal:", err)
	}

	// Preserve existing inventory when upgrading a database without reset metadata.
	if err := database.Exec(`UPDATE branch_stocks SET initial_stock = stock,
 reset_date = (CURRENT_TIMESTAMP AT TIME ZONE CASE WHEN branch = 'tokyo' THEN 'Asia/Tokyo' ELSE 'Asia/Jakarta' END)::date
 WHERE reset_date IS NULL`).Error; err != nil {
		log.Fatal("Migrasi stok awal gagal")
	}

	// Backfill existing menus only; conflict handling preserves stock on restarts.
	if err := database.Exec(`INSERT INTO branch_stocks (menu_id, branch, is_available, stock, initial_stock, reset_date)
 SELECT m.id, b.branch, true, 0, 0, (CURRENT_TIMESTAMP AT TIME ZONE CASE WHEN b.branch = 'tokyo' THEN 'Asia/Tokyo' ELSE 'Asia/Jakarta' END)::date FROM menus m
 CROSS JOIN (VALUES ('jakarta-selatan'), ('depok'), ('tokyo')) AS b(branch)
 WHERE m.deleted_at IS NULL
 ON CONFLICT (menu_id, branch) DO NOTHING`).Error; err != nil {
		log.Fatal("Migrasi stok cabang gagal")
	}

	log.Println("Migrasi tabel berhasil")

	DB = database
}
