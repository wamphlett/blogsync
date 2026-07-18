package cfg

type Otel struct {
	Endpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName string `env:"OTEL_SERVICE_NAME,default=blogsync"`
}
