package main

import (
	"embed"
	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
	"log"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() { application.RegisterEvent[core.Status]("connection:state-changed") }

func main() {
	app := application.New(application.Options{
		Name: "Xray Mini", Description: "Local Xray proxy client",
		Services: []application.Service{application.NewService(NewApp())},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Xray Mini", Width: 380, Height: 640, MinWidth: 360, MinHeight: 560,
		BackgroundColour: application.NewRGB(13, 17, 25), URL: "/",
		Windows: application.WindowsWindow{Theme: application.Dark},
	})
	if err := app.Run(); err != nil {
		log.Fatal("Unable to launch desktop window: ", err)
	}
}
