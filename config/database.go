package config

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {

	dsn := "root:Erenezel12!@tcp(127.0.0.1:3306)/WMS_GAFasdeli"

	db, err := sql.Open("mysql", dsn)

	if err != nil {
		panic(err)
	}

	err = db.Ping()

	if err != nil {
		panic(err)
	}

	fmt.Println("Database Connected!")

	DB = db
}
