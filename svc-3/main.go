package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"

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
		panic(fmt.Errorf("falied to init jaeger tracer : %v", err))
	}

	router := router.NewRouter()
	router.Post("/notification", NotificationHandler(generalConfig, tracer))
	logrus.Infoln(fmt.Sprintf("############################## %s Server Started : %s ##############################", generalConfig.LocalService.Name, serverPort))

	http.ListenAndServe(":"+serverPort, nethttp.Middleware(
		tracer,
		router,
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return r.URL.String()
		})))
	//http.ListenAndServe(":"+serverPort, router)
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

func NotificationHandler(generalConfig *config.GeneralConfig, tracer opentracing.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)

		span := opentracing.SpanFromContext(r.Context())
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			span.SetTag("request_id", reqID)
			span.SetTag("original_service", span.BaggageItem("original_service"))
			log.Printf("request_id: %s\n", reqID)
		}

		var sendNotificationRequest NotificationRequest
		err := json.NewDecoder(r.Body).Decode(&sendNotificationRequest)
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		span.LogFields(
			olog.String("event", "notification service: send called"),
			olog.Object("payload", sendNotificationRequest),
		)
		log.Printf("request payload: %v+\n", sendNotificationRequest)
		span.SetTag("email", sendNotificationRequest.Email)
		w.WriteHeader(http.StatusOK)

	}
}

func NotificationHandlerWithoutTracer(generalConfig *config.GeneralConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)

		w.WriteHeader(http.StatusOK)

	}
}

func doSomething() {
	time.Sleep(time.Duration(utils.GetRandomNumber()) * time.Millisecond)
}

type NotificationRequest struct {
	Email string `json:"email,omitempty" validate:"regexp=(^$|^([A-Za-z0-9_\\-.+])+@([A-Za-z0-9_\\-.])+\\.([A-Za-z]{2\\,})$)"`
}
