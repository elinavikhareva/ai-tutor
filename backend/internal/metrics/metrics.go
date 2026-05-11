package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "ai_tutor"

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "HTTP requests by route and status.",
	}, []string{"method", "route", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "route"})

	LLMRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "llm_requests_total",
		Help:      "Requests to the LLM API by operation and outcome.",
	}, []string{"operation", "outcome"})

	LLMDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "llm_request_duration_seconds",
		Help:      "LLM API latency, including retries.",
		Buckets:   prometheus.ExponentialBuckets(0.25, 2, 10),
	}, []string{"operation"})

	CompactionJobs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "compaction_jobs_total",
		Help:      "Processed lesson compaction jobs by outcome.",
	}, []string{"outcome"})
)

func Handler() http.Handler {
	return promhttp.Handler()
}
