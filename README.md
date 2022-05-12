Typesense Exporter
==================

Prometheus exporter for various metrics about [Typesense](https://github.com/typesense/typesense), written in Go, and
just barely glued together to be functional.

Cluster metrics and API stats are exposed based on what Typesense itself provides via its endpoints. See [their
documentation](https://typesense.org/docs/0.22.2/api/cluster-operations.html#cluster-metrics) for more information.

### Installation

Check the [packages](https://github.com/scraton?tab=packages&repo_name=typesense_exporter) or [releases](https://github.com/scraton/typesense_exporter/releases) pages for pre-built options.

#### Docker

```bash
docker pull ghcr.io/scraton/typesense_exporter:main
docker run --rm -p 9115:9115 -e TYPESENSE_API_KEY=xyz ghcr.io/scraton/typesense_exporter:main
```

### Configuration

**NOTE:** Just like the [Elasticsearch Exporter](https://github.com/prometheus-community/elasticsearch_exporter), this
exporter fetches information from a Typesense cluster on every scrape. Having too short a scrape interval could cause
performance issues, so please plan accordingly.

| Argument            | Env Variable      | Description                                  | Default               |
| --------            | ------------      | -----------                                  | -------               |
| listen-address      | LISTEN_ADDRESS    | address to listen on for metrics interface   | :9115                 |
| telemetry-path      | TELEMETRY_PATH    | path under which to expose metrics           | /metrics              |
| typesense-url       | TYPESENSE_URL     | HTTP API address for Typesense node          | http://localhost:8108 |
| typesense-timeout   | TYPESENSE_TIMEOUT | timeout for trying to get Typesense metrics  | 5s                    |
| typesense-api-key   | TYPESENSE_API_KEY | API key for typesense                        |                       |
| log-level           | LOG_LEVEL         | sets log level                               | info                  |

### Metrics

Please see [Typesense's documentation](https://typesense.org/docs/0.22.2/api/cluster-operations.html#cluster-metrics)
for cluster metrics and API stats.

| Name                                                  | Type     | Cardinality  | Help
| ----                                                  | ----     | -----------  | ----
| typesense_api_stats_delete_latency_seconds            | gauge    | 1            | Latency for delete requests in seconds
| typesense_api_stats_delete_requests_per_second        | gauge    | 1            | Requests per second for deletions
| typesense_api_stats_import_latency_seconds            | gauge    | 1            | Latency for delete requests in seconds
| typesense_api_stats_import_requests_per_second        | gauge    | 1            | Requests per second for imports
| typesense_api_stats_json_parse_failures               | counter  | 0            | Number of errors while parsing JSON
| typesense_api_stats_latency_seconds                   | gauge    | 3            | Latency for each method and endpoint
| typesense_api_stats_pending_write_batches             | gauge    | 1            | Pending write batches
| typesense_api_stats_requests_per_second               | gauge    | 3            | Requests per second for each method and endpoint
| typesense_api_stats_search_latency_seconds            | gauge    | 1            | Latency for search requests in seconds
| typesense_api_stats_search_requests_per_second        | gauge    | 1            | Requests per second for searches
| typesense_api_stats_total_requests_per_second         | gauge    | 1            | Requests per second for all endpoints
| typesense_api_stats_total_scrapes                     | counter  | 0            | Current total Typesense API stats scrapes
| typesense_api_stats_up                                | gauge    | 0            | Was the last scrape of the Typesense stats.json endpoint successful
| typesense_api_stats_write_latency_seconds             | gauge    | 1            | Latency for write requests
| typesense_api_stats_write_requests_per_second         | gauge    | 1            | Requets per second for writes
| typesense_cluster_metrics_json_parse_failures         | counter  | 0            | Number of errors while parsing JSON
| typesense_cluster_metrics_memory_active_bytes         | gauge    | 1            | Total active memory in use by Typesense
| typesense_cluster_metrics_memory_allocated_bytes      | gauge    | 1            | Total allocated memory in use by Typesense
| typesense_cluster_metrics_memory_fragmentation_ratio  | gauge    | 1            | Fragmentation ratio for Typesense memory
| typesense_cluster_metrics_memory_mapped_bytes         | gauge    | 1            | Total mapped memory in use by Typesense
| typesense_cluster_metrics_memory_metadata_bytes       | gauge    | 1            | Total memory used for metadata by Typesense
| typesense_cluster_metrics_memory_resident_bytes       | gauge    | 1            | Total resident memory in use by Typesense
| typesense_cluster_metrics_memory_retained_bytes       | gauge    | 1            | Total retained memory in use by Typesense
| typesense_cluster_metrics_total_scrapes               | counter  | 0            | Current total Typesense cluster metrics scrapes
| typesense_cluster_metrics_up                          | gauge    | 0            | Was the last scrape of the Typesense metrics.json endpoint successful

## Credit & License

Code is based on the original work done by
[elasticsearch_exporter](https://github.com/prometheus-community/elasticsearch_exporter).

## Contributing & Development

Contributions are welcome. Please fork the project on GitHub and open Pull Requests for any proposed changes.

### Building

```bash
# ensure promu utility is installed (one time step):
make promu 

promu crossbuild
make docker
```
