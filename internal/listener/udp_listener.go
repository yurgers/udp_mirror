package listener

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	"udp_mirror/config"
	"udp_mirror/internal/worker"
	"udp_mirror/pkg/metrics"
)

type UDPListener struct {
	conn     *net.UDPConn
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

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	// Увеличиваем буфер приема до 8MB
	err = conn.SetReadBuffer(8 * 1024 * 1024)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(ctx)

	return &UDPListener{
		conn:     conn,
		channels: chs,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Устанавливаем таймаут для `ReadFromUDP`
func (l *UDPListener) nextReadDeadline() time.Time {
	return time.Now().Add(100 * time.Millisecond)
}

// Проверяем, является ли ошибка таймаутом
func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// Listen начинает прием данных с UDP-соединения
func (l *UDPListener) Start() {
	defer l.Close()
	plName, _ := l.ctx.Value(config.PlNameKey).(string)
	log.Printf("[Pipeline %s] Сервер запущен и слушает на %s\n", plName, l.conn.LocalAddr())

	buffer := make([]byte, 65536-28)
	// buffer := make([]byte, 1024*10)

	for {
		select {
		case <-l.ctx.Done():
			log.Printf("[Pipeline %s] UDP Listener завершает работу...\n", plName)
			return
		default:
			// log.Println("запуск новой итериции ")
			l.conn.SetReadDeadline(l.nextReadDeadline())
			n, src, err := l.conn.ReadFromUDP(buffer)
			if err != nil {
				if isTimeoutError(err) {
					continue // Просто повторяем чтение, если таймаут
				}
				slog.Error(fmt.Sprintf("[Pipeline %s] Ошибка чтения из UDP: %v", plName, err))
				break
			}
			// slog.Debug(fmt.Sprintf("[Pipeline %s] Полученные данные от %v, в размере %v", plName, src, n))
			metrics.IncrementReceived(plName, src.IP.String(), n)

			l.processData(buffer[:n], src)

			// go func(data []byte, src *net.UDPAddr) {
			// 	l.processData(data, src)
			// }(buffer[:n], src)

		}
	}
}

// processData обрабатывает полученные данные
func (l *UDPListener) processData(data []byte, src *net.UDPAddr) {
	for _, channel := range l.channels {
		channel <- worker.IRPData{
			Data: data,
			Src: config.AddrConfig{
				Host: src.IP,
				Port: uint16(src.Port),
			},
		}
	}
}

// // Обрабатываем полученные данные и уведомнением переполнености канала.
// func (l *UDPListener) processData(data []byte, src *net.UDPAddr) {
// 	plName, _ := l.ctx.Value(config.PlNameKey).(string)
// 	d := worker.IRPData{
// 		Data: data,
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
// 			// case <-time.After(100 * time.Millisecond): // Таймаут 100мс
// 			// 	slog.Debug(fmt.Sprintf("[%s] Канал %v переполнен, пакет отброшен", plName, i))

// 		default:
// 			slog.Debug(fmt.Sprintf("[%s] Канал %v переполнен, пакет отброшен", plName, i))
// 		}
// 	}
// }

func (l *UDPListener) Shutdown() {
	for _, channel := range l.channels {
		close(channel)
	}

	l.Close()
}

// Close закрывает соединение UDP
func (l *UDPListener) Close() error {
	return l.conn.Close()
}

// LocalAddr возвращает локальный адрес соединения
func (l *UDPListener) LocalAddr() net.Addr {
	return l.conn.LocalAddr()
}
