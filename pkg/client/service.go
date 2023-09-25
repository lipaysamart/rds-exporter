package client

import (
	"net/http"

	"github.com/go-kit/log"
	"github.com/lipaysamart/rds-exporter/pkg/client/service"
	"github.com/prometheus/client_golang/prometheus"
)

type ServiceClient struct {
	collectors map[string]service.Collector
}

func (c *ServiceClient) Collect(namespace string, sub string, ch chan<- prometheus.Metric) {
	collector, ok := c.collectors[sub]
	if !ok {
		return
	}
	collector.Collect(namespace, ch)
}

func NewServiceClient(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (*ServiceClient, error) {
	sc := &ServiceClient{
		collectors: make(map[string]service.Collector),
	}
	if logger == nil {
		logger = log.NewNopLogger()
	}
	for name, fn := range service.CollectorFunc() {
		collector, err := fn(ak, secret, region, rt, logger)
		if err != nil {
			return nil, err
		}
		sc.collectors[name] = collector
	}
	return sc, nil
}
