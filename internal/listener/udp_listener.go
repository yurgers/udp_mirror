package listener

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"udp_mirror/config"
	"udp_mirror/internal/worker"
	"udp_mirror/pkg/metrics"

	"golang.org/x/sys/unix"
)

type UDPListener struct {
	addr *net.UDPAddr
	// conn     *net.UDPConn
	// channels []chan worker.IRPData
	channels []chan worker.IRPData

	ctx    context.Context
	cancel context.CancelFunc
}

// NewUDPListener создает новый экземпляр UDPListener
func NewUDPListener(ctx context.Context, serverAddr config.AddrConfig, chs []chan worker.IRPData) (*UDPListener, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", serverAddr.Host.String(), serverAddr.Port))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	return &UDPListener{
		addr: addr,

		channels: chs,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func listenReusePort(addr *net.UDPAddr) *net.UDPConn {
	// Настройка SO_REUSEPORT
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	lp, err := lc.ListenPacket(context.Background(), "udp", addr.String())

	conn := lp.(*net.UDPConn)

	// conn, err := net.ListenUDP("udp", addr)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Увеличиваем буфер приема до 8MB
	err = conn.SetReadBuffer(32 * 1024 * 1024)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

// Устанавливаем таймаут для `ReadFromUDP`
func (l *UDPListener) nextReadDeadline() time.Time {
	return time.Now().Add(300 * time.Millisecond)
	// return time.Now().Add(1 * time.Second)
}

// Проверяем, является ли ошибка таймаутом
func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// Listen начинает прием данных с UDP-соединения
func (l *UDPListener) Start(lName string) {
	conn := listenReusePort(l.addr)
	defer conn.Close()

	// defer l.Close()
	plName, _ := l.ctx.Value(config.PlNameKey).(string)
	log.Printf("[Pipeline %s] Сервер запущен и слушает на %s\n", plName, conn.LocalAddr())

	bufPool := sync.Pool{
		New: func() any {
			return make([]byte, 65536-28) // Выделяем буфер заранее
		},
	}
	// buffer := make([]byte, 1024*10)

	for {
		select {
		case <-l.ctx.Done():
			log.Printf("[Pipeline %s] UDP Listener завершает работу...\n", plName)
			return
		default:
			// log.Println("запуск новой итериции ")
			conn.SetReadDeadline(l.nextReadDeadline())

			// Получаем буфер из пула
			buffer := bufPool.Get().([]byte)

			n, src, err := conn.ReadFromUDP(buffer)
			if err != nil {
				bufPool.Put(buffer)
				if isTimeoutError(err) {
					continue // Просто повторяем чтение, если таймаут
				}
				slog.Error(fmt.Sprintf("[Pipeline %s] Ошибка чтения из UDP: %v", plName, err))
				continue
			}
			// slog.Debug(fmt.Sprintf("[Pipeline %s] Полученные данные от %v, в размере %v", plName, src, n))
			go metrics.IncrementReceived(lName, plName, src.IP.String(), n)

			safeData := make([]byte, n)
			copy(safeData, buffer[:n])
			// fmt.Printf("Адрес buffer: %p\n", unsafe.Pointer(&buffer[0]))
			// fmt.Printf("Адрес safeData: %p\n", unsafe.Pointer(&safeData[0]))

			l.processData(&safeData, src)

			bufPool.Put(buffer)
		}
	}
}

// // processData обрабатывает полученные данные
// func (l *UDPListener) processData(data *[]byte, src *net.UDPAddr) {
// 	// fmt.Printf("data: %v, Len: %d\n", data, len(data))
// 	// fmt.Printf("Адрес safeData: %p\n", unsafe.Pointer(&data[0]))
// 	for _, channel := range l.channels {
// 		channel <- worker.IRPData{
// 			Data: *data,
// 			Src: config.AddrConfig{
// 				Host: src.IP,
// 				Port: uint16(src.Port),
// 			},
// 		}
// 	}
// }

// Обрабатываем полученные данные и уведомнением переполнености канала.
func (l *UDPListener) processData(data *[]byte, src *net.UDPAddr) {
	d := worker.IRPData{
		Data: *data,
		Src: config.AddrConfig{
			Host: src.IP,
			Port: uint16(src.Port),
		},
	}

	for _, ch := range l.channels {
		select {
		case ch <- d:
		case <-time.After(100 * time.Millisecond):
		default:
		}
	}
}

// // Обрабатываем полученные данные и уведомнением переполнености канала.
// func (l *UDPListener) processData(data *[]byte, src *net.UDPAddr) {
// 	plName, _ := l.ctx.Value(config.PlNameKey).(string)
// 	d := worker.IRPData{
// 		Data: *data,
// 		Src: config.AddrConfig{
// 			Host: src.IP,
// 			Port: uint16(src.Port),
// 		},
// 	}

// 	for i, ch := range l.channels {
// 		select {
// 		case ch <- d:
// 			slog.Debug(fmt.Sprintf("[%s] пакет отправлен в Канал %v", plName, i))
// 			slog.Debug(fmt.Sprintf("[%s] Канал %v, наполнен на %d из %d", plName, i, len(ch), cap(ch)))
// 		// case <-time.After(100 * time.Millisecond): // Таймаут 100мс
// 		// 	slog.Debug(fmt.Sprintf("[%s] Канал %v переполнен, пакет отброшен", plName, i))

// 		default:
// 			slog.Info(fmt.Sprintf("[%s] Канал %v переполнен, пакет отброшен", plName, i))
// 		}
// 	}
// }

func (l *UDPListener) Shutdown() {
	for _, channel := range l.channels {
		close(channel)
	}

	// l.Close()
}

// Close закрывает соединение UDP
// func (l *UDPListener) Close() error {
// 	return l.conn.Close()
// }

// LocalAddr возвращает локальный адрес соединения
// func (l *UDPListener) LocalAddr() net.Addr {
// 	return l.conn.LocalAddr()
// }
