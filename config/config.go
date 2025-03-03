package config

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Pipeline []Pipeline   `yaml:"pipeline"`
	Pprof    *pprofConfig `yaml:"pprof,omitempty"`
	Prom     *promConfig  `yaml:"prometheus,omitempty"`
}

type Pipeline struct {
	Name    string         `yaml:"name"`
	Input   AddrConfig     `yaml:"input"`
	Targets []TargetConfig `yaml:"targets"`
}

type AddrConfig struct {
	Host net.IP `yaml:"host"`
	Port uint16 `yaml:"port"`
}

// type InputConfig struct {
// 	Host net.IP `yaml:"host"`
// 	Port uint16 `yaml:"port"`
// }

type TargetConfig struct {
	Host    net.IP `yaml:"host"`
	Port    uint16 `yaml:"port"`
	SrcHost net.IP `yaml:"src_host,omitempty"`
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

func GetConfig(fileName string) (Config, error) {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Ошибка открытия файла %s: %v", fileName, err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		msg := fmt.Sprintf("Ошибка парсинга файла %s: %v", fileName, err)
		slog.Error(msg)
		return cfg, err
	}
	return cfg, nil
}

type ctxKey string

const PlNameKey ctxKey = "pl_name"
