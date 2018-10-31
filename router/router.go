package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go"
)

// NewServeMux creates a new TracedServeMux.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(SetJSON)
	r.Use(cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}).Handler)
	return r
}

func SetJSON(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func TrackRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		rctx := chi.RouteContext(r.Context())
		routePattern := fmt.Sprintf("[%s] %s", rctx.RouteMethod, strings.Join(rctx.RoutePatterns, ""))
		opentracing.SpanFromContext(r.Context())
		if span := zipkin.SpanFromContext(r.Context()); span != nil {
			span.SetName(routePattern)
		}
	})
}
