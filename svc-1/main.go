package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	_ "net/http/pprof"

	"github.com/kyawmyintthein/jaeger-chi-demo/config"
	"github.com/kyawmyintthein/jaeger-chi-demo/internal/jaegersvc"
	"github.com/kyawmyintthein/jaeger-chi-demo/internal/utils"
	"github.com/kyawmyintthein/jaeger-chi-demo/router"
)

func main() {
	var configFilePath string
	var serverPort string
	flag.StringVar(&configFilePath, "config", "config.json", "absolute path to the configuration file")
	flag.StringVar(&serverPort, "server_port", "3031", "port on which server runs")
	flag.Parse()

	generalConfig := getConfig(configFilePath)
	tracer, err := jaegersvc.NewTracer(generalConfig)
	if err != nil {
		panic(fmt.Errorf("falied to init zipkin tracer : %v", err))
	}

	router := router.NewRouter(tracer)
	router.Get("/users/{user_id}", Get(generalConfig, tracer))
	logrus.Infoln(fmt.Sprintf("############################## %s Server Started : %s ##############################", generalConfig.LocalService.Name, serverPort))

	http.ListenAndServe(":"+serverPort, nethttp.Middleware(
		tracer,
		router,
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "request"
		})))
}

func getConfig(filepath string) *config.GeneralConfig {
	viper.SetConfigFile(filepath)
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load config file : %v \n", err))
	}

	generalConfig := &config.GeneralConfig{}
	viper.Unmarshal(generalConfig)
	return generalConfig
}

func Get(generalConfig *config.GeneralConfig, tracer opentracing.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)
		span := opentracing.SpanFromContext(r.Context())
		userID := chi.URLParam(r, "user_id")
		if userID != "" {
			span.SetTag("user_id", userID)
			log.Printf("user_id: %s\n", userID)
		}

		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			span.SetTag("request_id", reqID)
			log.Printf("request_id: %s\n", reqID)
		}
		span.SetBaggageItem("getBaggageItem", "BaggageItem")
		span.LogKV("test")
		span.LogFields(
			olog.String("event", "svc-1:get called"),
			olog.String("value", "test"),
		)
		span.SetTag(fmt.Sprintf("%s-called", generalConfig.LocalService.Name), time.Now())
		doSomething()
		span.SetTag(fmt.Sprintf("%s-done", generalConfig.LocalService.Name), time.Now())

		err := callService2(r.Context(), generalConfig, tracer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = callService3(r.Context(), generalConfig, tracer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	}
}

func doSomething() {
	time.Sleep(time.Duration(utils.GetRandomNumber()) * time.Millisecond)
}

func callService2(ctx context.Context, generalConfig *config.GeneralConfig, tracer opentracing.Tracer) error {
	service2Endpoint := fmt.Sprintf("%s:%d", generalConfig.Service2.Host, generalConfig.Service2.Port)
	url := fmt.Sprintf("%s/%s", service2Endpoint, "foo")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req, httpTracer := nethttp.TraceRequest(tracer, req, nethttp.OperationName("svc2: "+service2Endpoint))
	defer httpTracer.Finish()

	client := http.Client{Transport: &nethttp.Transport{}}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	//decoder := json.NewDecoder(res.Body)
	return nil
}

func callService3(ctx context.Context, generalConfig *config.GeneralConfig, tracer opentracing.Tracer) error {
	service3Endpoint := fmt.Sprintf("%s:%d", generalConfig.Service3.Host, generalConfig.Service3.Port)
	url := fmt.Sprintf("%s/%s", service3Endpoint, "bar")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(tracer, req, nethttp.OperationName("svc3: "+service3Endpoint))
	defer ht.Finish()

	client := http.Client{Transport: &nethttp.Transport{}}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	//decoder := json.NewDecoder(res.Body)
	return nil
}
