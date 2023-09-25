package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/lipaysamart/rds-exporter/pkg/client/service"
	"github.com/lipaysamart/rds-exporter/pkg/config"
)

var ignores = map[string]struct{}{
	"timestamp": {},
	"Maximum":   {},
	"Minimum":   {},
	"Average":   {},
}

var allNamespaces = map[string]string{
	"acs_rds_dashboard": "ApsaraDB RDS",
}

func AllNamespaces() map[string]string {
	return allNamespaces
}

func allNamesOfNamespaces() []string {
	ss := make([]string, 0, len(allNamespaces))
	for name := range allNamespaces {
		ss = append(ss, name)
	}
	return ss
}

type Datapoint map[string]interface{}

func (d Datapoint) Get(measure string) float64 {
	v, ok := d[measure]
	if !ok {
		return 0
	}
	return v.(float64)
}

func (d Datapoint) Labels() []string {
	labels := make([]string, 0)
	for k := range d {
		if _, ok := ignores[k]; !ok {
			labels = append(labels, k)
		}
	}
	sort.Strings(labels)
	return labels
}

func (d Datapoint) Values(labels ...string) []string {
	values := make([]string, 0, len(labels))
	for i := range labels {
		values = append(values, fmt.Sprintf("%s", d[labels[i]]))
	}
	return values
}

type MetricClient struct {
	cms    *cms.Client
	logger log.Logger
}

func NewMetricClient(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (*MetricClient, error) {
	cmsClient, err := cms.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	cmsClient.SetTransport(rt)
	if logger == nil {
		logger = log.NewNopLogger()
	}
	return &MetricClient{cmsClient, logger}, nil
}

func (c *MetricClient) retrive(sub string, name string, period string) ([]Datapoint, error) {
	req := cms.CreateDescribeMetricLastRequest()
	req.Namespace = sub
	req.MetricName = name
	req.Period = period
	resp, err := c.cms.DescribeMetricLast(req)
	if err != nil {
		return nil, err
	}
	var datapoints []Datapoint
	if err = json.Unmarshal([]byte(resp.Datapoints), &datapoints); err != nil {
		level.Debug(c.logger).Log("content", resp.GetHttpContentString(), "error", err)
		return nil, err
	}
	return datapoints, nil
}

func (c *MetricClient) Collect(namespace string, sub string, m *config.Metric, ch chan<- prometheus.Metric) {
	if m.Name == "" {
		level.Warn(c.logger).Log("msg", "metric name must been set")
		return
	}
	datapoints, err := c.retrive(sub, m.Name, m.Period)
	if err != nil {
		level.Error(c.logger).Log("msg", "failed to retrive datapoints", "namespace", sub, "name", m.String(), "error", err)
		return
	}
	for _, dp := range datapoints {
		val := dp.Get(m.Measure)
		ch <- prometheus.MustNewConstMetric(
			m.Desc(namespace, sub, dp.Labels()...),
			prometheus.GaugeValue,
			val,
			dp.Values(m.Dimensions...)...,
		)
	}
}

func (c *MetricClient) DescribeMetricMetaList(namespaces ...string) (map[string][]cms.Resource, error) {
	namespaces = filterNamespaces(namespaces...)
	m := make(map[string][]cms.Resource)
	for _, ns := range namespaces {
		req := cms.CreateDescribeMetricMetaListRequest()
		req.Namespace = ns
		req.PageSize = requests.NewInteger(100)
		resp, err := c.cms.DescribeMetricMetaList(req)
		if err != nil {
			return nil, err
		}
		level.Debug(c.logger).Log("content", resp.GetHttpContentString())
		m[ns] = resp.Resources.Resource
	}
	return m, nil
}

// GenerateExampleConfig create example config
func GenerateExampleConfig(ak, secret, region string, m map[string][]cms.Resource) *config.Config {
	cfg := &config.Config{
		AccessKey:       ak,
		AccessKeySecret: secret,
		Region:          region,
		InstanceInfos:   service.Services(),
		Metrics:         make(map[string][]*config.Metric),
	}
	for ns, resources := range m {
		metrics := make([]*config.Metric, 0, len(resources))
		for _, res := range resources {
			metrics = append(metrics, &config.Metric{
				Name:        res.MetricName,
				Period:      res.Periods,
				Description: res.Description,
				Dimensions:  strings.Split(res.Dimensions, ","),
				Unit:        res.Unit,
				Measure:     res.Statistics,
			})
		}
		cfg.Metrics[ns] = metrics
	}
	return cfg
}

func filterNamespaces(namespaces ...string) []string {
	if len(namespaces) == 0 {
		return allNamesOfNamespaces()
	}
	filters := make([]string, 0)
	for _, ns := range namespaces {
		if ns == "all" {
			return allNamesOfNamespaces()
		}
		if _, ok := allNamespaces[ns]; ok {
			filters = append(filters, ns)
		}
	}
	return filters
}
