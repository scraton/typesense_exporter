package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	defaultClusterMetricsLabels = []string{"cluster"}
)

type clusterMetric struct {
	Type prometheus.ValueType
	Desc *prometheus.Desc
	Value func(resp clusterMetricsResponse) float64
}

type clusterMetricsResponse struct {
	SystemCPU1ActivePercentage float64 `json:"system_cpu1_active_percentage,string"`
	SystemCPU2ActivePercentage float64 `json:"system_cpu2_active_percentage,string"`
	SystemCPU3ActivePercentage float64 `json:"system_cpu3_active_percentage,string"`
	SystemCPU4ActivePercentage float64 `json:"system_cpu4_active_percentage,string"`
	SystemCPUActivePercentage float64 `json:"system_cpu_active_percentage,string"`
	SystemDiskTotalBytes int `json:"system_disk_total_bytes,string"`
	SystemDiskUsedBytes int `json:"system_disk_used_bytes,string"`
	SystemMemoryTotalBytes int `json:"system_memory_total_bytes,string"`
	SystemMemoryUsedBytes int `json:"system_memory_used_bytes,string"`
	SystemNetworkReceivedBytes int `json:"system_network_received_bytes,string"`
	SystemNetworkSentBytes int `json:"system_network_sent_bytes,string"`
	TypesenseMemoryActiveBytes int `json:"typesense_memory_active_bytes,string"`
	TypesenseMemoryAllocatedBytes int `json:"typesense_memory_allocated_bytes,string"`
	TypesenseMemoryFragmentationRatio float64 `json:"typesense_memory_fragmentation_ratio,string"`
	TypesenseMemoryMappedBytes int `json:"typesense_memory_mapped_bytes,string"`
	TypesenseMemoryMetadataBytes int `json:"typesense_memory_metadata_bytes,string"`
	TypesenseMemoryResidentBytes int `json:"typesense_memory_resident_bytes,string"`
	TypesenseMemoryRetainedBytes int `json:"typesense_memory_retained_bytes,string"`
}

type ClusterMetrics struct {
	logger *log.Logger
	client *http.Client
	url    *url.URL

	up                              prometheus.Gauge
	totalScrapes, jsonParseFailures prometheus.Counter

	metrics []*clusterMetric
}

func NewClusterMetrics(logger *log.Logger, client *http.Client, url *url.URL) *ClusterMetrics {
	subsystem := "cluster_metrics"

	return &ClusterMetrics{
		logger: logger,
		client: client,
		url:    url,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "up"),
			Help: "Was the last scrape of the Typesense cluster metrics endpoint successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "total_scrapes"),
			Help: "Current total Typesense cluster metrics scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subsystem, "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		metrics: []*clusterMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_active_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryActiveBytes)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_allocated_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryAllocatedBytes)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_fragmentation_ratio"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryFragmentationRatio)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_mapped_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryMappedBytes)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_metadata_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryMetadataBytes)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_resident_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryResidentBytes)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, subsystem, "memory_retained_bytes"),
					"",
					defaultClusterMetricsLabels, nil,
				),
				Value: func(resp clusterMetricsResponse) float64 {
					return float64(resp.TypesenseMemoryRetainedBytes)
				},
			},
		},
	}
}

// Describe set Prometheus metrics descriptions.
func (c *ClusterMetrics) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.Desc
	}

	ch <- c.up.Desc()
	ch <- c.totalScrapes.Desc()
	ch <- c.jsonParseFailures.Desc()
}

// Collect collects cluster metrics.
func (c *ClusterMetrics) Collect(ch chan<- prometheus.Metric) {
	var err error
	c.totalScrapes.Inc()
	defer func() {
		ch <- c.up
		ch <- c.totalScrapes
		ch <- c.jsonParseFailures
	}()

	start := time.Now()
	resp, err := c.fetchAndDecodeClusterMetrics()
	if err != nil {
		c.up.Set(0)
		c.logger.WithError(err).Warnln("failed to fetch and decode cluster metrics")
		return
	}
	c.up.Set(1)

	c.logger.WithField("duration", time.Since(start)).Debugln("fetched cluster metrics successfully")

	for _, metric := range c.metrics {
		ch <- prometheus.MustNewConstMetric(
			metric.Desc,
			metric.Type,
			metric.Value(resp),
			c.url.String(),
		)
	}
}

func (c *ClusterMetrics) fetchAndDecodeClusterMetrics() (clusterMetricsResponse, error) {
	var resp clusterMetricsResponse

	u := *c.url
	u.Path = path.Join(u.Path, "/metrics.json")
	res, err := c.client.Get(u.String())
	if err != nil {
		return resp, fmt.Errorf("failed to get cluster metrics from %s: %s", u.String(), err)
	}
	defer func(){
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
