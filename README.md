# udp_mirror

## 📌 Описание проекта
Столкнулись с проблемой отправки данных, например логов, в два разных приемника, таких как продакшен и тестовый контур, но у вас нет возможности настроить две цели на оборудовании? Этот сервис решает эту задачу.
`udp_mirror` - это высокопроизводительный сервис для зеркалирования UDP-трафика. Он принимает входящие UDP-пакеты и перенаправляет в указанные целевые адреса.


## 🚀 Возможности
- 📡 Мультиплексирование трафика на множество целей
- 🎭 Подмена адресов и порта источника трафика
- 🏎 Высокая производительность благодаря `goroutine`
- 📊 Метрики Prometheus
- 📉 Поддержка `pprof` для профилирования
- ~~🔄 Горячая перезагрузка конфига (`SIGHUP`)~~

---

## 🛠 Установка

```sh
git clone https://github.com/Yurgers/udp_mirror.git
cd udp_mirror
go build -o udp_mirror ./cmd/server
```

---

## ⚙ Конфигурация
Пример `config.yml`:

```yaml
pipeline:
  - name: "udp_mirror_1"
    input:
      host: "0.0.0.0"
      port: 9000
    targets:
      - host: 192.168.1.100
        port: 9001
      - host: 192.168.1.101
        port: 9002
        src_host: 172.0.0.11
        src_port: 9002
pprof:
  enabled: true
  listen: "localhost:6060"

prometheus:
  enabled: true
  listen: "localhost:9090"
```

---

## ▶ Запуск

```sh
./udp_mirror -f config.yml
```

Запуск в фоне:
```sh
./udp_mirror -f config.yml -d
```

Горячая перезагрузка конфига:
```sh
./udp_mirror -s reload
```

Остановка:
```sh
./udp_mirror -s quit
```

Принудительное завершение:
```sh
./udp_mirror -s stop
```


---

## 📊 Мониторинг

### Prometheus
Если включено в `config.yml`, метрики доступны по `http://localhost:9090/metrics`.

### pprof
Если включено в `config.yml`, профайлер доступен по `http://localhost:6060/debug/pprof/`.

---

## 🔄 Архитектура
- **Pipeline** (`pipeline.go`) - управляет процессом обработки UDP-пакета
- **Listener** (`udp_listener.go`) - принимает UDP-пакеты
- **WorkerManager** (`worker_manager.go`) - управляет группой воркеров
- **Worker** (`worker.go`) - обрабатывает входящие пакеты
- **Sender** (`udp_sender.go`) - отправляет UDP-пакеты


---

