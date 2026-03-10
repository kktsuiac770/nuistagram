package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var once sync.Once

// Registry is the Prometheus registry used by NUIstagram.
// A custom registry avoids conflicts with packages that register to the default registry.
var Registry *prometheus.Registry

var (
	// HTTP layer metrics — recorded automatically by the Metrics middleware.
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Application-level metrics — incremented from individual handlers.
	PhotosUploadedTotal prometheus.Counter
	LoginsTotal         *prometheus.CounterVec
	RegistrationsTotal  *prometheus.CounterVec
	LikesTotal          prometheus.Counter
	CommentsTotal       prometheus.Counter
	FollowsTotal        prometheus.Counter
)

// Init creates and registers all metrics. Safe to call multiple times (idempotent).
func Init() {
	once.Do(func() { init_() })
}

func init_() {
	Registry = prometheus.NewRegistry()

	// Go runtime and OS process metrics.
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests partitioned by method, path, and status code.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"method", "path"})

	HTTPRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Current number of HTTP requests being processed.",
	})

	PhotosUploadedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuistagram_photos_uploaded_total",
		Help: "Total number of photos successfully uploaded.",
	})

	LoginsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nuistagram_logins_total",
		Help: "Total login attempts partitioned by status (success|failure).",
	}, []string{"status"})

	RegistrationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nuistagram_registrations_total",
		Help: "Total registration attempts partitioned by status (success|failure).",
	}, []string{"status"})

	LikesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuistagram_likes_total",
		Help: "Total photo like/unlike events.",
	})

	CommentsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuistagram_comments_total",
		Help: "Total comments posted.",
	})

	FollowsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuistagram_follows_total",
		Help: "Total follow events.",
	})

	Registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPRequestsInFlight,
		PhotosUploadedTotal,
		LoginsTotal,
		RegistrationsTotal,
		LikesTotal,
		CommentsTotal,
		FollowsTotal,
	)
}

