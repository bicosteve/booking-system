package entities

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Password    string    `json:"password"`
	IsSeller    bool      `json:"is_seller"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Config struct {
	App    AppConfig     `toml:"app"`
	Logger LoggerConfig  `toml:"logger"`
	Notify NotifyConfig  `toml:"notify"`
	Http   []HttpConfig  `toml:"http"`
	Mysql  []MysqlConfig `toml:"mysql"`
	Redis  []RedisConfig `toml:"redis"`
	Kafka  []KakfaConfig `toml:"kafka"`
}

type AppConfig struct {
	Id        string   `toml:"id"`
	Version   string   `toml:"version"`
	Enable    bool     `toml:"enable"`
	Developer []string `toml:"developer"`
	Args      args
}

type LoggerConfig struct {
	Writer  string `toml:"writer"`
	Level   string `toml:"level"`
	Path    string `toml:"path"`
	File    string `toml:"file"`
	Handler string `toml:"handler"`
}

type NotifyConfig struct {
	Preferences     []PrefConfig `toml:"preference"`
	EmailFrom       string       `toml:"email_from"`
	RetryMax        int          `toml:"retry_max"`
	RetryBackOff    int          `toml:"retry_backoff"`
	SmsUrl          string       `toml:"sms_url"`
	SmsClientId     string       `toml:"sms_client_id"`
	SmsClientSecret string       `toml:"sms_client_secret"`
}

type PrefConfig struct {
	Event   string   `toml:"event"`
	Channel string   `toml:"channel"`
	EmailTo []string `toml:"email"`
	SmsTo   []string `toml:"sms"`
}

type HttpConfig struct {
	Name string `toml:"name"`
	Host string `toml:"host"`
	Port int    `toml:"port"`
	Path string `toml:"path"`
	Cors struct {
		AllowedMethod []string `toml:"allowed_method"`
		AllowedHeader []string `toml:"allowed_header"`
		AllowedOrigin []string `toml:"allowed_origin"`
	} `toml:"cors"`
	Args args `toml:"args"`
}

type MysqlConfig struct {
	Name     string `toml:"name"`
	Schema   string `toml:"schema"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Args     args   `toml:"args"`
}

type RedisConfig struct {
	Name              string   `toml:"name"`
	Address           string   `toml:"address"`
	Database          int      `toml:"database"`
	Username          string   `toml:"username"`
	Password          string   `toml:"password"`
	Nodes             []string `toml:"nodes"`
	SentinelAddresses []string `toml:"sentinel_addresses"`
	Args              args     `toml:"args"`
}

type KakfaConfig struct {
	Name      string      `toml:"name"`
	Broker    string      `toml:"broker"`
	Topic     string      `toml:"topic"`
	Key       string      `toml:"key"`
	Data      interface{} `data:"data"`
	Consumers string      `toml:"consumers"`
	Producers int         `toml:"producers"`
}

type UserPayload struct {
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type Envelope map[string]interface{}

type Message struct {
	InfoLog  *log.Logger
	ErrorLog *log.Logger
}

type JSONResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type SerializedUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type args map[string]interface{}

var infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
var errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
var EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

var MessageLogs = &Message{
	InfoLog:  infoLog,
	ErrorLog: errorLog,
}

var ErrNoRecord = errors.New("models: no matching record found")
var ErrDuplicateEmail = errors.New("models: user already exists")
var ErrorInvalidCredentials = errors.New("models: incorrect password or email")
var ErrorDBConnection = errors.New("db: could not connect db becacuse ")
var ErrorDBPing = errors.New("db: could not ping db because ")
var SuccessDBPing = "db: successfully connected to db"

func (c Config) LoadConfigs(file string) (Config, error) {
	var config Config

	data, err := os.ReadFile(file)
	if err != nil {
		MessageLogs.ErrorLog.Fatalf("could not read toml file due to %v ", err)

	}

	_, err = toml.Decode(string(data), &config)
	if err != nil {
		MessageLogs.ErrorLog.Fatalf("could not load configs due to %v ", err)

	}

	return config, nil
}

func (c Config) FindHttpConfig(name string) (http HttpConfig, err error) {
	for _, config := range c.Http {
		if config.Name == name {
			return config, nil
		}
	}

	return http, fmt.Errorf("no http config found for name '%v' ", err)
}

func (c Config) FindMysqlConfig(name string) (mysql MysqlConfig, err error) {
	return MysqlConfig{}, nil
}

func (c Config) FindRedisConfig(name string) (mysql MysqlConfig, err error) {
	return MysqlConfig{}, nil
}

func (c Config) FindKafkaConfig(name string) (mysql MysqlConfig, err error) {
	return MysqlConfig{}, nil
}
