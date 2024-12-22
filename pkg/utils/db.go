package utils

import (
	"database/sql"
	"time"

	"github.com/bicosteve/booking-system/pkg/entities"
	_ "github.com/go-sql-driver/mysql"
)

func DbConnect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = db.Ping()
	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("%s %s", entities.ErrorDBPing.Error(), err)
		return nil, err
	}

	entities.MessageLogs.InfoLog.Printf("%s", entities.SuccessDBPing)
	return db, nil
}
