package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type StatsCollector struct {
	app                *application
	messageCountMetric *prometheus.Desc
}

func newStatsCollector(app *application) *StatsCollector {
	return &StatsCollector{
		messageCountMetric: prometheus.NewDesc(
			"mm_message_count", "The amount of observed messages",
			nil, nil,
		),
		app: app,
	}
}

func (collector *StatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.messageCountMetric
}

func (collector *StatsCollector) Collect(ch chan<- prometheus.Metric) {
	messageCount := collector.app.messagesCount

	ch <- prometheus.NewMetricWithTimestamp(
		time.Now(),
		prometheus.MustNewConstMetric(collector.messageCountMetric, prometheus.CounterValue, float64(messageCount)),
	)
}
