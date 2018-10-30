package config

type Service struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type GeneralConfig struct {
	LocalService Service
	Service1     Service
	Service2     Service
	Service3     Service
	Jaeger       struct {
		EndPoint string `json:"endpoint"`
	}
}
