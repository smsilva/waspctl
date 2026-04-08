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

func Get(key string) (string, error) {
	if !viper.IsSet(key) {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return viper.GetString(key), nil
}

func Set(key, value string) error {
	viper.Set(key, value)
	if err := viper.WriteConfig(); err != nil {
		if err2 := os.MkdirAll(filepath.Dir(viper.ConfigFileUsed()), 0o750); err2 != nil {
			return err2
		}
		return viper.SafeWriteConfig()
	}
	return nil
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
