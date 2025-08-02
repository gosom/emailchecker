package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"emailchecker/api/handlers"
	"emailchecker/api/handlers/middleware"
	"emailchecker/pkg/httpext"
	"emailchecker/pkg/httpmiddleware"
	"emailchecker/static"

	"github.com/go-chi/chi/v5"

	"emailchecker"
)

type Server struct {
	opsHandler   *handlers.OpsHandler
	checkHandler *handlers.CheckHandler

	httpServer *httpext.HTTPServer
	router     chi.Router
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewServer(checker *emailchecker.EmailChecker, opts ...httpext.Option) *Server {
	ans := Server{
		router: chi.NewRouter(),

		opsHandler:   handlers.NewOpsHandler(),
		checkHandler: handlers.NewCheckHandler(checker),
	}

	ans.setupRoutes()

	httpServer, err := httpext.New(
		ans.router,
		opts...,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create HTTP server: %v", err))
	}

	ans.httpServer = httpServer

	return &ans
}

func (s *Server) Run(ctx context.Context) error {
	return s.httpServer.Run(ctx)
}

func (s *Server) setupRoutes() {
	s.router.Use(httpmiddleware.Logging(
		httpmiddleware.SkipRequestBodyLogging(),
	),
		middleware.HostValidation,
	)

	s.router.NotFound(httpmiddleware.Handler(s.opsHandler.NotFound))
	s.router.MethodNotAllowed(httpmiddleware.Handler(s.opsHandler.MethodNotAllowed))

	s.router.Get("/health", httpmiddleware.Handler(s.opsHandler.Health))
	s.router.Get("/check/{email}", httpmiddleware.Handler(s.checkHandler.CheckEmail))

	staticFS, err := fs.Sub(static.StaticFiles, "src")
	if err != nil {
		panic(err)
	}

	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, staticFS, "index.html")
	})

	s.router.Get("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeFileFS(w, r, staticFS, "styles.css")
	})

	s.router.Get("/script.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		http.ServeFileFS(w, r, staticFS, "script.js")
	})
}
