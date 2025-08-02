package httpext

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/coreos/go-systemd/v22/daemon"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"emailchecker/pkg/log"
)

type Option func(*HTTPServer) error

type HTTPServer struct {
	srv      *http.Server
	listener net.Listener

	addr    string
	domains []string

	port      int
	certFile  string
	certCache string
	keyFile   string
}

func New(router http.Handler, opts ...Option) (*HTTPServer, error) {
	ans := HTTPServer{}

	for _, opt := range opts {
		if err := opt(&ans); err != nil {
			return nil, err
		}
	}

	setupDefaults(&ans)

	srv := &http.Server{
		Addr:              ans.addr,
		Handler:           router,
		ReadTimeout:       5 * time.Second,  //nolint:gomnd // TODO
		WriteTimeout:      10 * time.Second, //nolint:gomnd // TODO
		IdleTimeout:       5 * time.Second,  //nolint:gomnd // TODO
		ReadHeaderTimeout: 5 * time.Second,  //nolint:gomnd // TODO
		MaxHeaderBytes:    1 << 20,          //nolint:gomnd // TODO
	}

	if isHTTPSPort(ans.port) {
		if ans.certFile == "" || ans.keyFile == "" {
			autoTLSManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				Cache:      autocert.DirCache(ans.certCache),
				HostPolicy: autocert.HostWhitelist(ans.domains...),
			}
			// https://ssl-config.mozilla.org/#server=go&version=1.22.0&config=intermediate&guideline=5.7
			srv.TLSConfig = &tls.Config{
				MinVersion:               tls.VersionTLS12,
				PreferServerCipherSuites: true,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				},
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.X25519,
				},
				GetCertificate: autoTLSManager.GetCertificate,
				NextProtos: []string{
					"h2", "http/1.1", // enable HTTP/2
					acme.ALPNProto, // enable tls-alpn ACME challenges
				},
			}
		}
	}

	ans.srv = srv

	return &ans, nil
}

func WithDomains(domains ...string) Option {
	return func(s *HTTPServer) error {
		s.domains = domains

		return nil
	}
}

func WithCertFiles(certFile, keyFile string) Option {
	return func(s *HTTPServer) error {
		s.certFile = certFile
		s.keyFile = keyFile

		if _, err := os.Stat(certFile); err != nil {
			return err
		}

		if _, err := os.Stat(keyFile); err != nil {
			return err
		}

		return nil
	}
}

func WithAddr(addr string) Option {
	return func(s *HTTPServer) error {
		var err error

		s.addr = addr
		s.port, err = extractPortFromAddr(addr)

		return err
	}
}

func WithSystemdSocket() Option {
	return func(s *HTTPServer) error {
		listeners, err := activation.Listeners()
		if err != nil || len(listeners) == 0 {
			return nil
		}

		s.listener = listeners[0]
		return nil
	}
}

func (h *HTTPServer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		h.gracefulShutdown(ctx)
	}()

	errc := make(chan error, 1)
	ready := make(chan struct{})

	go func() {
		if isHTTPSPort(h.port) {
			if h.listener != nil {
				log.Info(ctx, fmt.Sprintf("Using systemd socket for HTTPS on: %s", h.addr))

				close(ready)

				errc <- h.srv.ServeTLS(h.listener, h.certFile, h.keyFile)

				return
			}

			if h.certFile != "" && h.keyFile != "" {
				log.Info(ctx, fmt.Sprintf("Starting HTTPS server on: %s", h.addr))

				close(ready)

				errc <- h.srv.ListenAndServeTLS(h.certFile, h.keyFile)

				return
			}

			log.Info(ctx, "Starting HTTPS server (autotls) on %s", h.addr)
			close(ready)

			errc <- h.srv.ListenAndServeTLS("", "")

			return
		}
		if h.listener != nil {
			log.Info(ctx, "Using systemd socket for HTTP")

			close(ready)

			errc <- h.srv.Serve(h.listener)

			return
		}

		log.Info(ctx, fmt.Sprintf("Starting HTTP server on %s", h.addr))

		close(ready)

		errc <- h.srv.ListenAndServe()
	}()

	<-ready
	select {
	case <-ctx.Done():
		log.Warn(ctx, "Context cancelled before server started")
	case <-time.After(time.Millisecond * 100):
		if h.listener != nil {
			sent, err := daemon.SdNotify(false, daemon.SdNotifyReady)
			switch {
			case err != nil:
				log.Error(ctx, fmt.Errorf("failed to notify systemd: %w", err))
			case !sent:
				log.Warn(ctx, "Failed to notify systemd about readiness")
			default:
				log.Info(ctx, "Notified systemd about readiness")
			}
		} else {
			log.Info(ctx, "Server is ready")
		}
	}

	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (h *HTTPServer) gracefulShutdown(ctx context.Context) {
	log.Info(ctx, "Starting graceful shutdown")

	if h.listener != nil {
		sent, err := daemon.SdNotify(false, daemon.SdNotifyStopping)
		switch {
		case err != nil:
			log.Error(ctx, fmt.Errorf("failed to notify systemd: %w", err))
		case !sent:
			log.Warn(ctx, "Failed to notify systemd about stopping")
		default:
			log.Info(ctx, "Notified systemd about stopping")
		}
	}

	shutdownCtx, shutdownStop := context.WithTimeout(
		context.WithoutCancel(ctx),
		time.Second*10,
	)
	defer shutdownStop()

	if err := h.srv.Shutdown(shutdownCtx); err != nil {
		log.Error(ctx, fmt.Errorf("graceful shutdown failed: %w", err))
		h.srv.Close()
	}

	log.Info(ctx, "Graceful shutdown complete")
}

func extractPortFromAddr(addr string) (int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}

	port, err := strconv.Atoi(portStr)

	return port, err
}

const (
	defaultAddr      = ":80"
	defaultCertCache = "/.cache/certcache"
	defaultHost      = "localhost"
)

func isHTTPSPort(port int) bool {
	return port == 443 || port == 8443 || port == 9443
}

func setupDefaults(s *HTTPServer) {
	if s.addr == "" {
		s.addr = defaultAddr
	}

	if len(s.domains) == 0 {
		s.domains = append(s.domains, defaultHost)
	}

	if s.certCache == "" {
		s.certCache = defaultCertCache
	}
}
