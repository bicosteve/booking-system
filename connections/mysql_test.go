package connections

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

var (
	host     = "127.0.0.1"
	port     = 3306
	username = "bico"
	password = "1234"
	schema   = "bookings"
)

func TestDatabaseConnection(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local", username, password, host, port, schema)

	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatalf("error %s occured", err)
	}

	defer db.Close()

	result, err := DatabaseConnection(dsn)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("there unfulfilled expectations: %s", err)
	}

}
