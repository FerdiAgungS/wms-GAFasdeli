package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// DB adalah variabel global untuk koneksi database
var DB *sql.DB

// ConnectDB melakukan inisialisasi koneksi ke database MySQL Aiven
func ConnectDB() {
	// Mengambil konfigurasi dari Environment Variables (Railway)
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// Validasi: Pastikan variabel tidak kosong
	if dbUser == "" || dbHost == "" || dbName == "" {
		log.Fatal("Error: Variabel lingkungan database tidak lengkap. Pastikan DB_USER, DB_HOST, dan DB_NAME sudah diatur di Railway.")
	}

	// Membuat DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser,
		dbPass,
		dbHost,
		dbPort,
		dbName,
	)

	// Membuka koneksi
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}

	// Cek koneksi ke database
	err = db.Ping()
	if err != nil {
		log.Fatalf("Gagal melakukan ping ke database: %v", err)
	}

	fmt.Println("Database berhasil terkoneksi!")

	DB = db
}
