package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	defaultAPIStatsLabels = []string{"cluster"}
)

type labeledValues struct {
	labels []string
	value  float64
}

type apiStat struct {
	Type  prometheus.ValueType
	Desc  *prometheus.Desc
	Value func(resp apiStatsResponse) []labeledValues
}

type apiMetric struct {
	Type  prometheus.ValueType
	Desc  *prometheus.Desc
	Value func(resp apiStatsResponse) float64
}

type apiStatEntry map[string]float64

type apiStatsResponse struct {
	DeleteLatency           float64      `json:"delete_latency_ms"`
	DeleteRequestsPerSecond float64      `json:"delete_requests_per_second"`
	ImportLatency           float64      `json:"import_latency_ms"`
	ImportRequestsPerSecond float64      `json:"import_requests_per_second"`
	Latency                 apiStatEntry `json:"latency_ms"`
	PendingWriteBatches     float64      `json:"pending_write_batches"`
	RequestsPerSecond       apiStatEntry `json:"requests_per_second"`
	SearchLatency           float64      `json:"search_latency_ms"`
	SearchRequestsPerSecond float64      `json:"search_requests_per_second"`
	TotalRequestsPerSecond  float64      `json:"total_requests_per_second"`
	WriteLatency            float64      `json:"write_latency_ms"`
	WriteRequestsPerSecond  float64      `json:"write_requests_per_second"`
}

type APIStats struct {
	logger *log.Logger
	client *http.Client
	url    *url.URL

	up                              prometheus.Gauge
	totalScrapes, jsonParseFailures prometheus.Counter

	metrics []*apiMetric
	stats   []*apiStat
}

func splitStatKey(s string) (string, string) {
	split := strings.Split(s, " ")
	return split[0], split[1]
}

func NewAPIStats(logger *log.Logger, client *http.Client, url *url.URL) *APIStats {
	subsystem := "api_stats"

	return &APIStats{
		logger: logger,
		client: client,
		url:    url,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "up"),
			Help: "Was the last scrape of the Typesense API stats endpoint successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "total_scrapes"),
			Help: "Current total Typesense API stats scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		metrics: []*apiMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "delete_latency_seconds"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.DeleteLatency) / 1000.0
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "delete_requests_per_second"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.DeleteRequestsPerSecond)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "import_latency_seconds"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.ImportLatency) / 1000.0
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "import_requests_per_second"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.ImportRequestsPerSecond)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "pending_write_batches"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.PendingWriteBatches)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "search_latency_seconds"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.SearchLatency) / 1000.0
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "search_requests_per_second"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.SearchRequestsPerSecond)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "total_requests_per_second"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.TotalRequestsPerSecond)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "write_latency_seconds"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.WriteLatency) / 1000.0
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "write_requests_per_second"),
					"",
					defaultAPIStatsLabels, nil,
				),
				Value: func(resp apiStatsResponse) float64 {
					return float64(resp.WriteRequestsPerSecond)
				},
			},
		},
		stats: []*apiStat{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "latency_seconds"),
					"",
					[]string{"cluster", "method", "endpoint"},
					nil,
				),
				Value: func(resp apiStatsResponse) []labeledValues {
					ret := make([]labeledValues, 0, len(resp.Latency))
					for key, val := range resp.Latency {
						method, endpoint := splitStatKey(key)
						ret = append(ret, labeledValues{
							labels: []string{url.String(), method, endpoint},
							value:  float64(val) / 1000.0,
						})
					}
					return ret
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "requests_per_second"),
					"",
					[]string{"cluster", "method", "endpoint"},
					nil,
				),
				Value: func(resp apiStatsResponse) []labeledValues {
					ret := make([]labeledValues, 0, len(resp.RequestsPerSecond))
					for key, val := range resp.RequestsPerSecond {
						method, endpoint := splitStatKey(key)
						ret = append(ret, labeledValues{
							labels: []string{url.String(), method, endpoint},
							value:  val,
						})
					}
					return ret
				},
			},
		},
	}
}

// Describe set Prometheus metrics descriptions.
func (c *APIStats) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.Desc
	}

	ch <- c.up.Desc()
	ch <- c.totalScrapes.Desc()
	ch <- c.jsonParseFailures.Desc()
}

// Collect collects APIStats metrics.
func (c *APIStats) Collect(ch chan<- prometheus.Metric) {
	var err error
	c.totalScrapes.Inc()
	defer func() {
		ch <- c.up
		ch <- c.totalScrapes
		ch <- c.jsonParseFailures
	}()

	start := time.Now()
	resp, err := c.fetchAndDecodeAPIStats()
	if err != nil {
		c.up.Set(0)
		c.logger.WithError(err).Warnln("failed to fetch and decode API stats")
		return
	}
	c.up.Set(1)

	c.logger.WithField("duration", time.Since(start)).Debugln("fetched API stats successfully")

	for _, metric := range c.metrics {
		ch <- prometheus.MustNewConstMetric(
			metric.Desc,
			metric.Type,
			metric.Value(resp),
			c.url.String(),
		)
	}

	for _, stat := range c.stats {
		for _, v := range stat.Value(resp) {
			ch <- prometheus.MustNewConstMetric(
				stat.Desc,
				stat.Type,
				v.value,
				v.labels...,
			)
		}
	}
}

func (c *APIStats) fetchAndDecodeAPIStats() (apiStatsResponse, error) {
	var resp apiStatsResponse

	u := *c.url
	u.Path = path.Join(u.Path, "/stats.json")
	res, err := c.client.Get(u.String())
	if err != nil {
		return resp, fmt.Errorf("failed to get API stats from %s: %s", u.String(), err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			c.logger.WithError(err).Warnln("failed to close http.Client")
		}
	}()

	if res.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("HTTP request failed with code %d", res.StatusCode)
	}

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.jsonParseFailures.Inc()
		return resp, err
	}
	if err := json.Unmarshal(bts, &resp); err != nil {
		c.jsonParseFailures.Inc()
		return resp, err
	}

	return resp, nil
}
