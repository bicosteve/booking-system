package app

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bicosteve/booking-system/entities"
)

func LoadConfigs(file string) (entities.Config, error) {
	var config entities.Config

	data, err := os.ReadFile(file)
	if err != nil {
		entities.MessageLogs.ErrorLog.Fatalf("could not read toml file due to %v ", err)
		return entities.Config{}, err

	}

	_, err = toml.Decode(string(data), &config)
	if err != nil {
		entities.MessageLogs.ErrorLog.Fatalf("could not load configs due to %v ", err)
		return entities.Config{}, err

	}

	return config, nil
}
