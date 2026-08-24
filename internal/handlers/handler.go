package handlers

import "server-watch/internal/system"

type Handler struct {
	system *system.System
}

func NewHandler(sys *system.System) *Handler {
	return &Handler{
		system: sys,
	}
}
