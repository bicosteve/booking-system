package connections

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/pkg/entities"
)

type Mysqldb struct {
	Connection *sql.DB
	ctx        context.Context
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

	db := Mysqldb{Connection: connection, ctx: ctx, Config: config}

	go db.close()

	return db, nil

}

func (m Mysqldb) close() error {
	<-m.ctx.Done()
	return m.Connection.Close()
}
