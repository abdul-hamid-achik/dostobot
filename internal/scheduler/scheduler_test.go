package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealth_SetHealthy(t *testing.T) {
	h := NewHealth()

	h.SetHealthy("test", "all good")

	status := h.GetStatus("test")
	assert.True(t, status.Healthy)
	assert.Equal(t, "all good", status.Message)
	assert.Nil(t, status.LastError)
	assert.WithinDuration(t, time.Now(), status.LastCheck, time.Second)
	assert.WithinDuration(t, time.Now(), status.LastSuccess, time.Second)
}

func TestHealth_SetUnhealthy(t *testing.T) {
	h := NewHealth()

	err := assert.AnError
	h.SetUnhealthy("test", err)

	status := h.GetStatus("test")
	assert.False(t, status.Healthy)
	assert.Equal(t, err, status.LastError)
	assert.Equal(t, err.Error(), status.Message)
	assert.WithinDuration(t, time.Now(), status.LastCheck, time.Second)
}

func TestHealth_GetStatus_NotFound(t *testing.T) {
	h := NewHealth()

	status := h.GetStatus("nonexistent")
	assert.Nil(t, status)
}

func TestHealth_GetAllStatuses(t *testing.T) {
	h := NewHealth()

	h.SetHealthy("comp1", "ok")
	h.SetHealthy("comp2", "ok")
	h.SetUnhealthy("comp3", assert.AnError)

	statuses := h.GetAllStatuses()
	assert.Len(t, statuses, 3)
	assert.True(t, statuses["comp1"].Healthy)
	assert.True(t, statuses["comp2"].Healthy)
	assert.False(t, statuses["comp3"].Healthy)
}

func TestHealth_IsOverallHealthy(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		h := NewHealth()
		h.SetHealthy("comp1", "ok")
		h.SetHealthy("comp2", "ok")

		assert.True(t, h.IsOverallHealthy())
	})

	t.Run("one unhealthy", func(t *testing.T) {
		h := NewHealth()
		h.SetHealthy("comp1", "ok")
		h.SetUnhealthy("comp2", assert.AnError)

		assert.False(t, h.IsOverallHealthy())
	})

	t.Run("empty", func(t *testing.T) {
		h := NewHealth()
		assert.True(t, h.IsOverallHealthy())
	})
}
