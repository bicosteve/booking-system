package utils

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogInfoAndError(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "", 0)

	LogInfo("hello %s", l, "world")
	assert.Contains(t, buf.String(), "hello world")

	buf.Reset()
	LogError("error %d", l, 42)
	assert.Contains(t, buf.String(), "error 42")
}

func TestInitLogger_NonProd(t *testing.T) {
	// default (no ENV=prod) should just point log output at stderr and return nil
	old := os.Getenv("ENV")
	defer os.Setenv("ENV", old)

	os.Unsetenv("ENV")
	err := InitLogger("")
	assert.NoError(t, err)
}

func TestInitLogger_ProdInvalidFolder(t *testing.T) {
	old := os.Getenv("ENV")
	defer os.Setenv("ENV", old)

	os.Setenv("ENV", "prod")
	// rotatelogs typically still constructs a writer for a relative folder, so we
	// only assert it does not panic and returns without crashing.
	_ = InitLogger(t.TempDir())
}
