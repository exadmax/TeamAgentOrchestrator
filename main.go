package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := NewApp()
	var browserArgs []string
	if port := os.Getenv("TAO_DEVTOOLS_PORT"); port != "" {
		// lets scripts read the page over the DevTools protocol (manual checks without screen access)
		browserArgs = append(browserArgs, "--remote-debugging-port="+port)
	}
	app := application.New(application.Options{
		Name:        "TeamAgent Orquestrador",
		Description: "Orquestrador de times de agentes de IA",
		Services:    []application.Service{application.NewService(a)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Windows:     application.WindowsOptions{AdditionalBrowserArgs: browserArgs},
	})
	a.wapp = app
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "TeamAgent Orquestrador",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(0, 128, 128), // Win95 teal desktop
		URL:              "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
