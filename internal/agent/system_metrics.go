package agent

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// SystemMetrics предоставляет сведения о памяти и загрузке процессоров
type SystemMetrics interface {
	// VirtualMemory возвращает общий и свободный объём памяти в байтах
	VirtualMemory(context.Context) (uint64, uint64, error)
	// CPUPercent возвращает загрузку каждого логического процессора в процентах
	CPUPercent(context.Context) ([]float64, error)
}

type gopsutilMetrics struct{}

func (gopsutilMetrics) VirtualMemory(ctx context.Context) (uint64, uint64, error) {
	memory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	return memory.Total, memory.Free, nil
}

func (gopsutilMetrics) CPUPercent(ctx context.Context) ([]float64, error) {
	return cpu.PercentWithContext(ctx, 0, true)
}
