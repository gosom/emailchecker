package api

import (
	"context"
	"fmt"

	"emailchecker/api/handlers"
	"emailchecker/pkg/httpext"
	"emailchecker/pkg/httpmiddleware"

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
	)

	s.router.NotFound(httpmiddleware.Handler(s.opsHandler.NotFound))
	s.router.MethodNotAllowed(httpmiddleware.Handler(s.opsHandler.MethodNotAllowed))

	s.router.Get("/health", httpmiddleware.Handler(s.opsHandler.Health))
	s.router.Get("/check/{email}", httpmiddleware.Handler(s.checkHandler.CheckEmail))
}
