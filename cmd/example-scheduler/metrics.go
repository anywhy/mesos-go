package main

import (
	"net"
	"net/http"
	"strconv"
	"time"

	schedmetrics "github.com/mesos/mesos-go/cmd/example-scheduler/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func initMetrics(cfg config) *metricsAPI {
	schedmetrics.Register()
	metricsAddress := net.JoinHostPort(cfg.server.address, strconv.Itoa(cfg.metrics.port))
	http.Handle(cfg.metrics.path, prometheus.Handler())
	api := newMetricsAPI()
	go forever("api-server", cfg.jobRestartDelay, api.jobStartCount, func() error { return http.ListenAndServe(metricsAddress, nil) })
	return api
}

type metricCounter func(...string)
type metricAdder func(float64, ...string)
type metricWatcher func(float64, ...string)

func (a metricAdder) Int(x int, s ...string) {
	a(float64(x), s...)
}

func newMetricAdder(m prometheus.Counter) metricAdder {
	return func(x float64, _ ...string) { m.Add(float64(x)) }
}

func newMetricCounter(m prometheus.Counter) metricCounter {
	return func(_ ...string) { m.Inc() }
}

func newMetricCounters(m *prometheus.CounterVec) metricCounter {
	return func(s ...string) { m.WithLabelValues(s...).Inc() }
}

func newMetricWatcher(m prometheus.Summary) metricWatcher {
	return func(x float64, _ ...string) { m.Observe(x) }
}

func newMetricWatchers(m *prometheus.SummaryVec) metricWatcher {
	return func(x float64, s ...string) { m.WithLabelValues(s...).Observe(x) }
}

// Since records an observation of time.Now().Sub(t) in microseconds
func (w metricWatcher) Since(t time.Time) {
	w(schedmetrics.InMicroseconds(time.Now().Sub(t)))
}

type metricsAPI struct {
	subscriptionAttempts  metricCounter
	apiErrorCount         metricCounter
	errorsReceived        metricCounter
	updatesReceived       metricCounter
	failuresReceived      metricCounter
	subscribedReceived    metricCounter
	offersReceived        metricAdder
	offersDeclined        metricAdder
	reviveCount           metricCounter
	tasksLaunched         metricAdder
	tasksFinished         metricCounter
	processOffersLatency  metricWatcher
	launchesPerOfferCycle metricWatcher
	offeredResources      metricWatcher
	jobStartCount         metricCounter
	artifactDownloads     metricCounter
}

func newMetricsAPI() *metricsAPI {
	return &metricsAPI{
		subscriptionAttempts:  newMetricCounter(schedmetrics.SubscriptionAttempts),
		apiErrorCount:         newMetricCounters(schedmetrics.APIErrorCount),
		errorsReceived:        newMetricCounter(schedmetrics.ErrorsReceived),
		failuresReceived:      newMetricCounter(schedmetrics.FailuresReceived),
		updatesReceived:       newMetricCounter(schedmetrics.UpdatesReceived),
		subscribedReceived:    newMetricCounter(schedmetrics.SubscribedReceived),
		offersReceived:        newMetricAdder(schedmetrics.OffersReceived),
		offersDeclined:        newMetricAdder(schedmetrics.OffersDeclined),
		reviveCount:           newMetricCounter(schedmetrics.ReviveCount),
		tasksLaunched:         newMetricAdder(schedmetrics.TasksLaunched),
		tasksFinished:         newMetricCounter(schedmetrics.TasksFinished),
		processOffersLatency:  newMetricWatcher(schedmetrics.ProcessOffersLatency),
		launchesPerOfferCycle: newMetricWatcher(schedmetrics.TasksLaunchedPerOfferCycle),
		offeredResources:      newMetricWatchers(schedmetrics.OfferedResources),
		jobStartCount:         newMetricCounters(schedmetrics.JobStartCount),
		artifactDownloads:     newMetricCounter(schedmetrics.ArtifactDownloads),
	}
}