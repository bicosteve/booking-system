package connections

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	_ "github.com/go-sql-driver/mysql"
)

func DatabaseConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {

		utils.LogError(fmt.Sprintf("%s %s", entities.ErrorDBPing.Error(), err), entities.ErrorLog)
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(time.Second * 60)

	utils.LogInfo(fmt.Sprintf("%s", entities.SuccessDBPing), entities.InfoLog)
	return db, nil
}
