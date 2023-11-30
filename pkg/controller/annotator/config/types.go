package config

// AnnotatorConfiguration holds configuration for a node annotator.
type AnnotatorConfiguration struct {
	PrometheusConfig
	// BindingHeapSize limits the size of Binding Heap, which stores the lastest
	// pod scheduled imformation.
	BindingHeapSize int32
	// ConcurrentSyncs specified the number of annotator controller workers.
	ConcurrentSyncs int32
	// PolicyConfigPath specified the path of Scheduler Policy File.
	PolicyConfigPath string
}

// PrometheusConfig holds configuration for a prometheus client.
type PrometheusConfig struct {
	// PrometheusAddr is the address of Prometheus Service.
	PrometheusAddr string
	// PrometheusUser is the basic auth username of Prometheus Service.
	PrometheusUser string
	// PrometheusPassword is the basic auth password of Prometheus Service.
	PrometheusPassword string
	// PrometheusBearer is the custom bearer auth header of Prometheus Service.
	PrometheusBearer string
	// PrometheusBearerToken is the bearer auth token of Prometheus Service.
	PrometheusBearerToken string
}
