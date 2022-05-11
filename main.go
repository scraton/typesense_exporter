package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	collector "github.com/scraton/typesense_exporter/collector"

	flag "github.com/namsral/flag"
	log "github.com/sirupsen/logrus"

	prometheus "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	version "github.com/prometheus/common/version"
)

const name = "typesense_exporter"

type transportWithAPIKey struct {
	underlyingTransport http.RoundTripper
	apiKey              string
}

func (t *transportWithAPIKey) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Typesense-API-Key", t.apiKey)
	return t.underlyingTransport.RoundTrip(req)
}

func main() {
	var (
		listenAddressFlag    string
		telemetryPathFlag    string
		typesenseURLFlag     string
		typesenseTimeoutFlag string
		typesenseApiKeyFlag  string
		logLevelFlag         string
	)

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "", 0)
	fs.StringVar(&listenAddressFlag, "listen-address", ":9115", "address to listen on for metrics interface")
	fs.StringVar(&telemetryPathFlag, "telemetry-path", "/metrics", "path under which to expose metrics")
	fs.StringVar(&typesenseURLFlag, "typesense-url", "http://localhost:8108", "HTTP API address for Typesense node")
	fs.StringVar(&typesenseTimeoutFlag, "typesense-timeout", "5s", "timeout for trying to get Typesense metrics")
	fs.StringVar(&typesenseApiKeyFlag, "typesense-api-key", "", "API key for typesense")
	fs.StringVar(&logLevelFlag, "log-level", "info", "sets log level")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}

		log.WithError(err).Fatal("unable to parse arguments")
	}

	// Initialize logger
	logLevel, _ := log.ParseLevel(logLevelFlag)
	logger := &log.Logger{
		Out:       os.Stdout,
		Formatter: new(log.TextFormatter),
		Hooks:     make(log.LevelHooks),
		Level:     logLevel,
	}

	typesenseURL, err := url.Parse(typesenseURLFlag)
	if err != nil {
		logger.WithError(err).Fatalf("unable to parse typesense url")
	}

	typesenseTimeout, err := time.ParseDuration(typesenseTimeoutFlag)
	if err != nil {
		logger.WithError(err).Fatalf("unable to parse timeout")
	}

	if typesenseApiKeyFlag == "" {
		logger.Fatal("no API key provided")
	}

	logger.WithFields(log.Fields{
		"listen":  listenAddressFlag,
		"path":    telemetryPathFlag,
		"url":     typesenseURL,
		"timeout": typesenseTimeout,
	}).Debugln("initialized")

	var httpTransport http.RoundTripper

	httpTransport = &transportWithAPIKey{
		apiKey: typesenseApiKeyFlag,
		underlyingTransport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	httpClient := &http.Client{
		Timeout:   typesenseTimeout,
		Transport: httpTransport,
	}

	prometheus.MustRegister(version.NewCollector(name))
	prometheus.MustRegister(collector.NewClusterMetrics(logger, httpClient, typesenseURL))
	prometheus.MustRegister(collector.NewAPIStats(logger, httpClient, typesenseURL))

	server := &http.Server{}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	mux := http.DefaultServeMux
	mux.Handle(telemetryPathFlag, promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`<html>
			<head><title>Typesense Exporter</title></head>
			<body>
			<h1>Typesense Exporter</h1>
			<p><a href="` + telemetryPathFlag + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			logger.WithError(err).Errorln("failed handling writing")
		}
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
	})

	server.Handler = mux
	server.Addr = listenAddressFlag

	logger.WithField("addr", listenAddressFlag).Infof("starting typesense exporter")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}

			logger.WithError(err).Fatalln("server failed")
		}
	}()

	<-ctx.Done()
	logger.Infoln("shutting down")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	server.Shutdown(shutdownCtx)
}
