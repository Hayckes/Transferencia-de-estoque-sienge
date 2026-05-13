//go:build !cgo

package ui

import "fmt"

func Run() {
	fmt.Println("A interface grafica Fyne requer CGO habilitado neste ambiente.")
}
