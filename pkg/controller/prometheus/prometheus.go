package prometheus

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane-scheduler/pkg/utils"
)

const (
	DefaultPrometheusQueryTimeout = 10 * time.Second
)

// PromClient provides client to interact with Prometheus.
type PromClient interface {
	// QueryByNodeIP queries data by node IP.
	QueryByNodeIP(string, string) (string, error)
	// QueryByNodeName queries data by node IP.
	QueryByNodeName(string, string) (string, error)
	// QueryByNodeIPWithOffset queries data by node IP with offset.
	QueryByNodeIPWithOffset(string, string, string) (string, error)
}

type promClient struct {
	API v1.API
}

// NewPromClient returns PromClient interface.
func NewPromClient(addr string) (PromClient, error) {
	config := api.Config{
		Address: addr,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &promClient{
		API: v1.NewAPI(client),
	}, nil
}

func (p *promClient) QueryByNodeIP(metricName, ip string) (string, error) {
	klog.V(4).Infof("Try to query %s by node IP[%s]", metricName, ip)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"} /100", metricName, ip)

	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	if utils.IsValidIPv6(ip) {
		querySelector = fmt.Sprintf("%s{instance=~\"\\\\[%s\\\\]:.+\"} /100", metricName, ip)
	} else {
		querySelector = fmt.Sprintf("%s{instance=~\"%s:.+\"} /100", metricName, ip)
	}

	result, err = p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) QueryByNodeName(metricName, name string) (string, error) {
	klog.V(4).Infof("Try to query %s by node name[%s]", metricName, name)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"} /100", metricName, name)

	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) QueryByNodeIPWithOffset(metricName, ip, offset string) (string, error) {
	klog.V(4).Info("Try to query %s with offset %s by node IP[%s]", metricName, offset, ip)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"} offset %s /100", metricName, ip, offset)
	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	querySelector = fmt.Sprintf("%s{instance=~\"%s:.+\"} offset %s /100", metricName, ip, offset)
	result, err = p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) query(query string) (string, error) {
	klog.V(4).Infof("Begin to query prometheus by promQL [%s]...", query)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPrometheusQueryTimeout)
	defer cancel()

	result, warnings, err := p.API.Query(ctx, query, time.Now())
	if err != nil {
		return "", err
	}

	if len(warnings) > 0 {
		return "", fmt.Errorf("unexpected warnings: %v", warnings)
	}

	if result.Type() != model.ValVector {
		return "", fmt.Errorf("illege result type: %v", result.Type())
	}

	var metricValue string
	for _, elem := range result.(model.Vector) {
		if float64(elem.Value) < float64(0) || math.IsNaN(float64(elem.Value)) {
			elem.Value = 0
		}
		metricValue = strconv.FormatFloat(float64(elem.Value), 'f', 5, 64)
	}

	return metricValue, nil
}
