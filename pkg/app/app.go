package app

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

func LoadConfigs(file string) (entities.Config, error) {
	var config entities.Config

	data, err := os.ReadFile(file)
	if err != nil {
		utils.LogError("Could not read toml file due to "+err.Error(), entities.ErrorLog)
		return entities.Config{}, err

	}

	_, err = toml.Decode(string(data), &config)
	if err != nil {
		utils.LogError("Error logging config because of "+err.Error(), entities.ErrorLog)
		return entities.Config{}, err

	}

	return config, nil
}
