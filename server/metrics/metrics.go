package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// Create and register system metrics collector
type systemCollector struct {
	diskSpace    *prometheus.Desc
	memoryMetric *prometheus.Desc
	cpuUsage     *prometheus.Desc
}

// Implement collector interface
func (collector *systemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.diskSpace
	ch <- collector.memoryMetric
	ch <- collector.cpuUsage
}

func (collector *systemCollector) Collect(ch chan<- prometheus.Metric) {
	// Collect disk metrics
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

	// Collect memory metrics
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

	// Collect CPU metrics
	if cpuPercentage, err := cpu.Percent(0, false); err == nil {
		ch <- prometheus.MustNewConstMetric(
			collector.cpuUsage,
			prometheus.GaugeValue,
			cpuPercentage[0],
		)
	}
}

func Init() {
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
	prometheus.MustRegister(collector)
}
