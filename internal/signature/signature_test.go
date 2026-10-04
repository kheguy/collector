package signature

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateAndValid(t *testing.T) {
	data := []byte("metrics")
	const key = "secret"

	encoded := Calculate(data, key)

	assert.Equal(t, "a6312fb98077b7e932fa66d9c1d5ed96898071c128eea1a132c5603674ba28de", encoded)
	assert.True(t, Valid(data, key, encoded))
	assert.False(t, Valid([]byte("changed"), key, encoded))
	assert.False(t, Valid(data, "wrong-key", encoded))
	assert.False(t, Valid(data, key, "not-hex"))
}
