package agent

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemMetrics interface {
	VirtualMemory(context.Context) (uint64, uint64, error)
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
