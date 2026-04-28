// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"shaper/server/core"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

var (
	queryDurationHistogram *prometheus.HistogramVec
	queryStatusCounter     *prometheus.CounterVec
	activeQueriesGauge     *prometheus.GaugeVec
	slowQueriesCounter     *prometheus.CounterVec

	metricsInitOnce sync.Once
	metricsRegistry sync.Once
)

func Init() {
	metricsInitOnce.Do(func() {
		initQueryMetrics()
		initSystemMetrics()
	})
}

func initQueryMetrics() {
	queryDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shaper_query_duration_seconds",
			Help:    "Duration of SQL queries in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		},
		[]string{"query_type", "status"},
	)
	safeRegister(queryDurationHistogram)

	queryStatusCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shaper_queries_total",
			Help: "Total number of queries executed",
		},
		[]string{"query_type", "status"},
	)
	safeRegister(queryStatusCounter)

	activeQueriesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shaper_active_queries",
			Help: "Number of currently active queries",
		},
		[]string{"query_type"},
	)
	safeRegister(activeQueriesGauge)

	slowQueriesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shaper_slow_queries_total",
			Help: "Total number of slow queries (>= 1 second)",
		},
		[]string{"query_type"},
	)
	safeRegister(slowQueriesCounter)

	tracker := core.GetQueryTracker()
	tracker.SetOnUpdate(func(exec *core.QueryExecution) {
		updateMetrics(exec)
	})
}

func safeRegister(collector prometheus.Collector) {
	metricsRegistry.Do(func() {})
	err := prometheus.Register(collector)
	if err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return
		}
		panic(err)
	}
}

func updateMetrics(exec *core.QueryExecution) {
	if exec == nil {
		return
	}

	queryType := string(exec.Type)
	status := string(exec.Status)

	if exec.Status == core.QueryStatusRunning {
		activeQueriesGauge.WithLabelValues(queryType).Inc()
		return
	}

	if !exec.IsTerminal() {
		return
	}

	activeQueriesGauge.WithLabelValues(queryType).Dec()

	queryStatusCounter.WithLabelValues(queryType, status).Inc()

	if exec.DurationMs != nil {
		durationSeconds := float64(*exec.DurationMs) / 1000.0
		queryDurationHistogram.WithLabelValues(queryType, status).Observe(durationSeconds)
	}

	if exec.IsSlowQuery {
		slowQueriesCounter.WithLabelValues(queryType).Inc()
	}
}

type systemCollector struct {
	diskSpace    *prometheus.Desc
	memoryMetric *prometheus.Desc
	cpuUsage     *prometheus.Desc
}

func (collector *systemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.diskSpace
	ch <- collector.memoryMetric
	ch <- collector.cpuUsage
}

func (collector *systemCollector) Collect(ch chan<- prometheus.Metric) {
	if usage, err := disk.Usage("/"); err == nil {
		ch <- prometheus.MustNewConstMetric(
			collector.diskSpace,
			prometheus.GaugeValue,
			float64(usage.Total),
			"/", "total",
		)
		ch <- prometheus.MustNewConstMetric(
			collector.diskSpace,
			prometheus.GaugeValue,
			float64(usage.Used),
			"/", "used",
		)
	}

	if vmstat, err := mem.VirtualMemory(); err == nil {
		ch <- prometheus.MustNewConstMetric(
			collector.memoryMetric,
			prometheus.GaugeValue,
			float64(vmstat.Total),
			"total",
		)
		ch <- prometheus.MustNewConstMetric(
			collector.memoryMetric,
			prometheus.GaugeValue,
			float64(vmstat.Available),
			"available",
		)
		ch <- prometheus.MustNewConstMetric(
			collector.memoryMetric,
			prometheus.GaugeValue,
			float64(vmstat.Used),
			"used",
		)
	}

	if cpuPercentage, err := cpu.Percent(0, false); err == nil {
		ch <- prometheus.MustNewConstMetric(
			collector.cpuUsage,
			prometheus.GaugeValue,
			cpuPercentage[0],
		)
	}
}

func initSystemMetrics() {
	collector := &systemCollector{
		diskSpace: prometheus.NewDesc(
			"system_disk_space_bytes",
			"Available disk space in bytes",
			[]string{"path", "type"},
			nil,
		),
		memoryMetric: prometheus.NewDesc(
			"system_memory_bytes",
			"System memory usage in bytes",
			[]string{"type"},
			nil,
		),
		cpuUsage: prometheus.NewDesc(
			"system_cpu_usage_percent",
			"Current CPU usage percentage",
			nil,
			nil,
		),
	}
	safeRegister(collector)
}
