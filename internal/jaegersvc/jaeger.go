package jaegersvc

import (
	//"log"

	"github.com/kyawmyintthein/jaeger-chi-demo/config"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jconfig "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics"
)

func NewTracer(generalConfig *config.GeneralConfig) (opentracing.Tracer, error) {
	cfg := &jconfig.Configuration{
		Sampler: &jconfig.SamplerConfig{
			Type:              "const",
			Param:             1,
			SamplingServerURL: "10.30.1.136:5778",
		},
		Reporter: &jconfig.ReporterConfig{
			LogSpans:           true,
			CollectorEndpoint:  "http://10.30.1.136:14268/api/traces",
			LocalAgentHostPort: "10.30.1.136:6831",
		},
	}
	cfg.ServiceName = generalConfig.LocalService.Name
	cfg.Sampler.Type = "const"
	cfg.Sampler.Param = 1
	var metricsFactory metrics.Factory
	//metricsFactory.Namespace(generalConfig.LocalService.Name, nil)
	tracer, _, err := cfg.NewTracer(
		jconfig.Logger(jaeger.StdLogger),
		//	jconfig.Metrics(metricsFactory),
		jconfig.Observer(rpcmetrics.NewObserver(metricsFactory, rpcmetrics.DefaultNameNormalizer)),
	)
	return tracer, err
}
