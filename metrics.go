package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	withdrawalsClaimed prometheus.Counter
	withdrawalsCounter uint64

	withdrawalAddresses prometheus.Gauge
	addressesCounter    uint64

	registry *prometheus.Registry
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		withdrawalsClaimed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "autoclaimer",
			Name:      "withdrawals_claimed",
			Help:      "Number of withdrawals claimed on the last claim.",
		}),
		withdrawalAddresses: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "autoclaimer",
			Name:      "withdrawal_addresses",
			Help:      "Number of unique withdrawal addresess processed on the last claim.",
		}),
	}
	reg.MustRegister(m.withdrawalsClaimed, m.withdrawalAddresses)
	m.registry = reg
	return m
}

func (m *Metrics) Commit() {
	m.withdrawalsClaimed.Add(float64(m.withdrawalsCounter))
	m.withdrawalsCounter = 0

	m.withdrawalAddresses.Set(float64(m.addressesCounter))
	m.addressesCounter = 0
}

func (m *Metrics) serve(ctx context.Context) {
	promHandler := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	server := &http.Server{Addr: ":9090", Handler: promHandler}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error("metrics serve error: %s", err.Error())
		}
	}()

	// blocks untill parent context will be canceled
	<-ctx.Done()

	// gracefull shutdown
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(c); err != nil {
		log.Error("can't stop metrics server: %s", err.Error())
	}
}
