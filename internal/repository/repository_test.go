package repository

import (
	"testing"

	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMakeNewMemoryStorage(t *testing.T) {
	s := MakeNewMemoryStorage()
	assert.NotNil(t, s)
	assert.NotNil(t, s.data)
	assert.Empty(t, s.data)
}

func TestMemStorage_SetAndGet_Gauge(t *testing.T) {
	s := MakeNewMemoryStorage()

	result := s.Set("v1", models.Gauge, 1.11)
	assert.Equal(t, 1.11, result)

	val := s.Get("v1")
	assert.Equal(t, 1.11, val)
}

func TestMemStorage_SetAndGet_Counter(t *testing.T) {
	s := MakeNewMemoryStorage()

	s.Set("shhhhhhh", models.Counter, 2)
	s.Set("shhhhhhh", models.Counter, 1)

	val := s.Get("shhhhhhh")
	assert.Equal(t, 3, val)
}

func TestMemStorage_Set_GaugeOverwrite(t *testing.T) {
	s := MakeNewMemoryStorage()

	s.Set("rrrr", models.Gauge, 2.22)
	s.Set("rrrr", models.Gauge, 3.33)

	val := s.Get("rrrr")
	assert.Equal(t, 3.33, val)
}

func TestMemStorage_GetAll(t *testing.T) {
	s := MakeNewMemoryStorage()

	s.Set("z", models.Gauge, 1.1)
	s.Set("x", models.Counter, 2)
	s.Set("c", models.Gauge, 3.3)

	all := s.GetAll()
	assert.Len(t, all, 3)
	assert.Equal(t, 1.1, all["z"])
	assert.Equal(t, 2, all["x"])
	assert.Equal(t, 3.3, all["c"])
}

func TestMemStorage_GetAll_ReturnsCopy(t *testing.T) {
	s := MakeNewMemoryStorage()
	s.Set("aaaaaa", models.Gauge, 11.11)

	all := s.GetAll()
	all["aaaaaa"] = 88.8888

	original := s.Get("aaaaaa")
	assert.Equal(t, 11.11, original)
}

func TestMemStorage_Clear(t *testing.T) {
	s := MakeNewMemoryStorage()
	s.Set("a", models.Gauge, 1.1)
	s.Set("b", models.Counter, 2)

	s.Clear()

	all := s.GetAll()
	assert.Empty(t, all)
}

func TestMemStorage_Get_NotFound(t *testing.T) {
	s := MakeNewMemoryStorage()
	val := s.Get("what????")
	assert.Nil(t, val)
}
