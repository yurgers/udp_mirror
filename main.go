package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"

	"net/http/pprof"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/ipv4"
	"gopkg.in/yaml.v3"
)

type IRPData struct {
	Data []byte
	Src  addrConfig
}

type addrConfig struct {
	Host net.IP `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type targetConfig struct {
	Host net.IP `yaml:"host"`
	Port uint16 `yaml:"port"` 

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

type Config struct {
	Server  addrConfig     `yaml:"server"`
	Targets []targetConfig `yaml:"targets"`
	Pprof   *pprofConfig   `yaml:"pprof,omitempty"`
	Prom    *promConfig    `yaml:"prometheus,omitempty"`
}

type Worker struct {
	target targetConfig
	//local string
	// conn      *net.UDPConn
	// RawConn   *ipv4.RawConn
	sender PacketSender
}

func (w *Worker) ProcessPackets(c <-chan IRPData) {
	listen := "127.0.0.1"
	if w.target.Host.String() != listen {
		listen = ""
	}

	con, err := net.ListenPacket("ip4:udp", listen)
	if err != nil {
		log.Fatalln(err)
	}
	defer con.Close()

	log.Println("Worker запущен:", w.target)

	//new raw packet connection
	rawConn, err := ipv4.NewRawConn(con)
	if err != nil {
		log.Fatalln(err)
	}
	defer rawConn.Close()

	for {
		data, ok := <-c
		if !ok {
			log.Printf("Worker processing data: %v, channel closed\n", data)
			break // exit break loop
		} else {
			if w.target.SrcPort != 0 {
				data.Src.Port = w.target.SrcPort
			}

			dst := addrConfig{
				w.target.Host,
				w.target.Port,
			}

			// fmt.Println("Полученно, ", len(data.Data))
			w.sender.SendPacket(rawConn, data.Data, data.Src, dst)
		}
	}

	log.Println("Worker завершен:", w.target)
}

type WorkerManager struct {
	workers []*Worker
}

func (wm *WorkerManager) AddWorker(w *Worker) {
	wm.workers = append(wm.workers, w)
}

func (wm *WorkerManager) StartWorkers(c chan IRPData) {
	for _, w := range wm.workers {
		go w.ProcessPackets(c)
	}
}

type PacketProcessor interface {
	ProcessPacket(data []byte, src addrConfig)
}

type PacketSender interface {
	SendPacket(rawConn *ipv4.RawConn, data []byte, src addrConfig, dst addrConfig) //error
}

func GetConfig(fileName string) Config {
	f, err := os.Open(fileName)
	if err != nil {
		log.Println("ошибка открытия файла:", fileName)
		log.Fatal(err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Println("ошибка парсинга конфига:", fileName)
		log.Fatal(err)
	}

	log.Printf("Текущий конфиг: %+v\n", cfg)
	return cfg
}

type UDPListener struct {
	conn     *net.UDPConn
	channels []chan IRPData
}

func (l *UDPListener) Listen() {
	for {
		buffer := make([]byte, 65536)
		n, src, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

		receivedPacketsCounter.WithLabelValues(src.IP.String()).Inc()
		bytesReceived.WithLabelValues(src.IP.String()).Add(float64(n))
		// fmt.Printf("Полученны данные от %s, %d\n", src.IP.String(), n)

		l.processData(buffer[:n], src)
	}
}

func (l *UDPListener) processData(data []byte, src *net.UDPAddr) {
	for _, channel := range l.channels {
		channel <- IRPData{
			Data: data,
			Src: addrConfig{
				Host: src.IP,
				Port: uint16(src.Port),
			},
		}

		// fmt.Println("данных, ", len(data), "переданы в ", channel)
	}
}

func (l *UDPListener) Shutdown() {
	for _, channel := range l.channels {
		close(channel)
	}
}

type UDPSender struct {
}

var id uint16

func (s *UDPSender) SendPacket(rawCon *ipv4.RawConn, data []byte, src addrConfig, dst addrConfig) {
	id += 1
	mtu := 1480 //1500 - 20 (ip) - 8 (udp)

	// src.IP = net.IPv4(192, 168, 1, 78)

	udpHeader := []byte{
		byte(src.Port >> 8), byte(src.Port), // Src порт
		byte(dst.Port >> 8), byte(dst.Port), // Dst порт
		byte((len(data) + 8) >> 8), byte(len(data) + 8), // Размер пакета (заголовок + данные)
		byte(0), byte(0), // Подставляем 0, для иннорирования контрольной сумму
	}

	iph := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0x00,
		TotalLen: ipv4.HeaderLen + len(data) + 8,
		ID:       int(id),
		//Flags:    ipv4.DontFragment,
		FragOff:  0,
		TTL:      64,
		Protocol: 17,
		Src:      src.Host.To4(),
		Dst:      dst.Host.To4(),
	}

	buffer := append(udpHeader, data...)

	if len(buffer) <= mtu {
		err := rawCon.WriteTo(iph, buffer, nil)
		if err != nil {
			log.Fatal("WriteTo: ", err)
		}

		recipient := fmt.Sprintf("%s:%d", dst.Host.String(), dst.Port)
		sentPacketsCounter.WithLabelValues(recipient).Inc()
		bytesSent.WithLabelValues(recipient).Add(float64(len(data)))

		// fmt.Println(iph)
		// fmt.Println("=====================")

	} else {
		// fmt.Println("MTU привешен", len(data), "len= ", len(buffer))
		fragOff := 0

		for {
			// fmt.Println("fragOff: ", fragOff)

			if len(buffer) > mtu {
				fragmet := buffer[:mtu]
				// fmt.Println("len fragmet ", len(fragmet))
				buffer = buffer[mtu:]
				// fmt.Println("len new  buffer", len(buffer))
				iph.Flags = ipv4.MoreFragments
				iph.TotalLen = ipv4.HeaderLen + len(fragmet)
				iph.FragOff = fragOff
				fragOff += len(fragmet) / 8

				// fmt.Printf("buffer: %v\n", buffer)

				err := rawCon.WriteTo(iph, fragmet, nil)
				if err != nil {
					log.Fatal("WriteTo: ", err)
				}

			} else {
				iph.Flags = 0
				iph.FragOff = fragOff
				iph.TotalLen = ipv4.HeaderLen + len(buffer)
				// fmt.Println("send finish buffer ", len(buffer))
				// fmt.Println("FragOff finish", iph.FragOff)

				err := rawCon.WriteTo(iph, buffer, nil)
				if err != nil {
					log.Fatal("WriteTo: ", err)
				}

				// fmt.Println(iph)
				// fmt.Println("=====================")
				return

			}

			// fmt.Println(iph)
			// fmt.Println("-------------------")

		}

	}

	// fmt.Printf("\t > Переданны данные в %s:%d, %d\n", dst.IP.String(), dst.Port, len(data))

}

func NewUDPListener(config *Config, channels []chan IRPData) (*UDPListener, error) {
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Server.Host.String(), config.Server.Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}

	return &UDPListener{
		conn:     conn,
		channels: channels,
		// config:   config,
	}, nil
}

var (
	receivedPacketsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_packets_total",
			Help: "Total number of received packets",
		},
		[]string{"sender"},
	)

	bytesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_bytes_total",
			Help: "Total number of received bytes",
		},
		[]string{"sender"},
	)

	sentPacketsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_packets_total",
			Help: "Total number of sent packets",
		},
		[]string{"recipient"},
	)

	bytesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_bytes_total",
			Help: "Total number of sent bytes",
		},
		[]string{"recipient"},
	)
)

func promRegister() {
	// Регистрация метрик
	prometheus.MustRegister(receivedPacketsCounter)
	prometheus.MustRegister(bytesReceived)
	prometheus.MustRegister(sentPacketsCounter)
	prometheus.MustRegister(bytesSent)

}

func main() {
	configFilePtr := flag.String("f", "config.yml", "Path to the config file")
	backgroundFlag := flag.Bool("d", false, "Run in background mode")
	flag.Parse()
	configFile := *configFilePtr

	cfg := GetConfig(configFile)
	fmt.Println(cfg)

	if *backgroundFlag {
		fmt.Println("Running in background mode")

		// Запуск программы в фоновом режиме
		fmt.Println("Args: ", flag.Args())
		cmdArgs := append([]string{"-f", *configFilePtr}, flag.Args()...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Start()

		// Завершение текущей программы
		os.Exit(0)
	}

	promRegister()

	// pprof
	if cfg.Pprof.Enabled {
		go func() {
			r := http.NewServeMux()

			// Регистрация pprof-обработчиков
			r.HandleFunc("/debug/pprof/", pprof.Index)
			r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			r.HandleFunc("/debug/pprof/profile", pprof.Profile)
			r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			r.HandleFunc("/debug/pprof/trace", pprof.Trace)

			err := http.ListenAndServe(cfg.Pprof.Listen, r)

			if err != nil {
				log.Fatal("pprof listen failed", err)
			}
		}()
	}

	// prometheus
	if cfg.Prom.Enabled {
		go func() {
			//http.Handle("/metrics", promhttp.Handler())
			err := http.ListenAndServe(cfg.Prom.Listen, promhttp.Handler())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("prometheus start Listen: %s\n", cfg.Prom.Listen)
		}()
	}

	channels := make([]chan IRPData, len(cfg.Targets))

	listener, err := NewUDPListener(&cfg, channels)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.conn.Close()

	workerManager := &WorkerManager{}
	for i, target := range cfg.Targets {
		channels[i] = make(chan IRPData, 10)
		worker := &Worker{
			target: target,
			sender: &UDPSender{},
		}
		workerManager.AddWorker(worker)
	}

	var wg sync.WaitGroup
	for i, w := range workerManager.workers {
		wg.Add(1)
		go func(worker *Worker, ch <-chan IRPData) {
			defer wg.Done()
			worker.ProcessPackets(ch)
			fmt.Println("stop worker.ProcessPackets")
		}(w, channels[i])
	}

	fmt.Println("Сервер запущен и слушает на", listener.conn.LocalAddr())
	go func() {
		listener.Listen()
		defer listener.Shutdown()
	}()

	wg.Wait()
}
