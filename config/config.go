package config

import (
	"log"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  AddrConfig     `yaml:"server"`
	Targets []TargetConfig `yaml:"targets"`
	Pprof   *pprofConfig   `yaml:"pprof,omitempty"`
	Prom    *promConfig    `yaml:"prometheus,omitempty"`
}

type AddrConfig struct {
	Host net.IP `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type TargetConfig struct {
	Host    net.IP `yaml:"host"`
	Port    uint16 `yaml:"port"`
	SrcPort uint16 `yaml:"src_port,omitempty"`
}

type pprofConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

type promConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

func GetConfig(fileName string) Config {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Ошибка открытия файла %s: %v", fileName, err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalf("Ошибка парсинга файла %s: %v", fileName, err)
	}
	return cfg
}
