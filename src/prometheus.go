package miner

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusExporter struct {
	miner *Miner

	streamerPoints     *prometheus.GaugeVec
	streamerViewers    *prometheus.GaugeVec
	streamerLiveStatus *prometheus.GaugeVec
	totalStreamers     prometheus.Gauge
	totalUsers         prometheus.Gauge
}

func NewPrometheusExporter(miner *Miner) *PrometheusExporter {
	exporter := &PrometheusExporter{
		miner: miner,

		streamerPoints: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "twitch_channel_points",
				Help: "Current channel points for each user-streamer combination",
			},
			[]string{"username", "streamer", "streamer_id"},
		),

		streamerViewers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "twitch_streamer_viewers",
				Help: "Current viewer count for each streamer",
			},
			[]string{"streamer", "streamer_id"},
		),

		streamerLiveStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "twitch_streamer_live",
				Help: "Whether a streamer is currently live (1) or not (0)",
			},
			[]string{"streamer", "streamer_id"},
		),

		totalStreamers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "twitch_total_streamers",
				Help: "Total number of streamers being monitored",
			},
		),

		totalUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "twitch_total_users",
				Help: "Total number of users configured",
			},
		),
	}

	prometheus.MustRegister(exporter.streamerPoints)
	prometheus.MustRegister(exporter.streamerViewers)
	prometheus.MustRegister(exporter.streamerLiveStatus)
	prometheus.MustRegister(exporter.totalStreamers)
	prometheus.MustRegister(exporter.totalUsers)

	return exporter
}

func (e *PrometheusExporter) UpdateMetrics() {
	e.miner.Lock.Lock()
	defer e.miner.Lock.Unlock()

	// Update total counts
	e.totalStreamers.Set(float64(len(e.miner.Streamers)))
	e.totalUsers.Set(float64(len(e.miner.Users)))

	for _, streamer := range e.miner.Streamers {
		// Update points for each user-streamer combination
		for user, points := range streamer.Points {
			e.streamerPoints.WithLabelValues(
				user.Username,
				streamer.Username,
				streamer.ID,
			).Set(float64(points))
		}

		// Update viewer count
		e.streamerViewers.WithLabelValues(
			streamer.Username,
			streamer.ID,
		).Set(float64(streamer.Viewers))

		// Update live status
		liveStatus := 0.0
		if streamer.IsLive() {
			liveStatus = 1.0
		}
		e.streamerLiveStatus.WithLabelValues(
			streamer.Username,
			streamer.ID,
		).Set(liveStatus)
	}
}

func (e *PrometheusExporter) StartServer(port int) error {
	e.UpdateMetrics()

	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Prometheus exporter starting on port %d\n", port)
	fmt.Printf("Metrics available at http://localhost:%d/metrics\n", port)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
