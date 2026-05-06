package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sienge-transfer/models"
)

const (
	AppDirName     = "sienge-transfer"
	ConfigFileName = "config.json"
)

var (
	ErrConfigNotFound = errors.New("config.json nao encontrado")
	ErrInvalidConfig  = errors.New("config.json invalido")
)

type Store struct {
	Dir string
}

func DefaultDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, AppDirName), nil
}

func NewStore(dir string) Store {
	return Store{Dir: dir}
}

func DefaultStore() (Store, error) {
	dir, err := DefaultDir()
	if err != nil {
		return Store{}, err
	}

	return NewStore(dir), nil
}

func (s Store) EnsureDir() error {
	if strings.TrimSpace(s.Dir) == "" {
		return fmt.Errorf("%w: diretorio local nao informado", ErrInvalidConfig)
	}

	return os.MkdirAll(s.Dir, 0o700)
}

func (s Store) ConfigPath() string {
	return filepath.Join(s.Dir, ConfigFileName)
}

func (s Store) SecretKeyPath() string {
	return filepath.Join(s.Dir, SecretKeyFileName)
}

func (s Store) Exists() bool {
	_, err := os.Stat(s.ConfigPath())
	return err == nil
}

func (s Store) Save(cfg models.Config) error {
	if err := ValidatePlainConfig(cfg); err != nil {
		return err
	}

	if err := s.EnsureDir(); err != nil {
		return err
	}

	key, err := s.LoadOrCreateKey()
	if err != nil {
		return err
	}

	stored := cfg
	stored.Empresa.APISenha, err = EncryptString(cfg.Empresa.APISenha, key)
	if err != nil {
		return err
	}
	stored.Empresa.SenhaCifrada = true

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.ConfigPath(), data, 0o600)
}

func (s Store) Load() (models.Config, error) {
	data, err := os.ReadFile(s.ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return models.Config{}, ErrConfigNotFound
	}
	if err != nil {
		return models.Config{}, err
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.Config{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if err := ValidateEncryptedConfig(cfg); err != nil {
		return models.Config{}, err
	}

	key, err := s.LoadOrCreateKey()
	if err != nil {
		return models.Config{}, err
	}

	cfg.Empresa.APISenha, err = DecryptString(cfg.Empresa.APISenha, key)
	if err != nil {
		return models.Config{}, fmt.Errorf("%w: senha da API nao pode ser descriptografada", ErrInvalidConfig)
	}
	cfg.Empresa.SenhaCifrada = false

	if err := ValidatePlainConfig(cfg); err != nil {
		return models.Config{}, err
	}

	return cfg, nil
}

func (s Store) UpdateCredentials(empresa models.Empresa) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}

	cfg.Empresa = empresa
	return s.Save(cfg)
}

func ValidatePlainConfig(cfg models.Config) error {
	if strings.TrimSpace(cfg.Usuario.Nome) == "" {
		return fmt.Errorf("%w: nome do usuario obrigatorio", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Usuario.Cargo) == "" {
		return fmt.Errorf("%w: cargo do usuario obrigatorio", ErrInvalidConfig)
	}
	if err := validateEmpresa(cfg.Empresa); err != nil {
		return err
	}
	if cfg.Empresa.SenhaCifrada {
		return fmt.Errorf("%w: senha deve estar em texto claro apenas em memoria", ErrInvalidConfig)
	}
	if err := validateObras(cfg.Obras); err != nil {
		return err
	}

	return nil
}

func ValidateEncryptedConfig(cfg models.Config) error {
	if strings.TrimSpace(cfg.Usuario.Nome) == "" {
		return fmt.Errorf("%w: nome do usuario obrigatorio", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Usuario.Cargo) == "" {
		return fmt.Errorf("%w: cargo do usuario obrigatorio", ErrInvalidConfig)
	}
	if err := validateEmpresa(cfg.Empresa); err != nil {
		return err
	}
	if !cfg.Empresa.SenhaCifrada {
		return fmt.Errorf("%w: senha da API deve estar criptografada", ErrInvalidConfig)
	}
	if err := validateObras(cfg.Obras); err != nil {
		return err
	}

	return nil
}

func validateEmpresa(empresa models.Empresa) error {
	if strings.TrimSpace(empresa.Nome) == "" {
		return fmt.Errorf("%w: nome da empresa obrigatorio", ErrInvalidConfig)
	}
	if strings.TrimSpace(empresa.Subdominio) == "" {
		return fmt.Errorf("%w: subdominio da empresa obrigatorio", ErrInvalidConfig)
	}
	if strings.TrimSpace(empresa.APIUsuario) == "" {
		return fmt.Errorf("%w: usuario da API obrigatorio", ErrInvalidConfig)
	}
	if strings.TrimSpace(empresa.APISenha) == "" {
		return fmt.Errorf("%w: senha da API obrigatoria", ErrInvalidConfig)
	}

	return nil
}

func validateObras(obras []models.Obra) error {
	if len(obras) == 0 {
		return fmt.Errorf("%w: cadastre pelo menos uma obra", ErrInvalidConfig)
	}

	seen := make(map[int]bool, len(obras))
	for _, obra := range obras {
		if obra.ID <= 0 {
			return fmt.Errorf("%w: ID da obra deve ser numerico positivo", ErrInvalidConfig)
		}
		if strings.TrimSpace(obra.Nome) == "" {
			return fmt.Errorf("%w: nome da obra obrigatorio", ErrInvalidConfig)
		}
		if seen[obra.ID] {
			return fmt.Errorf("%w: ID da obra duplicado", ErrInvalidConfig)
		}
		seen[obra.ID] = true
	}

	return nil
}
