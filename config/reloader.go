package config

import (
	"log"
	"sync"
)

// ConfigReloader управляет конфигурацией и её обновлением
type ConfigReloader struct {
	mu     sync.Mutex
	config *Config
}

// LoadConfig загружает конфиг из файла
func (cr *ConfigReloader) LoadConfig(fileName string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cfg := GetConfig(fileName) // Читаем новый конфиг
	cr.config = &cfg

	log.Println("[Config] Конфигурация обновлена!")
	return nil
}

// GetConfigCopy возвращает копию текущего конфига (чтобы избежать гонки данных)
func (cr *ConfigReloader) GetConfigCopy() Config {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return *cr.config
}

// а у меня конфиг, может менять мледующие вещи:
// добавлять/удалять целый  pipeline
// менять имя pipeline

// менять input

// изменять targets, как его настройку так и их количество.
