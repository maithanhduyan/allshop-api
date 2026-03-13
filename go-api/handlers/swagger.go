package handlers

import (
	_ "embed"
	"net/http"
)

//go:embed swagger-ui/index.html
var swaggerIndexHTML []byte

// SwaggerUI returns an http.HandlerFunc that serves the OpenAPI spec and Swagger UI.
func SwaggerUI(specData []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/docs/openapi.yaml":
			w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
			w.Write(specData)
		case "/docs", "/docs/", "/docs/index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(swaggerIndexHTML)
		default:
			http.NotFound(w, r)
		}
	}
}
