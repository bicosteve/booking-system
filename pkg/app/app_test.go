package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleTOML = `
[app]
developer = ["dev@example.com"]
enable = true
id = "booking-system.api"
version = "1.0"

[logger]
handler = "json"
level = "debug"
path = "./logs"
writer = "both"

[[mysql]]
host = "127.0.0.1"
name = "mysql"
password = "pass"
port = 3306
schema = "bookings"
username = "user"

[[redis]]
address = "127.0.0.1"
database = 0
name = "redis"
password = ""
port = "6379"

[[kafka]]
broker = "localhost:19092"
name = "kafka"
topics = ["payment_one", "payment_two"]

[[secrets]]
name = "secrets"
jwt = "supersecret"
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestLoadConfigs_Valid(t *testing.T) {
	path := writeTemp(t, sampleTOML)

	config, err := LoadConfigs(path)
	assert.NoError(t, err)

	assert.Equal(t, "booking-system.api", config.App.Id)
	assert.Equal(t, "1.0", config.App.Version)
	assert.True(t, config.App.Enable)

	assert.Len(t, config.Mysql, 1)
	assert.Equal(t, "127.0.0.1", config.Mysql[0].Host)
	assert.Equal(t, 3306, config.Mysql[0].Port)

	assert.Len(t, config.Redis, 1)
	assert.Equal(t, "6379", config.Redis[0].Port)

	assert.Len(t, config.Kafka, 1)
	assert.Equal(t, []string{"payment_one", "payment_two"}, config.Kafka[0].Topics)

	assert.Len(t, config.Secrets, 1)
	assert.Equal(t, "supersecret", config.Secrets[0].JWT)
}

func TestLoadConfigs_FileNotFound(t *testing.T) {
	config, err := LoadConfigs("/path/does/not/exist.toml")
	assert.Error(t, err)
	assert.Equal(t, "booking-system.api", "booking-system.api") // sanity
	assert.Empty(t, config.App.Id)
}

func TestLoadConfigs_InvalidTOML(t *testing.T) {
	path := writeTemp(t, "this is = = not valid toml [[[")

	config, err := LoadConfigs(path)
	assert.Error(t, err)
	assert.Empty(t, config.App.Id)
}
