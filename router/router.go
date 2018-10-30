package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go"
)

// NewServeMux creates a new TracedServeMux.
func NewServeMux(tracer opentracing.Tracer) *TracedServeMux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(SetJSON)
	r.Use(cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}).Handler)
	return &TracedServeMux{
		mux:    r,
		tracer: tracer,
	}
}

// TracedServeMux is a wrapper around http.ServeMux that instruments handlers for tracing.
type TracedServeMux struct {
	mux    *chi.Mux
	tracer opentracing.Tracer
}

// Handle implements http.ServeMux#Handle
func (tm *TracedServeMux) Handle(pattern string, handler http.Handler) {
	middleware := nethttp.Middleware(
		tm.tracer,
		handler,
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method + " " + pattern
		}))
	tm.mux.Handle(pattern, middleware)
}

// ServeHTTP implements http.ServeMux#ServeHTTP
func (tm *TracedServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm.mux.ServeHTTP(w, r)
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
		if span := zipkin.SpanFromContext(r.Context()); span != nil {
			span.SetName(routePattern)
		}
	})
}
