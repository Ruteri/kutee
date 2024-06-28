package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/flashbots/go-template/common"
	"github.com/flashbots/go-template/metrics"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/atomic"
)

type HTTPServerConfig struct {
	ListenAddr  string
	MetricsAddr string
	EnablePprof bool
	Log         *slog.Logger

	DrainDuration            time.Duration
	GracefulShutdownDuration time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration

	Auth AuthConfig
}

type AuthConfig struct {
	AuthenticatedUsers map[string][]byte
	PasswordHasher     func(string) []byte
}

var EmptyAuthConfig AuthConfig = AuthConfig{
	AuthenticatedUsers: make(map[string][]byte),
	PasswordHasher: func(p string) []byte {
		return sha256.New().Sum([]byte(p))
	},
}

func (a AuthConfig) ParseJSONUsers(jsonConfig []byte) AuthConfig {
	if err := json.Unmarshal(jsonConfig, &(a.AuthenticatedUsers)); err != nil {
		panic(err)
	}
	return a
}

var DummyAuthConfig AuthConfig = AuthConfig{
	AuthenticatedUsers: map[string][]byte{
		"test": []byte("test"),
	},
	PasswordHasher: func(p string) []byte {
		return []byte(p)
	},
}

type Server struct {
	cfg     *HTTPServerConfig
	isReady atomic.Bool
	log     *slog.Logger

	kuteeAPI *KuteeAPI

	srv     *http.Server
	metrics *metrics.MetricsServer
}

func New(cfg *HTTPServerConfig) (srv *Server, err error) {
	metricsSrv, err := metrics.New(common.PackageName, cfg.MetricsAddr)
	if err != nil {
		return nil, err
	}

	srv = &Server{
		cfg:      cfg,
		log:      cfg.Log,
		kuteeAPI: NewKuteeAPI(cfg.Auth.AuthenticatedUsers, cfg.Auth.PasswordHasher),
		srv:      nil,
		metrics:  metricsSrv,
	}
	srv.isReady.Swap(true)

	measureAndHandle := func(name string, handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
		histogramName := "request_duration_" + name
		return func(w http.ResponseWriter, r *http.Request) {
			m := srv.metrics.Float64Histogram(
				histogramName,
				"API request handling duration",
				metrics.UomMicroseconds,
				metrics.BucketsRequestDuration...,
			)

			start := time.Now()

			handler(w, r)
			m.Record(r.Context(), float64(time.Since(start).Microseconds()))
		}
	}

	measureAuthenticateAndHandle := func(name string, handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
		return measureAndHandle(name, srv.kuteeAPI.AuthenticateAndHandle(handler))
	}

	mux := chi.NewRouter()

	mux.With(srv.httpLogger).Post("/api/upload_image", measureAuthenticateAndHandle("upload_image", srv.kuteeAPI.uploadImageTarball))
	mux.With(srv.httpLogger).Get("/api/start_workload", measureAuthenticateAndHandle("start_workload", srv.kuteeAPI.startWorkload))

	mux.With(srv.httpLogger).Get("/livez", srv.handleLivenessCheck)
	mux.With(srv.httpLogger).Get("/readyz", srv.handleReadinessCheck)
	mux.With(srv.httpLogger).Get("/drain", srv.handleDrain)
	mux.With(srv.httpLogger).Get("/undrain", srv.handleUndrain)

	if cfg.EnablePprof {
		srv.log.Info("pprof API enabled")
		mux.Mount("/debug", middleware.Profiler())
	}

	srv.srv = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv, nil
}

func (s *Server) httpLogger(next http.Handler) http.Handler {
	return httplogger.LoggingMiddlewareSlog(s.log, next)
}

func (s *Server) RunInBackground() {
	// metrics
	if s.cfg.MetricsAddr != "" {
		go func() {
			s.log.With("metricsAddress", s.cfg.MetricsAddr).Info("Starting metrics server")
			err := s.metrics.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.log.Error("HTTP server failed", "err", err)
			}
		}()
	}

	// api
	go func() {
		s.log.Info("Starting HTTP server", "listenAddress", s.cfg.ListenAddr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("HTTP server failed", "err", err)
		}
	}()
}

func (s *Server) Shutdown() {
	// api
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulShutdownDuration)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.Error("Graceful HTTP server shutdown failed", "err", err)
	} else {
		s.log.Info("HTTP server gracefully stopped")
	}

	// metrics
	if len(s.cfg.MetricsAddr) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulShutdownDuration)
		defer cancel()

		if err := s.metrics.Shutdown(ctx); err != nil {
			s.log.Error("Graceful metrics server shutdown failed", "err", err)
		} else {
			s.log.Info("Metrics server gracefully stopped")
		}
	}
}
