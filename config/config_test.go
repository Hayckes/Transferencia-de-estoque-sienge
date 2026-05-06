package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestDefaultDirUsesAppDirName(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir() error = %v", err)
	}

	if filepath.Base(dir) != AppDirName {
		t.Fatalf("DefaultDir() base = %q, want %q", filepath.Base(dir), AppDirName)
	}
}

func TestLoadOrCreateKeyCreatesAndReuses32ByteKey(t *testing.T) {
	store := NewStore(t.TempDir())

	key, err := store.LoadOrCreateKey()
	if err != nil {
		t.Fatalf("LoadOrCreateKey() error = %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}

	reused, err := store.LoadOrCreateKey()
	if err != nil {
		t.Fatalf("LoadOrCreateKey() second call error = %v", err)
	}
	if string(reused) != string(key) {
		t.Fatal("LoadOrCreateKey() did not reuse existing key")
	}

	info, err := os.Stat(store.SecretKeyPath())
	if err != nil {
		t.Fatalf("Stat(secret.key) error = %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("secret.key mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestLoadOrCreateKeyRejectsInvalidKeySize(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if err := os.WriteFile(store.SecretKeyPath(), []byte("curta"), 0o600); err != nil {
		t.Fatalf("WriteFile(secret.key) error = %v", err)
	}

	_, err := store.LoadOrCreateKey()
	if !errors.Is(err, ErrInvalidSecretKey) {
		t.Fatalf("LoadOrCreateKey() error = %v, want ErrInvalidSecretKey", err)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir())
	key, err := store.LoadOrCreateKey()
	if err != nil {
		t.Fatalf("LoadOrCreateKey() error = %v", err)
	}

	encrypted, err := EncryptString("senha-secreta", key)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}
	if encrypted == "senha-secreta" {
		t.Fatal("EncryptString() returned plaintext")
	}
	if !strings.HasPrefix(encrypted, cipherPrefix) {
		t.Fatalf("EncryptString() = %q, want prefix %q", encrypted, cipherPrefix)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}
	if decrypted != "senha-secreta" {
		t.Fatalf("DecryptString() = %q, want plaintext", decrypted)
	}
}

func TestDecryptStringRejectsWrongKey(t *testing.T) {
	store := NewStore(t.TempDir())
	key, err := store.LoadOrCreateKey()
	if err != nil {
		t.Fatalf("LoadOrCreateKey() error = %v", err)
	}
	encrypted, err := EncryptString("senha-secreta", key)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	wrongKey := make([]byte, 32)
	wrongKey[0] = 1
	_, err = DecryptString(encrypted, wrongKey)
	if err == nil {
		t.Fatal("DecryptString() error = nil, want error")
	}
}

func TestDecryptStringRejectsInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)

	if _, err := DecryptString("invalido", key); err == nil {
		t.Fatal("DecryptString() without prefix error = nil, want error")
	}
	if _, err := DecryptString(cipherPrefix+"abc", key); err == nil {
		t.Fatal("DecryptString() invalid payload error = nil, want error")
	}
}

func TestSaveAndLoadConfigEncryptsPassword(t *testing.T) {
	store := NewStore(t.TempDir())
	cfg := validConfig()

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(store.ConfigPath())
	if err != nil {
		t.Fatalf("ReadFile(config.json) error = %v", err)
	}
	if strings.Contains(string(raw), "senha-secreta") {
		t.Fatalf("config.json contains plaintext password: %s", string(raw))
	}

	var stored models.Config
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("json.Unmarshal(stored config) error = %v", err)
	}
	if !stored.Empresa.SenhaCifrada {
		t.Fatal("stored config should mark password as encrypted")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Empresa.APISenha != "senha-secreta" {
		t.Fatalf("loaded password = %q, want plaintext in memory", loaded.Empresa.APISenha)
	}
	if loaded.Empresa.SenhaCifrada {
		t.Fatal("loaded config should keep decrypted password in memory")
	}
}

func TestLoadDetectsMissingAndCorruptedConfig(t *testing.T) {
	store := NewStore(t.TempDir())

	_, err := store.Load()
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("Load() missing error = %v, want ErrConfigNotFound", err)
	}

	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if err := os.WriteFile(store.ConfigPath(), []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}

	_, err = store.Load()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load() corrupted error = %v, want ErrInvalidConfig", err)
	}
}

func TestValidatePlainConfigRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.Config)
	}{
		{name: "missing user name", mutate: func(c *models.Config) { c.Usuario.Nome = "" }},
		{name: "missing user role", mutate: func(c *models.Config) { c.Usuario.Cargo = "" }},
		{name: "missing company name", mutate: func(c *models.Config) { c.Empresa.Nome = "" }},
		{name: "missing subdomain", mutate: func(c *models.Config) { c.Empresa.Subdominio = "" }},
		{name: "missing api user", mutate: func(c *models.Config) { c.Empresa.APIUsuario = "" }},
		{name: "missing api password", mutate: func(c *models.Config) { c.Empresa.APISenha = "" }},
		{name: "encrypted password in memory", mutate: func(c *models.Config) { c.Empresa.SenhaCifrada = true }},
		{name: "no buildings", mutate: func(c *models.Config) { c.Obras = nil }},
		{name: "invalid building id", mutate: func(c *models.Config) { c.Obras[0].ID = 0 }},
		{name: "missing building name", mutate: func(c *models.Config) { c.Obras[0].Nome = "" }},
		{name: "duplicated building", mutate: func(c *models.Config) { c.Obras = append(c.Obras, c.Obras[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			if err := ValidatePlainConfig(cfg); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("ValidatePlainConfig() error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestUpdateCredentialsPreservesUserAndBuildings(t *testing.T) {
	store := NewStore(t.TempDir())
	cfg := validConfig()
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	newEmpresa := models.Empresa{
		Nome:       "Nova Empresa",
		Subdominio: "novaempresa",
		APIUsuario: "novo.usuario",
		APISenha:   "nova-senha",
	}
	if err := store.UpdateCredentials(newEmpresa); err != nil {
		t.Fatalf("UpdateCredentials() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Usuario != cfg.Usuario {
		t.Fatalf("Usuario = %#v, want %#v", loaded.Usuario, cfg.Usuario)
	}
	if len(loaded.Obras) != len(cfg.Obras) || loaded.Obras[0] != cfg.Obras[0] {
		t.Fatalf("Obras = %#v, want %#v", loaded.Obras, cfg.Obras)
	}
	if loaded.Empresa.Nome != newEmpresa.Nome || loaded.Empresa.APISenha != newEmpresa.APISenha {
		t.Fatalf("Empresa = %#v, want updated credentials", loaded.Empresa)
	}
}

func validConfig() models.Config {
	return models.Config{
		Usuario: models.Usuario{Nome: "Joao Silva", Cargo: "Engenheiro"},
		Empresa: models.Empresa{
			Nome:       "Construtora XYZ",
			Subdominio: "construtoraxyz",
			APIUsuario: "joao.silva",
			APISenha:   "senha-secreta",
		},
		Obras: []models.Obra{{ID: 121, Nome: "Residencial Novo Horizonte"}},
	}
}
