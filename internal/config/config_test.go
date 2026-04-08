package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smsilva/waspctl/internal/config"
	"github.com/spf13/viper"
)

func setupTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	viper.Reset()
	config.SetConfigPath(path)
	viper.SetConfigFile(path)
	return path
}

func TestSetAndGet(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"simple string", "provider", "aws"},
		{"region", "region", "us-east-1"},
		{"domain", "domain", "wasp.silvios.me"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupTempConfig(t)

			if err := config.Set(tc.key, tc.value); err != nil {
				t.Fatalf("Set(%q, %q) error: %v", tc.key, tc.value, err)
			}

			got, err := config.Get(tc.key)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tc.key, err)
			}
			if got != tc.value {
				t.Errorf("Get(%q) = %q, want %q", tc.key, got, tc.value)
			}
		})
	}
}

func TestGetMissingKey(t *testing.T) {
	setupTempConfig(t)
	_, err := config.Get("nonexistent")
	if !errors.Is(err, config.ErrKeyNotFound) {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestSetCreatesFile(t *testing.T) {
	path := setupTempConfig(t)
	// arquivo ainda não existe
	if err := config.Set("key", "val"); err != nil {
		t.Fatalf("Set em arquivo inexistente falhou: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file não foi criado: %v", err)
	}
}

func TestSetCreatesFileWithExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")
	viper.Reset()
	// Simula primeira execução: só chama SetConfigPath, sem viper.SetConfigFile
	config.SetConfigPath(path)

	if err := config.Set("provider", "aws"); err != nil {
		t.Fatalf("Set com path explícito falhou: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file não foi criado em %s: %v", path, err)
	}

	// Verifica que o valor foi salvo corretamente
	viper.SetConfigFile(path)
	_ = viper.ReadInConfig()
	got, err := config.Get("provider")
	if err != nil {
		t.Fatalf("Get após Set falhou: %v", err)
	}
	if got != "aws" {
		t.Errorf("Get(provider) = %q, want %q", got, "aws")
	}
}

func TestListSorted(t *testing.T) {
	setupTempConfig(t)
	_ = config.Set("zebra", "z")
	_ = config.Set("apple", "a")
	_ = config.Set("mango", "m")

	pairs := config.List()
	want := []string{"apple", "mango", "zebra"}
	if len(pairs) != len(want) {
		t.Fatalf("List() retornou %d itens, esperado %d", len(pairs), len(want))
	}
	for i, p := range pairs {
		if p.Key != want[i] {
			t.Errorf("pairs[%d].Key = %q, want %q", i, p.Key, want[i])
		}
	}
}

func TestListEmpty(t *testing.T) {
	setupTempConfig(t)
	pairs := config.List()
	if len(pairs) != 0 {
		t.Errorf("List() com config vazio retornou %d itens, esperado 0", len(pairs))
	}
}

func TestSetPreservesExistingKeys(t *testing.T) {
	setupTempConfig(t)
	_ = config.Set("provider", "aws")
	_ = config.Set("region", "us-east-1")

	val, err := config.Get("provider")
	if err != nil {
		t.Fatalf("Get(provider) após segundo Set: %v", err)
	}
	if val != "aws" {
		t.Errorf("provider sobrescrito: got %q, want %q", val, "aws")
	}
}
