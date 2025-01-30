package worker

import (
	"log"
	"udp_mirror/config"
	"udp_mirror/internal/sender"
)

type IRPData struct {
	Data []byte
	Src  config.AddrConfig
}

type Worker struct {
	Target config.TargetConfig
	Sender sender.PacketSender
}

func (w *Worker) ProcessPackets(ch <-chan IRPData) {
	// listen := "127.0.0.1"
	// if w.Target.Host.String() != listen {
	// 	listen = ""
	// }

	// conn, err := net.ListenPacket("ip4:udp", listen)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// defer conn.Close()

	//

	// rawConn, err := ipv4.NewRawConn(conn)
	// if err != nil {
	// 	log.Fatalf("Ошибка создания RawConn: %v", err)
	// }
	// defer rawConn.Close()

	log.Println("Worker запущен:", w.Target)

	for data := range ch {
		// log.Println("Полученные данные:", len(data.Data), "для", w.Target)
		if w.Target.SrcPort != 0 {
			data.Src.Port = w.Target.SrcPort
		}

		if w.Target.SrcHost != nil {
			data.Src.Host = w.Target.SrcHost
		}

		w.Sender.SendPacket(data.Data, data.Src)

		// time.Sleep(500 * time.Millisecond)
	}

	log.Println("Worker завершен:", w.Target)

}
