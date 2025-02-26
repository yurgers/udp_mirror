package worker

import (
	"context"
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

func (w *Worker) StartProcessPackets(ctx context.Context, ch <-chan IRPData) {
	plName, _ := ctx.Value(config.PlNameKey).(string)

	log.Printf("[Pipeline %s] Worker запущен: %+v", plName, w.Target)

	for data := range ch {
		// log.Printf("Полученные данные: len: %d для %v\n", len(data.Data), w.Target)
		// log.Printf("Адрес inSafeData: %p\n", unsafe.Pointer(&data.Data[0]))
		if w.Target.SrcPort != 0 {
			data.Src.Port = w.Target.SrcPort
		}

		if w.Target.SrcHost != nil {
			data.Src.Host = w.Target.SrcHost
		}

		w.Sender.SendPacket(data.Data, data.Src)

		// time.Sleep(500 * time.Millisecond)
	}

	log.Printf("[Pipeline %s] Worker завершен: %+v\n", plName, w.Target)

}
