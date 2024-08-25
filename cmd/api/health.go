package main

import (
	"net/http"
)

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.cfg.env,
			"version":     version,
		},
	}

	err := app.writeJson(w, http.StatusOK, data, nil)
	if err != nil {
		app.serverErrorReponse(w, r, err)
	}
}
