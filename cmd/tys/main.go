package main

import (
	"os"

	"github.com/ZygmuntJakub/1000/internal/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "simulation" {
		StartSimulation()
		return
	}
	h := handler.Handler{}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/ping", func(c echo.Context) error {
		return c.String(200, "pong")
	})

	e.POST("/player", h.AddPlayer)
	e.GET("/player", h.ListPlayers)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "1337"
	}
	e.Logger.Fatal(e.Start(":" + httpPort))
}
