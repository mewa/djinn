package djinn

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	MHttpRequests       = stats.Int64("dcron/http_requests", "Number of HTTP API requests", stats.UnitDimensionless)
	MHttpRequestLatency = stats.Float64("dcron/http_request_latency", "HTTP API request latency", "ms")
	MJobExecutions      = stats.Int64("dcron/job_executions", "Executions of jobs in distributed cron", stats.UnitDimensionless)
)

var (
	KeyStatus, _ = tag.NewKey("status")
	KeyMethod, _ = tag.NewKey("method")
	KeyType, _   = tag.NewKey("type")
)

var (
	HttpRequestLatencyView = &view.View{
		Name:        "http_request_latency",
		Measure:     MHttpRequestLatency,
		Description: "Put job request latency distribution",
		Aggregation: view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
	}
	HttpRequestCountView = &view.View{
		Name:        "http_requests",
		Measure:     MHttpRequests,
		Description: "The number of new jobs",
		Aggregation: view.Count(),
	}
	JobExecutionsView = &view.View{
		Name:        "job_executions",
		Measure:     MJobExecutions,
		Description: "The number of job executions",
		Aggregation: view.Count(),
	}
)

func (d *Djinn) initMetrics() error {
	view.Register(HttpRequestLatencyView)
	view.Register(HttpRequestCountView)
	view.Register(JobExecutionsView)
	return nil
}
