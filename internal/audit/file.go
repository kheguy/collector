package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

// FileObserver записывает события аудита в файл, по одному JSON объекту на строку
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileObserver открывает файл для добавления событий и создаёт его при необходимости
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	if err := ensureNewline(file); err != nil {
		file.Close()
		return nil, err
	}
	return &FileObserver{file: file}, nil
}

func ensureNewline(file *os.File) error {
	info, err := file.Stat()
	if err != nil || info.Size() == 0 {
		return err
	}
	var last [1]byte
	if _, err := file.ReadAt(last[:], info.Size()-1); err != nil {
		return err
	}
	if last[0] != '\n' {
		_, err = file.Write([]byte{'\n'})
	}
	return err
}

// Notify добавляет событие в файл
func (o *FileObserver) Notify(_ context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return json.NewEncoder(o.file).Encode(event)
}

// Close закрывает файл аудита
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.file.Close()
}
