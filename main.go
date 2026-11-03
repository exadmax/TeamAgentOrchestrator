package main

import (
	"embed"
	"log"
	"os"

	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if lang := loadSettings().Lang; lang != "" {
		_ = i18n.Set(lang) // an unknown value keeps English
	}
	a := NewApp()
	var browserArgs []string
	if port := os.Getenv("TAO_DEVTOOLS_PORT"); port != "" {
		// lets scripts read the page over the DevTools protocol (manual checks without screen access)
		browserArgs = append(browserArgs, "--remote-debugging-port="+port)
	}
	app := application.New(application.Options{
		Name:        "TeamAgent Orchestrator",
		Description: "Orchestrator for teams of AI coding agents",
		Services:    []application.Service{application.NewService(a)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Windows:     application.WindowsOptions{AdditionalBrowserArgs: browserArgs},
	})
	a.wapp = app
	a.win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            i18n.T("app.window_title"),
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
