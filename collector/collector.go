package collector

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Namespace defines the common namespace to be used by all metrics.
const namespace = "typesense"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "duration_seconds"),
		"typesense_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "success"),
		"typesense_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(context.Context, chan<- prometheus.Metric) error
}

type TypesenseCollector struct {
	Collectors map[string]Collector
	logger     *log.Logger
}

// NewTypesenseCollector creates a new TypesenseCollector
func NewTypesenseCollector(logger *log.Logger, httpClient *http.Client, typesenseURL *url.URL) (*TypesenseCollector, error) {
	collectors := make(map[string]Collector)

	return &TypesenseCollector{
		Collectors: collectors,
		logger:     logger,
	}, nil
}

// Describe implements the prometheus.Collector interface.
func (e TypesenseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (e TypesenseCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	ctx := context.TODO()
	wg.Add(len(e.Collectors))
	for name, c := range e.Collectors {
		go func(name string, c Collector) {
			execute(ctx, name, c, ch, e.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(ctx context.Context, name string, c Collector, ch chan<- prometheus.Metric, logger *log.Logger) {
	begin := time.Now()
	err := c.Update(ctx, ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		success = 0
		logger.WithError(err).WithFields(log.Fields{
			"name":             name,
			"duration_seconds": duration.Seconds(),
		}).Errorln("collector failed")
	} else {
		success = 1
		logger.WithFields(log.Fields{
			"name":             name,
			"duration_seconds": duration.Seconds(),
		}).Debugln("collector succeeded")
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}
