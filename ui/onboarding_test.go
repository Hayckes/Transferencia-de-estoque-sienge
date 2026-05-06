package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sienge-transfer/config"
	"sienge-transfer/models"
)

func TestNeedsOnboarding(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "valid config", err: nil, want: false},
		{name: "missing config", err: config.ErrConfigNotFound, want: true},
		{name: "invalid config", err: config.ErrInvalidConfig, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeConfigStore{loadErr: tt.err}
			got, err := NeedsOnboarding(store)
			if err != nil {
				t.Fatalf("NeedsOnboarding() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NeedsOnboarding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsOnboardingReturnsUnexpectedErrors(t *testing.T) {
	wantErr := errors.New("erro de disco")
	_, err := NeedsOnboarding(&fakeConfigStore{loadErr: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("NeedsOnboarding() error = %v, want %v", err, wantErr)
	}
}

func TestValidateCredentialsInput(t *testing.T) {
	empresa, err := ValidateCredentialsInput(CredentialsInput{
		EmpresaNome: " Construtora XYZ ",
		Subdominio:  "https://MinhaEmpresa.sienge.com.br/",
		APIUsuario:  " joao.silva ",
		APISenha:    " senha ",
	})
	if err != nil {
		t.Fatalf("ValidateCredentialsInput() error = %v", err)
	}

	want := models.Empresa{Nome: "Construtora XYZ", Subdominio: "minhaempresa", APIUsuario: "joao.silva", APISenha: "senha"}
	if empresa != want {
		t.Fatalf("empresa = %#v, want %#v", empresa, want)
	}
}

func TestValidateCredentialsInputRejectsRequiredFields(t *testing.T) {
	_, err := ValidateCredentialsInput(CredentialsInput{})
	if err == nil {
		t.Fatal("ValidateCredentialsInput() error = nil, want error")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	for _, want := range []string{"nome da empresa", "subdominio", "usuario da API", "senha da API"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want containing %q", err.Error(), want)
		}
	}
}

func TestValidateUserInput(t *testing.T) {
	usuario, err := ValidateUserInput(UserInput{Nome: " Joao Silva ", Cargo: " Engenheiro "})
	if err != nil {
		t.Fatalf("ValidateUserInput() error = %v", err)
	}

	want := models.Usuario{Nome: "Joao Silva", Cargo: "Engenheiro"}
	if usuario != want {
		t.Fatalf("usuario = %#v, want %#v", usuario, want)
	}
}

func TestValidateUserInputRejectsRequiredFields(t *testing.T) {
	_, err := ValidateUserInput(UserInput{})
	if err == nil {
		t.Fatal("ValidateUserInput() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "nome completo") || !strings.Contains(err.Error(), "cargo") {
		t.Fatalf("error = %q, want user field errors", err.Error())
	}
}

func TestValidateWorksInput(t *testing.T) {
	obras, err := ValidateWorksInput(WorksInput{Obras: []models.Obra{{ID: 121, Nome: " Residencial "}, {ID: 205, Nome: "Comercial"}}})
	if err != nil {
		t.Fatalf("ValidateWorksInput() error = %v", err)
	}

	want := []models.Obra{{ID: 121, Nome: "Residencial"}, {ID: 205, Nome: "Comercial"}}
	if !reflect.DeepEqual(obras, want) {
		t.Fatalf("obras = %#v, want %#v", obras, want)
	}
}

func TestValidateWorksInputRejectsInvalidWorks(t *testing.T) {
	tests := []struct {
		name  string
		input WorksInput
		want  string
	}{
		{name: "empty", input: WorksInput{}, want: "cadastre pelo menos uma obra"},
		{name: "invalid id", input: WorksInput{Obras: []models.Obra{{ID: 0, Nome: "Obra"}}}, want: "ID da obra"},
		{name: "missing name", input: WorksInput{Obras: []models.Obra{{ID: 121}}}, want: "nome da obra"},
		{name: "duplicate id", input: WorksInput{Obras: []models.Obra{{ID: 121, Nome: "A"}, {ID: 121, Nome: "B"}}}, want: "duplicado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateWorksInput(tt.input)
			if err == nil {
				t.Fatal("ValidateWorksInput() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tt.want)
			}
		})
	}
}

func TestOnboardingServiceCompleteValidatesCredentialsAndSavesConfig(t *testing.T) {
	store := &fakeConfigStore{}
	validator := &fakeCredentialValidator{}
	service := OnboardingService{Store: store, Validator: validator}

	cfg, err := service.Complete(context.Background(), validCompleteOnboardingInput())
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !validator.called {
		t.Fatal("credential validator was not called")
	}
	if !store.saved {
		t.Fatal("config was not saved")
	}
	if cfg.Usuario.Nome != "Joao Silva" || cfg.Empresa.Subdominio != "minhaempresa" || len(cfg.Obras) != 1 {
		t.Fatalf("cfg = %#v, want completed config", cfg)
	}
	if !reflect.DeepEqual(store.savedConfig, cfg) {
		t.Fatalf("savedConfig = %#v, want returned cfg %#v", store.savedConfig, cfg)
	}
}

func TestOnboardingServiceCompleteDoesNotSaveWhenCredentialsFail(t *testing.T) {
	store := &fakeConfigStore{}
	validator := &fakeCredentialValidator{err: errors.New("401")}
	service := OnboardingService{Store: store, Validator: validator}

	_, err := service.Complete(context.Background(), validCompleteOnboardingInput())
	if err == nil {
		t.Fatal("Complete() error = nil, want error")
	}
	if store.saved {
		t.Fatal("config should not be saved when credentials validation fails")
	}
}

func TestOnboardingServiceCompleteRejectsInvalidStepBeforeValidatingCredentials(t *testing.T) {
	store := &fakeConfigStore{}
	validator := &fakeCredentialValidator{}
	service := OnboardingService{Store: store, Validator: validator}
	input := validCompleteOnboardingInput()
	input.Works.Obras = nil

	_, err := service.Complete(context.Background(), input)
	if err == nil {
		t.Fatal("Complete() error = nil, want validation error")
	}
	if validator.called {
		t.Fatal("credentials should not be validated when local input is invalid")
	}
	if store.saved {
		t.Fatal("config should not be saved when local input is invalid")
	}
}

func TestOnboardingServiceUpdateCredentialsPreservesExistingConfigThroughStore(t *testing.T) {
	store := &fakeConfigStore{}
	validator := &fakeCredentialValidator{}
	service := OnboardingService{Store: store, Validator: validator}

	empresa, err := service.UpdateCredentials(context.Background(), CredentialsInput{
		EmpresaNome: "Nova Empresa",
		Subdominio:  "novaempresa",
		APIUsuario:  "novo.usuario",
		APISenha:    "nova-senha",
	})
	if err != nil {
		t.Fatalf("UpdateCredentials() error = %v", err)
	}
	if !validator.called {
		t.Fatal("credential validator was not called")
	}
	if !store.credentialsUpdated {
		t.Fatal("credentials were not updated")
	}
	if store.updatedEmpresa != empresa {
		t.Fatalf("updatedEmpresa = %#v, want %#v", store.updatedEmpresa, empresa)
	}
}

func TestOnboardingServiceUpdateCredentialsDoesNotUpdateWhenValidationFails(t *testing.T) {
	store := &fakeConfigStore{}
	validator := &fakeCredentialValidator{err: errors.New("403")}
	service := OnboardingService{Store: store, Validator: validator}

	_, err := service.UpdateCredentials(context.Background(), validCompleteOnboardingInput().Credentials)
	if err == nil {
		t.Fatal("UpdateCredentials() error = nil, want error")
	}
	if store.credentialsUpdated {
		t.Fatal("credentials should not be updated when validation fails")
	}
}

func validCompleteOnboardingInput() CompleteOnboardingInput {
	return CompleteOnboardingInput{
		Credentials: CredentialsInput{EmpresaNome: "Construtora XYZ", Subdominio: "MinhaEmpresa", APIUsuario: "joao.silva", APISenha: "senha"},
		User:        UserInput{Nome: "Joao Silva", Cargo: "Engenheiro"},
		Works:       WorksInput{Obras: []models.Obra{{ID: 121, Nome: "Residencial"}}},
	}
}

type fakeConfigStore struct {
	loadErr            error
	saveErr            error
	loadedConfig       models.Config
	saved              bool
	savedConfig        models.Config
	credentialsUpdated bool
	updatedEmpresa     models.Empresa
}

func (s *fakeConfigStore) Load() (models.Config, error) {
	if s.loadErr != nil {
		return models.Config{}, s.loadErr
	}
	return s.loadedConfig, nil
}

func (s *fakeConfigStore) Save(cfg models.Config) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = true
	s.savedConfig = cfg
	return nil
}

func (s *fakeConfigStore) UpdateCredentials(empresa models.Empresa) error {
	s.credentialsUpdated = true
	s.updatedEmpresa = empresa
	return nil
}

type fakeCredentialValidator struct {
	called bool
	err    error
}

func (v *fakeCredentialValidator) ValidateCredentials(ctx context.Context, empresa models.Empresa) error {
	v.called = true
	return v.err
}
