package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/kyawmyintthein/jaeger-chi-demo/config"
	"github.com/kyawmyintthein/jaeger-chi-demo/internal/jaegersvc"
	"github.com/kyawmyintthein/jaeger-chi-demo/internal/utils"
	"github.com/kyawmyintthein/jaeger-chi-demo/router"
)

func main() {
	var configFilePath string
	var serverPort string
	flag.StringVar(&configFilePath, "config", "config.json", "absolute path to the configuration file")
	flag.StringVar(&serverPort, "server_port", "3033", "port on which server runs")
	flag.Parse()

	generalConfig := getConfig(configFilePath)
	tracer, err := jaegersvc.NewTracer(generalConfig)
	if err != nil {
		panic(fmt.Errorf("falied to init zipkin tracer : %v", err))
	}

	router := router.NewServeMux(tracer)
	router.Handle("/bar", http.HandlerFunc(Bar(generalConfig, tracer)))
	logrus.Infoln(fmt.Sprintf("############################## %s Server Started : %s ##############################", generalConfig.LocalService.Name, serverPort))
	http.ListenAndServe(":"+serverPort, router)
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

func Bar(generalConfig *config.GeneralConfig, tracer opentracing.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)

		span := opentracing.SpanFromContext(r.Context())
		span.SetTag(fmt.Sprintf("%s-called", generalConfig.LocalService.Name), time.Now())
		doSomething()
		span.SetTag(fmt.Sprintf("%s-done", generalConfig.LocalService.Name), time.Now())
	}
}

func doSomething() {
	time.Sleep(time.Duration(utils.GetRandomNumber()) * time.Millisecond)
}
