package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/kyawmyintthein/zipkin-chi-demo/config"
	"github.com/kyawmyintthein/zipkin-chi-demo/internal/utils"
	"github.com/kyawmyintthein/zipkin-chi-demo/internal/zipkinsvc"
	"github.com/kyawmyintthein/zipkin-chi-demo/router"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
)

func main() {
	var configFilePath string
	var serverPort string
	flag.StringVar(&configFilePath, "config", "config.json", "absolute path to the configuration file")
	flag.StringVar(&serverPort, "server_port", "8082", "port on which server runs")
	flag.Parse()

	generalConfig := getConfig(configFilePath)
	tracer, err := zipkinsvc.NewTracer(generalConfig)

	if err != nil {
		panic(fmt.Errorf("falied to init zipkin tracer : %v", err))
	}

	zipkinClient, err := zipkinsvc.NewClient(tracer)
	if err != nil {
		panic(fmt.Errorf("falied to init zipkin tracer : %v", err))
	}

	router := router.GetRouter(tracer)
	router.Post("/foo", Post(zipkinClient, generalConfig))

	logrus.Infoln(fmt.Sprintf("############################## %s Server Started ##############################", generalConfig.LocalService.Name))
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

func Post(client *zipkinhttp.Client, generalConfig *config.GeneralConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)

		// retrieve span from context (created by server middleware)
		span := zipkin.SpanFromContext(r.Context())
		span.Tag(fmt.Sprintf("%s-called", generalConfig.LocalService.Name), time.Now().String())
		doSomething()
		span.Annotate(time.Now(), fmt.Sprintf("%s-done", generalConfig.LocalService.Name))
		err := callService3(r.Context(), generalConfig, client)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func doSomething() {
	time.Sleep(time.Duration(utils.GetRandomNumber()) * time.Millisecond)
}

func callService3(ctx context.Context, generalConfig *config.GeneralConfig, client *zipkinhttp.Client) error {
	service2Endpoint := fmt.Sprintf("%s:%d", generalConfig.Service3.Host, generalConfig.Service3.Port)
	url := fmt.Sprintf("%s/%s", service2Endpoint, "bar")
	newRequest, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	newRequest = newRequest.WithContext(ctx)
	res, err := client.DoWithAppSpan(newRequest, url)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}
