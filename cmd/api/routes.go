package main

import (
	"net/http"

	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	v1router := http.NewServeMux()
	v1router.HandleFunc("GET /health", app.healthHandler)
	v1router.HandleFunc("GET /movies", app.listMovieHandler)
	v1router.HandleFunc("POST /movies", app.createMovieHandler)
	v1router.HandleFunc("GET /movies/{id}", app.showMovieHandler)
	v1router.HandleFunc("PATCH /movies/{id}", app.updateMovieHandler)
	v1router.HandleFunc("DELETE /movies/{id}", app.deleteMovieHandler)

	router := http.NewServeMux()
	router.Handle("/v1/", http.StripPrefix("/v1", v1router))

	middlewareChain := alice.New(app.rateLimit, app.httpError)

	return middlewareChain.Then(router)
}
