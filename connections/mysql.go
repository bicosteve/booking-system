package connections

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/entities"
)

type Mysqldb struct {
	Connection *sql.DB
	CTX        context.Context
	Config     entities.MysqlConfig
}

func ConnectSQLDB(ctx context.Context, config entities.MysqlConfig) (Mysqldb, error) {

	connection, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local", config.Username, config.Password, config.Host, config.Port, config.Schema))
	if err != nil {
		return Mysqldb{}, err
	}

	connection.SetMaxOpenConns(10)
	connection.SetMaxIdleConns(10)
	connection.SetConnMaxLifetime(5 * time.Second)
	connection.SetConnMaxIdleTime(5 * time.Second)

	db := &Mysqldb{Connection: connection, CTX: ctx, Config: config}

	// defer connection.Close()

	entities.MessageLogs.InfoLog.Printf("%s", entities.SuccessDBPing)

	return *db, nil

}
