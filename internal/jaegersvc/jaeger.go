package jaegersvc

import (
	"log"

	"github.com/kyawmyintthein/jaeger-chi-demo/config"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jconfig "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics"
)

func NewTracer(generalConfig *config.GeneralConfig) (opentracing.Tracer, error) {
	cfg, err := jconfig.FromEnv()
	if err != nil {
		log.Fatal(err)
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
