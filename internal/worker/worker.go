// internal/worker/worker.go
package worker

import (
	"log"
	"net"
	"udp_mirror/config"

	"golang.org/x/net/ipv4"
)

type IRPData struct {
	Data []byte
	Src  config.AddrConfig
}

type Worker struct {
	Target config.TargetConfig
	Sender PacketSender
}

type PacketSender interface {
	SendPacket(rawConn *ipv4.RawConn, data []byte, src config.AddrConfig, dst config.AddrConfig)
}

func (w *Worker) ProcessPackets(c <-chan IRPData) {
	listen := "127.0.0.1"
	if w.Target.Host.String() != listen {
		listen = ""
	}

	con, err := net.ListenPacket("ip4:udp", listen)
	if err != nil {
		log.Fatalln(err)
	}
	defer con.Close()

	log.Println("Worker запущен:", w.Target)

	conn, err := ipv4.NewRawConn(con)
	if err != nil {
		log.Fatalf("Ошибка создания RawConn: %v", err)
	}
	defer conn.Close()

	for data := range c {
		dst := config.AddrConfig{
			Host: w.Target.Host,
			Port: w.Target.Port,
		}

		w.Sender.SendPacket(conn, data.Data, data.Src, dst)
	}
}
