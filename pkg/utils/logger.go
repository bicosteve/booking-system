package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	rotateLogs "github.com/lestrrat-go/file-rotatelogs"
)

func InitLogger(logFolder string) error {
	// Check if the environment is not prod then use terminal
	if os.Getenv("ENV") != "prod" {
		log.SetOutput(os.Stderr)
		return nil
	}

	writer, err := rotateLogs.New(
		fmt.Sprintf("%s/app-%s.log", logFolder, "%Y-%m-%d"),
		rotateLogs.WithLinkName(logFolder+"link"),
		rotateLogs.WithRotationTime(time.Hour*24),
	)

	if err != nil {
		fmt.Printf("unable to initialize writer, logging on stderr")
		log.SetOutput(os.Stderr)
		return err
	}

	log.SetOutput(writer)
	return nil
}

func logger(msg string, level *log.Logger, params ...any) {
	message := fmt.Sprintf(msg, params...)
	level.Printf("%s", message)

}

func LogInfo(msg string, inf *log.Logger, params ...any) {
	logger(msg, inf, params...)
}

func LogError(msg string, err *log.Logger, params ...any) {
	logger(msg, err, params...)
}
