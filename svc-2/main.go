package main

import (
	"bytes"
	"encoding/json"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
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
	flag.StringVar(&serverPort, "server_port", "3032", "port on which server runs")
	flag.Parse()

	generalConfig := getConfig(configFilePath)
	tracer, err := jaegersvc.NewTracer(generalConfig)
	if err != nil {
		panic(fmt.Errorf("falied to init jaeger tracer : %v", err))
	}

	router := router.NewRouter(tracer)
	router.Post("/users/register", RegisterHandler(generalConfig, tracer))
	logrus.Infoln(fmt.Sprintf("############################## %s Server Started : %s ##############################", generalConfig.LocalService.Name, serverPort))

	http.ListenAndServe(":"+serverPort, nethttp.Middleware(
		tracer,
		router,
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return r.URL.String()
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

func RegisterHandler(generalConfig *config.GeneralConfig, tracer opentracing.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("get called with method: %s\n", r.Method)

		span := opentracing.SpanFromContext(r.Context())
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			span.SetTag("request_id", reqID)
			log.Printf("request_id: %s\n", reqID)
		}

		span.LogFields(
			olog.String("event", "user service: register called"),
			olog.String("value", "test"),
		)

		var userReg UserRegisterRequestModel
		err := json.NewDecoder(r.Body).Decode(&userReg)
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return 
		}
	
		log.Printf("request payload: %v+\n", userReg)
		span.SetTag("external_id", userReg.ExternalId)
		span.SetTag("email", userReg.Email)
		span.SetTag("device_id", userReg.Device.DeviceId)

		doSomething()

		err = sendNotification(r.Context(), &userReg, generalConfig, tracer)
		if err != nil {
			log.Printf("error: %v+", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	}
}

func doSomething() {
	time.Sleep(time.Duration(utils.GetRandomNumber()) * time.Millisecond)
}

func sendNotification(ctx context.Context, payload *UserRegisterRequestModel, generalConfig *config.GeneralConfig, tracer opentracing.Tracer) error {
	service3Endpoint := fmt.Sprintf("%s:%d", generalConfig.Service3.Host, generalConfig.Service3.Port)
	url := fmt.Sprintf("%s/%s", service3Endpoint, "notification")
	by, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(by))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(tracer, req, nethttp.OperationName(fmt.Sprintf("%s:%s", generalConfig.Service3.Name, url)))
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

type UserRegisterRequestModel struct {
	ExternalId        string             `json:"external_id,omitempty" validate:"regexp=^[A-Z]+\\-\\S+$"`
	FirstName         string             `json:"first_name,omitempty"`
	LastName          string             `json:"last_name,omitempty"`
	Dob               string             `json:"dob,omitempty" validate:"regexp=(^$|^([0-9][0-9]{3})\\-(0[1-9]|1[0-2])\\-(0[1-9]|[12][0-9]|3[01])$)"`
	ImageUrl          string             `json:"image_url,omitempty"`
	Email             string             `json:"email,omitempty" validate:"regexp=(^$|^([A-Za-z0-9_\\-.+])+@([A-Za-z0-9_\\-.])+\\.([A-Za-z]{2\\,})$)"`
	Password          string             `json:"password" validate:"min=8"`
	PhoneNo           string             `json:"phone_no,omitempty" validate:"regexp=(^$|^\\d+$)"`
	ISDCode           string             `json:"isd_code,omitempty" validate:"regexp=(^$|^\\d+$)"`
	Source            string             `json:"source"`
	AddressLine1      string             `json:"address_line_1,omitempty"`
	AddressLine2      string             `json:"address_line_2,omitempty"`
	AddressLine3      string             `json:"address_line_3,omitempty"`
	City              string             `json:"city,omitempty"`
	State             string             `json:"state,omitempty"`
	Country           string             `json:"country,omitempty"`
	PostalCode        string             `json:"postal_code,omitempty"`
	SecurityQuestions []SecurityQuestion `json:"security_questions,omitempty"`
	Device            UserDevice         `json:"device"`
	AuthRequired      bool               `json:"auth_required"`
	LoginDuration     int                `json:"login_duration"`
}

type UserDevice struct {
	Id          string `json:"id,omitempty" bson:"_id,omitempty"`
	UserId      string `json:"user_id" bson:"user_id"`
	DeviceId    string `json:"device_id" bson:"device_id"`
	Imei        string `json:"imei,omitempty" bson:"imei,omitempty"`
	MacId       string `json:"mac_id,omitempty" bson:"mac_id,omitempty"`
	DeviceToken string `json:"device_token,omitempty" bson:"device_token,omitempty"`
	DeviceType  string `json:"device_type,omitempty" bson:"device_type,omitempty"`
	DeviceOS    string `json:"device_os,omitempty" bson:"device_os,omitempty"`
	UserAgent   string `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
	Status      string `json:"status,omitempty" bson:"status,omitempty"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
	UpdatedAt   int64  `json:"updated_at" bson:"updated_at"`
}

type SecurityQuestion struct {
	Question string `json:"question" bson:"question"`
	Answer   string `json:"answer" bson:"answer"`
}