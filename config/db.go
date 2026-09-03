package config

import (
	"fmt"
	"log"
	"os"

	category_model "pos-go/models/category_model"
	menu_model "pos-go/models/menu_model"
	transaction_model "pos-go/models/transaction_model"
	user_model "pos-go/models/user_model"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	// Memuat .env jika tersedia.
	// Di Render, konfigurasi akan dibaca dari Environment Variables.
	_ = godotenv.Load()

	// Ambil environment variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Pastikan konfigurasi database sudah diisi
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Konfigurasi database belum lengkap")
	}

	// Supabase membutuhkan koneksi SSL
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=require",
		host,
		user,
		password,
		dbname,
		port,
	)

	// Koneksi ke database
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke database: ", err)
	}

	log.Println("Berhasil terhubung ke database PostgreSQL")

	// AutoMigrate tabel
	err = database.AutoMigrate(
		&user_model.User{},
		&category_model.Category{},
		&menu_model.Menu{},
		&transaction_model.Transaction{},
		&transaction_model.TransactionItem{},
	)
	if err != nil {
		log.Fatal("Migrasi gagal: ", err)
	}

	log.Println("Migrasi tabel berhasil")

	DB = database
}