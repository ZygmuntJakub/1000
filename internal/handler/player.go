package handler

import "github.com/labstack/echo/v4"

func (h *Handler) AddPlayer(c echo.Context) error {
	return c.String(200, "to do")
}

func (h *Handler) ListPlayers(c echo.Context) error {
	return c.String(200, "to do")
}
