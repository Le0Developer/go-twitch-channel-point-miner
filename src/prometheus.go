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

func NewPrometheusExporter(miner *Miner) (*PrometheusExporter, error) {
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

	if err := prometheus.Register(exporter.streamerPoints); err != nil {
		return nil, fmt.Errorf("failed to register streamerPoints: %w", err)
	}
	if err := prometheus.Register(exporter.streamerViewers); err != nil {
		return nil, fmt.Errorf("failed to register streamerViewers: %w", err)
	}
	if err := prometheus.Register(exporter.streamerLiveStatus); err != nil {
		return nil, fmt.Errorf("failed to register streamerLiveStatus: %w", err)
	}
	if err := prometheus.Register(exporter.totalStreamers); err != nil {
		return nil, fmt.Errorf("failed to register totalStreamers: %w", err)
	}
	if err := prometheus.Register(exporter.totalUsers); err != nil {
		return nil, fmt.Errorf("failed to register totalUsers: %w", err)
	}

	return exporter, nil
}

func (e *PrometheusExporter) Unregister() {
	prometheus.Unregister(e.streamerPoints)
	prometheus.Unregister(e.streamerViewers)
	prometheus.Unregister(e.streamerLiveStatus)
	prometheus.Unregister(e.totalStreamers)
	prometheus.Unregister(e.totalUsers)
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

		// Update live status
		liveStatus := 0.0
		if streamer.IsLive() {
			liveStatus = 1.0
			// Update viewer count
			e.streamerViewers.WithLabelValues(
				streamer.Username,
				streamer.ID,
			).Set(float64(streamer.Viewers))
		} else {
			// Update viewer count to 0 if not live
			e.streamerViewers.WithLabelValues(
				streamer.Username,
				streamer.ID,
			).Set(0)
		}
		e.streamerLiveStatus.WithLabelValues(
			streamer.Username,
			streamer.ID,
		).Set(liveStatus)
	}
}

func (e *PrometheusExporter) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func (e *PrometheusExporter) StartServer(host string, port int) error {
	e.UpdateMetrics()

	handler := e.Handler()

	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Prometheus exporter starting on %s\n", addr)
	if host == "" || host == "0.0.0.0" {
		fmt.Printf("Metrics available at http://localhost:%d/metrics (listening on all interfaces)\n", port)
	} else {
		fmt.Printf("Metrics available at http://%s/metrics\n", addr)
	}

	return http.ListenAndServe(addr, handler)
}
