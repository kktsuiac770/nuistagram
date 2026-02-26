package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Metrics struct {
	requestsTotal    *Counter
	requestDuration  *Histogram
	requestsInFlight *Gauge
	errorsTotal      *Counter
	dbQueriesTotal   *Counter
	dbQueryDuration  *Histogram
}

type Counter struct {
	mu    sync.RWMutex
	value float64
}

type Gauge struct {
	mu    sync.RWMutex
	value float64
}

type Histogram struct {
	mu      sync.RWMutex
	buckets map[float64]uint64
	sum     float64
	count   uint64
}

var defaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

func NewHistogram() *Histogram {
	h := &Histogram{buckets: make(map[float64]uint64)}
	for _, b := range defaultBuckets {
		h.buckets[b] = 0
	}
	return h
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Add(v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += v
}

func (c *Counter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (g *Gauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value++
}

func (g *Gauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value--
}

func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += v
	h.count++
	for _, b := range defaultBuckets {
		if v <= b {
			h.buckets[b]++
		}
	}
}

var M *Metrics

func Init() {
	M = &Metrics{
		requestsTotal:    &Counter{},
		requestDuration:  NewHistogram(),
		requestsInFlight: &Gauge{},
		errorsTotal:      &Counter{},
		dbQueriesTotal:   &Counter{},
		dbQueryDuration:  NewHistogram(),
	}
}

func (m *Metrics) RecordRequest(method, path string, status int, duration time.Duration) {
	m.requestsTotal.Inc()
	m.requestDuration.Observe(duration.Seconds())
	if status >= 400 {
		m.errorsTotal.Inc()
	}
}

func (m *Metrics) IncInFlight() { m.requestsInFlight.Inc() }
func (m *Metrics) DecInFlight() { m.requestsInFlight.Dec() }

func (m *Metrics) RecordDBQuery(duration time.Duration) {
	m.dbQueriesTotal.Inc()
	m.dbQueryDuration.Observe(duration.Seconds())
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		M.IncInFlight()
		defer M.DecInFlight()

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		M.RecordRequest(r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	output := `# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total ` + strconv.FormatFloat(M.requestsTotal.Value(), 'f', 0, 64) + `

# HELP http_request_duration_seconds HTTP request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_sum ` + strconv.FormatFloat(M.requestDuration.sum, 'f', 6, 64) + `
http_request_duration_seconds_count ` + strconv.FormatUint(M.requestDuration.count, 10) + `
`
	for _, b := range defaultBuckets {
		output += `http_request_duration_seconds_bucket{le="` + strconv.FormatFloat(b, 'f', -1, 64) + `"} ` + strconv.FormatUint(M.requestDuration.buckets[b], 10) + "\n"
	}
	output += `http_request_duration_seconds_bucket{le="+Inf"} ` + strconv.FormatUint(M.requestDuration.count, 10) + "\n"

	output += `
# HELP http_requests_in_flight Current number of HTTP requests being processed
# TYPE http_requests_in_flight gauge
http_requests_in_flight ` + strconv.FormatFloat(M.requestsInFlight.Value(), 'f', 0, 64) + `

# HELP http_errors_total Total number of HTTP errors
# TYPE http_errors_total counter
http_errors_total ` + strconv.FormatFloat(M.errorsTotal.Value(), 'f', 0, 64) + `

# HELP db_queries_total Total number of database queries
# TYPE db_queries_total counter
db_queries_total ` + strconv.FormatFloat(M.dbQueriesTotal.Value(), 'f', 0, 64) + `

# HELP db_query_duration_seconds Database query latency
# TYPE db_query_duration_seconds histogram
db_query_duration_seconds_sum ` + strconv.FormatFloat(M.dbQueryDuration.sum, 'f', 6, 64) + `
db_query_duration_seconds_count ` + strconv.FormatUint(M.dbQueryDuration.count, 10) + "\n"

	w.Write([]byte(output))
}
