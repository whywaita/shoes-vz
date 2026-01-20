package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the server
type Metrics struct {
	// Runner metrics
	RunnersTotal          *prometheus.GaugeVec
	RunnersIdle           prometheus.Gauge
	RunnersBusy           prometheus.Gauge
	RunnerStartupDuration prometheus.Histogram
	RunnerJobDuration     prometheus.Histogram

	// Agent metrics
	AgentsTotal           *prometheus.GaugeVec
	AgentsOnline          prometheus.Gauge
	AgentsCapacityRunners *prometheus.GaugeVec
	AgentsCurrentRunners  *prometheus.GaugeVec

	// Capacity metrics
	CapacityTotalRunners     prometheus.Gauge
	CapacityAvailableRunners prometheus.Gauge
	CapacityUtilizationRatio prometheus.Gauge

	// Error metrics
	RunnerFailuresTotal *prometheus.CounterVec
	RunnerErrors        prometheus.Gauge

	// Request metrics
	AddInstanceRequestsTotal    *prometheus.CounterVec
	DeleteInstanceRequestsTotal *prometheus.CounterVec
	AddInstanceDuration         prometheus.Histogram
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// Runner metrics
		RunnersTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "shoesvz_runners_total",
				Help: "Total number of runners by state",
			},
			[]string{"state"},
		),
		RunnersIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_runners_idle",
				Help: "Number of idle runners",
			},
		),
		RunnersBusy: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_runners_busy",
				Help: "Number of busy runners (running jobs)",
			},
		),
		RunnerStartupDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "shoesvz_runner_startup_duration_seconds",
				Help:    "Runner startup duration from CREATING to RUNNING",
				Buckets: []float64{10, 30, 60, 120, 300, 600},
			},
		),
		RunnerJobDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "shoesvz_runner_job_duration_seconds",
				Help:    "Job execution duration",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200},
			},
		),

		// Agent metrics
		AgentsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "shoesvz_agents_total",
				Help: "Total number of agents by status",
			},
			[]string{"status"},
		),
		AgentsOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_agents_online",
				Help: "Number of online agents",
			},
		),
		AgentsCapacityRunners: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "shoesvz_agents_capacity_runners",
				Help: "Maximum number of runners per agent",
			},
			[]string{"agent_id", "hostname"},
		),
		AgentsCurrentRunners: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "shoesvz_agents_current_runners",
				Help: "Current number of runners per agent",
			},
			[]string{"agent_id", "hostname"},
		),

		// Capacity metrics
		CapacityTotalRunners: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_capacity_total_runners",
				Help: "Total maximum number of runners across all agents",
			},
		),
		CapacityAvailableRunners: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_capacity_available_runners",
				Help: "Number of available runner slots",
			},
		),
		CapacityUtilizationRatio: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_capacity_utilization_ratio",
				Help: "Capacity utilization ratio (0.0-1.0)",
			},
		),

		// Error metrics
		RunnerFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "shoesvz_runner_failures_total",
				Help: "Total number of runner failures by reason",
			},
			[]string{"reason"},
		),
		RunnerErrors: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "shoesvz_runner_errors",
				Help: "Current number of runners in error state",
			},
		),

		// Request metrics
		AddInstanceRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "shoesvz_add_instance_requests_total",
				Help: "Total number of AddInstance requests",
			},
			[]string{"status"},
		),
		DeleteInstanceRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "shoesvz_delete_instance_requests_total",
				Help: "Total number of DeleteInstance requests",
			},
			[]string{"status"},
		),
		AddInstanceDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "shoesvz_add_instance_duration_seconds",
				Help:    "Duration of AddInstance requests",
				Buckets: []float64{10, 30, 60, 120, 300, 600},
			},
		),
	}

	return m
}
