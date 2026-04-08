package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"
)

var ErrKeyNotFound = errors.New("key not found")

type Pair struct {
	Key   string
	Value string
}

// configPath armazena o caminho do arquivo de configuração definido em initConfig
var configPath string

// SetConfigPath define o caminho do arquivo de configuração (chamado por initConfig)
func SetConfigPath(path string) {
	configPath = path
}

func Get(key string) (string, error) {
	if !viper.IsSet(key) {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return viper.GetString(key), nil
}

func Set(key, value string) error {
	viper.Set(key, value)

	if configPath == "" {
		return fmt.Errorf("config file path not set")
	}

	// Verifica se o arquivo existe
	_, err := os.Stat(configPath)
	fileExists := err == nil

	if fileExists {
		// Arquivo existe: WriteConfigAs sobrescreve
		return viper.WriteConfigAs(configPath)
	}

	// Arquivo não existe: precisa criar diretório primeiro
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		return err
	}
	// SafeWriteConfigAs só escreve se o arquivo não existir
	return viper.SafeWriteConfigAs(configPath)
}

func List() []Pair {
	all := viper.AllSettings()
	pairs := make([]Pair, 0, len(all))
	for k, v := range all {
		pairs = append(pairs, Pair{Key: k, Value: fmt.Sprintf("%v", v)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key < pairs[j].Key
	})
	return pairs
}
