package utils

import (
	"database/sql"
	"time"

	"github.com/bicosteve/booking-system/entities"
	_ "github.com/go-sql-driver/mysql"
)

func DatabaseConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		LogError(entities.ErrorDBPing.Error()+err.Error(), entities.ErrorLog)
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	LogInfo(entities.SuccessDBPing, entities.InfoLog)
	return db, nil
}
