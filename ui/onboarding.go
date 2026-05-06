package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sienge-transfer/config"
	"sienge-transfer/models"
)

type ConfigStore interface {
	Load() (models.Config, error)
	Save(models.Config) error
	UpdateCredentials(models.Empresa) error
}

type CredentialValidator interface {
	ValidateCredentials(context.Context, models.Empresa) error
}

type OnboardingService struct {
	Store     ConfigStore
	Validator CredentialValidator
}

type CredentialsInput struct {
	EmpresaNome string
	Subdominio  string
	APIUsuario  string
	APISenha    string
}

type UserInput struct {
	Nome  string
	Cargo string
}

type WorksInput struct {
	Obras []models.Obra
}

type CompleteOnboardingInput struct {
	Credentials CredentialsInput
	User        UserInput
	Works       WorksInput
}

type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Errors, "; ")
}

func NeedsOnboarding(store ConfigStore) (bool, error) {
	_, err := store.Load()
	if err == nil {
		return false, nil
	}
	if errors.Is(err, config.ErrConfigNotFound) || errors.Is(err, config.ErrInvalidConfig) {
		return true, nil
	}

	return false, err
}

func (s OnboardingService) Complete(ctx context.Context, input CompleteOnboardingInput) (models.Config, error) {
	if s.Store == nil {
		return models.Config{}, errors.New("armazenamento de configuracao nao informado")
	}
	if s.Validator == nil {
		return models.Config{}, errors.New("validador de credenciais nao informado")
	}

	empresa, err := ValidateCredentialsInput(input.Credentials)
	if err != nil {
		return models.Config{}, err
	}
	usuario, err := ValidateUserInput(input.User)
	if err != nil {
		return models.Config{}, err
	}
	obras, err := ValidateWorksInput(input.Works)
	if err != nil {
		return models.Config{}, err
	}

	if err := s.Validator.ValidateCredentials(ctx, empresa); err != nil {
		return models.Config{}, fmt.Errorf("credenciais da API nao validadas: %w", err)
	}

	cfg := models.Config{
		Usuario: usuario,
		Empresa: empresa,
		Obras:   obras,
	}
	if err := s.Store.Save(cfg); err != nil {
		return models.Config{}, err
	}

	return cfg, nil
}

func (s OnboardingService) UpdateCredentials(ctx context.Context, input CredentialsInput) (models.Empresa, error) {
	if s.Store == nil {
		return models.Empresa{}, errors.New("armazenamento de configuracao nao informado")
	}
	if s.Validator == nil {
		return models.Empresa{}, errors.New("validador de credenciais nao informado")
	}

	empresa, err := ValidateCredentialsInput(input)
	if err != nil {
		return models.Empresa{}, err
	}
	if err := s.Validator.ValidateCredentials(ctx, empresa); err != nil {
		return models.Empresa{}, fmt.Errorf("credenciais da API nao validadas: %w", err)
	}
	if err := s.Store.UpdateCredentials(empresa); err != nil {
		return models.Empresa{}, err
	}

	return empresa, nil
}

func ValidateCredentialsInput(input CredentialsInput) (models.Empresa, error) {
	empresa := models.Empresa{
		Nome:       strings.TrimSpace(input.EmpresaNome),
		Subdominio: normalizeSubdomain(input.Subdominio),
		APIUsuario: strings.TrimSpace(input.APIUsuario),
		APISenha:   strings.TrimSpace(input.APISenha),
	}

	var validationErrors []string
	if empresa.Nome == "" {
		validationErrors = append(validationErrors, "nome da empresa obrigatorio")
	}
	if empresa.Subdominio == "" {
		validationErrors = append(validationErrors, "subdominio da empresa obrigatorio")
	}
	if empresa.APIUsuario == "" {
		validationErrors = append(validationErrors, "usuario da API obrigatorio")
	}
	if empresa.APISenha == "" {
		validationErrors = append(validationErrors, "senha da API obrigatoria")
	}

	if len(validationErrors) > 0 {
		return models.Empresa{}, &ValidationError{Errors: validationErrors}
	}

	return empresa, nil
}

func ValidateUserInput(input UserInput) (models.Usuario, error) {
	usuario := models.Usuario{
		Nome:  strings.TrimSpace(input.Nome),
		Cargo: strings.TrimSpace(input.Cargo),
	}

	var validationErrors []string
	if usuario.Nome == "" {
		validationErrors = append(validationErrors, "nome completo do usuario obrigatorio")
	}
	if usuario.Cargo == "" {
		validationErrors = append(validationErrors, "cargo do usuario obrigatorio")
	}

	if len(validationErrors) > 0 {
		return models.Usuario{}, &ValidationError{Errors: validationErrors}
	}

	return usuario, nil
}

func ValidateWorksInput(input WorksInput) ([]models.Obra, error) {
	var validationErrors []string
	seen := make(map[int]bool, len(input.Obras))
	obras := make([]models.Obra, 0, len(input.Obras))

	for _, obra := range input.Obras {
		obra.Nome = strings.TrimSpace(obra.Nome)
		if obra.ID <= 0 {
			validationErrors = append(validationErrors, "ID da obra deve ser numerico positivo")
		}
		if obra.Nome == "" {
			validationErrors = append(validationErrors, "nome da obra obrigatorio")
		}
		if obra.ID > 0 && seen[obra.ID] {
			validationErrors = append(validationErrors, "ID da obra duplicado")
		}
		seen[obra.ID] = true
		obras = append(obras, obra)
	}

	if len(obras) == 0 {
		validationErrors = append(validationErrors, "cadastre pelo menos uma obra")
	}
	if len(validationErrors) > 0 {
		return nil, &ValidationError{Errors: validationErrors}
	}

	return obras, nil
}

func normalizeSubdomain(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimSuffix(value, "/")
	value = strings.TrimSuffix(value, ".sienge.com.br")
	return strings.TrimSpace(value)
}
