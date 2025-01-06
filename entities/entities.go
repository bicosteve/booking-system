package entities

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"time"
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
	Name      string `toml:"name"`
	Host      string `toml:"host"`
	Port      int    `toml:"port"`
	AdminPort int    `toml:"adminport"`
	Path      string `toml:"path"`
	Cors      struct {
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
	Name     string `toml:"name"`
	Address  string `toml:"address"`
	Password string `toml:"password"`
	Port     string `toml:"port"`
	Database int    `toml:"database"`
}

type KakfaConfig struct {
	Name   string `toml:"name"`
	Broker string `toml:"broker"`
	Topic  string `toml:"topic"`
	Key    string `toml:"key"`
}

type UserPayload struct {
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	IsVendor        string `json:"is_vendor"`
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

var ErrNoRecord = errors.New("MODELS: no matching record found")
var ErrDuplicateEmail = errors.New("MODELS: user already exists")
var ErrorInvalidCredentials = errors.New("MODELS: incorrect password or email")
var ErrorDBConnection = errors.New("DB: could not connect db becacuse ")
var ErrorDBPing = errors.New("DB: could not ping db because ")
var SuccessDBPing = "MYSQL: successfully connected to db"
var CTX = context.Background()
