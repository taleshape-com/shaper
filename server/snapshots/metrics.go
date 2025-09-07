package snapshots

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricSnapshotCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shaper_snapshot_total",
			Help: "Total number of snapshots",
		},
		[]string{"status"},
	)

	metricSnapshotTotalDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shaper_snapshot_duration_total_seconds",
			Help:    "Duration of snapshots in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
	)

	metricSqliteSnapshotDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shaper_snapshot_sqlite_duration_seconds",
			Help:    "Duration of SQLite snapshot in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
	)

	metricDuckdbSnapshotDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shaper_snapshot_duckdb_duration_seconds",
			Help:    "Duration of DuckDB snapshot in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
	)
)
