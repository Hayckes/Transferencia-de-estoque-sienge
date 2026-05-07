package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

func BuildFatalErrorContent(message string) fyne.CanvasObject {
	return container.NewCenter(widget.NewLabel(message))
}

func BuildOnboardingContent(window fyne.Window, store ConfigStore, done func(configLoaded)) fyne.CanvasObject {
	service := OnboardingService{Store: store, Validator: SiengeCredentialValidator{}}
	var credentials CredentialsInput
	var user UserInput
	var obras []models.Obra

	status := widget.NewLabel("")
	content := container.NewVBox()

	var showStep1 func()
	var showStep2 func()
	var showStep3 func()

	showStep1 = func() {
		empresa := widget.NewEntry()
		empresa.SetPlaceHolder("Nome da empresa")
		subdominio := widget.NewEntry()
		subdominio.SetPlaceHolder("Subdominio Sienge")
		usuario := widget.NewEntry()
		usuario.SetPlaceHolder("Usuario API")
		senha := widget.NewPasswordEntry()
		senha.SetPlaceHolder("Senha API")

		content.Objects = []fyne.CanvasObject{
			widget.NewLabel("Configuracao inicial - Credenciais Sienge"),
			withMinTypingInputWidth(empresa),
			withMinTypingInputWidth(subdominio),
			withMinTypingInputWidth(usuario),
			withMinTypingInputWidth(senha),
			status,
			widget.NewButton("Validar e continuar", func() {
				credentials = CredentialsInput{EmpresaNome: empresa.Text, Subdominio: subdominio.Text, APIUsuario: usuario.Text, APISenha: senha.Text}
				empresaModel, err := ValidateCredentialsInput(credentials)
				if err != nil {
					status.SetText(err.Error())
					return
				}
				status.SetText(StatusLoading)
				go func() {
					err := service.Validator.ValidateCredentials(context.Background(), empresaModel)
					fyne.Do(func() {
						if err != nil {
							status.SetText("Credenciais nao validadas: " + err.Error())
							return
						}
						status.SetText("")
						showStep2()
					})
				}()
			}),
		}
		content.Refresh()
	}

	showStep2 = func() {
		nome := widget.NewEntry()
		nome.SetPlaceHolder("Nome completo")
		cargo := widget.NewEntry()
		cargo.SetPlaceHolder("Cargo/Função")
		content.Objects = []fyne.CanvasObject{
			widget.NewLabel("Configuracao inicial - Usuario"),
			withMinTypingInputWidth(nome),
			withMinTypingInputWidth(cargo),
			status,
			container.NewHBox(
				widget.NewButton("Voltar", showStep1),
				widget.NewButton("Continuar", func() {
					user = UserInput{Nome: nome.Text, Cargo: cargo.Text}
					if _, err := ValidateUserInput(user); err != nil {
						status.SetText(err.Error())
						return
					}
					status.SetText("")
					showStep3()
				}),
			),
		}
		content.Refresh()
	}

	showStep3 = func() {
		idEntry := widget.NewEntry()
		idEntry.SetPlaceHolder("ID da obra")
		nomeEntry := widget.NewEntry()
		nomeEntry.SetPlaceHolder("Nome da obra")
		lista := widget.NewLabel(worksListText(obras))
		add := func() {
			id, err := strconv.Atoi(strings.TrimSpace(idEntry.Text))
			if err != nil || id <= 0 {
				status.SetText("ID da obra deve ser numerico positivo")
				return
			}
			nova := models.Obra{ID: id, Nome: strings.TrimSpace(nomeEntry.Text)}
			validated, err := ValidateWorksInput(WorksInput{Obras: append(append([]models.Obra(nil), obras...), nova)})
			if err != nil {
				status.SetText(err.Error())
				return
			}
			obras = validated
			idEntry.SetText("")
			nomeEntry.SetText("")
			lista.SetText(worksListText(obras))
			status.SetText("")
		}

		content.Objects = []fyne.CanvasObject{
			widget.NewLabel("Configuracao inicial - Obras"),
			container.NewHBox(withMinTypingInputWidth(idEntry), withMinTypingInputWidth(nomeEntry), widget.NewButton("+ Adicionar outra obra", add)),
			lista,
			status,
			container.NewHBox(
				widget.NewButton("Voltar", showStep2),
				widget.NewButton("Concluir", func() {
					cfg, err := service.Complete(context.Background(), CompleteOnboardingInput{Credentials: credentials, User: user, Works: WorksInput{Obras: obras}})
					if err != nil {
						status.SetText(err.Error())
						return
					}
					if done != nil {
						done(configLoaded{Config: cfg})
					}
				}),
			),
		}
		content.Refresh()
	}

	showStep1()
	return container.NewCenter(container.NewPadded(content))
}

func worksListText(obras []models.Obra) string {
	if len(obras) == 0 {
		return "Nenhuma obra cadastrada."
	}
	lines := make([]string, 0, len(obras))
	for _, obra := range obras {
		lines = append(lines, obra.Label())
	}
	return fmt.Sprintf("Obras cadastradas:\n%s", strings.Join(lines, "\n"))
}
