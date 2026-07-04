package connections

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

// DatabaseConnection opens the pool and immediately pings. Since these tests do
// not have a live MySQL server available, we assert on the error paths that do
// not require a reachable database.

func TestDatabaseConnection_InvalidDSN(t *testing.T) {
	// A malformed DSN causes sql.Open (or the first Ping) to fail.
	_, err := DatabaseConnection("not-a-valid-dsn")
	assert.Error(t, err)
}

func TestDatabaseConnection_UnreachableHost(t *testing.T) {
	// A well-formed DSN pointing at a port where nothing is listening should
	// fail on Ping.
	dsn := "user:pass@tcp(127.0.0.1:1)/schema?charset=latin1&parseTime=True&loc=Local"
	db, err := DatabaseConnection(dsn)
	assert.Error(t, err)
	assert.Nil(t, db)
}
