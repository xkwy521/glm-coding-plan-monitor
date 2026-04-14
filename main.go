package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	cfg := app.LoadConfig()

	err := wails.Run(&options.App{
		Title:       "GLM 使用量监控",
		Width:       cfg.WindowWidth,
		Height:      50,
		AssetServer: &assetserver.Options{Assets: assets},
		Frameless:         true,
		AlwaysOnTop:       true,
		DisableResize:     false,
		StartHidden:       true,
		BackgroundColour:  &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		HideWindowOnClose: true,
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.SystemDefault,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
