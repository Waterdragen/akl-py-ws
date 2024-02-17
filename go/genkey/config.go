package genkey

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func ReadWeights(config *UserConfig) {
	b, err := GenkeyReadFile("config.toml")
	if err != nil {
		panic(fmt.Sprintf("There was an issue reading the config file.\n%v", err))
	}

	_, err = toml.Decode(string(b), config)
	if err != nil {
		panic("Toml decoding error")
	}

	if !fileExists(filepath.Join("./genkey", config.Paths.Corpora, config.Corpus) + ".json") {
		panic(fmt.Sprintf("Invalid config: Corpus [%s] does not exist.\n", config.Corpus))
	}

	if config.Generation.Selection > config.Generation.InitialPopulation {
		panic("Invalid config: Generation.Selection cannot be greater than Generation.InitialPopulation.")
	}
}
