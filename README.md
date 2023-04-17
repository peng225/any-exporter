# any-exporter

## What's this?

any-exporter is an artificial metrics exporter.

By exporting whatever metrics you want and querying them by
[PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/), you can check the subtle behavior of the PromQL.

## How to use

### Basics

You can post a YAML file to define the metrics to export.
Then, add a Prometheus scraping rule for any-exporter.

### Metrics definition

The metrics definition is written in the YAML format.
You need to specify the following items in a YAML file.

- spec
  - name: Metrics name
  - type: Metrics type (currently, only counter, gauge and histogram are supported)
  - labels: The list of metrics labels
  - buckets (for histogram): Histogram buckets
- data
  - labels: The list of the key and value.
    - key: The key's name
    - value: The value of the key
  - sequence (for counter and gauge): The exported sequence of the values. You can define the sequence by using the notation for [Prometheus's unit test](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/#series) without '_' which specifies the missing sample. Each value is exported in order every time the metrics are scraped. Note that each value in a sequence of counter means to-be-added value while that of counter does the actual exported value.
  - observedValues (for histogram): The list of observed values. Each list item is consumed one by one every time the metrics are scraped. Though you can use Prometheus's unit test notation here, the semantics is quite different from those of counter and gauge. All values specified in a list item are digested at the same scraping time.

You can define several metrics in a YAML file.

See also the sample files in `e2e` directory.

### API reference

#### /recipe

| method | description| response |
|------|------|---|
| post | Post the definition of the metrics. You should set the request body to the input YAML file contents.| 200: success<br />400: input YAML file is invalid<br />409: the metrics is already registered |
| delete | Delete the definition of the metrics which has no data to export anymore. By setting the `force` parameter to `true`, you can delete all the metrics definitions forcibly.| 200: success |

#### /metrics

| method | description|response |
|------|------|---|
| get | You can scrape the exported metrics. |200: success |

#### /health

| method | description|response |
|------|------|---|
| get | This can be used for the health check. |200: success |
