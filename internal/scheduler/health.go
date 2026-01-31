package scheduler

import (
	"sync"
	"time"
)

// HealthStatus represents the health of a component.
type HealthStatus struct {
	Healthy     bool
	LastCheck   time.Time
	LastSuccess time.Time
	LastError   error
	Message     string
}

// Health tracks the health of various components.
type Health struct {
	mu         sync.RWMutex
	components map[string]*HealthStatus
}

// NewHealth creates a new health tracker.
func NewHealth() *Health {
	return &Health{
		components: make(map[string]*HealthStatus),
	}
}

// SetHealthy marks a component as healthy.
func (h *Health) SetHealthy(component, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	if _, exists := h.components[component]; !exists {
		h.components[component] = &HealthStatus{}
	}

	h.components[component].Healthy = true
	h.components[component].LastCheck = now
	h.components[component].LastSuccess = now
	h.components[component].LastError = nil
	h.components[component].Message = message
}

// SetUnhealthy marks a component as unhealthy.
func (h *Health) SetUnhealthy(component string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	if _, exists := h.components[component]; !exists {
		h.components[component] = &HealthStatus{}
	}

	h.components[component].Healthy = false
	h.components[component].LastCheck = now
	h.components[component].LastError = err
	h.components[component].Message = err.Error()
}

// GetStatus returns the status of a component.
func (h *Health) GetStatus(component string) *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if status, exists := h.components[component]; exists {
		// Return a copy to avoid race conditions
		return &HealthStatus{
			Healthy:     status.Healthy,
			LastCheck:   status.LastCheck,
			LastSuccess: status.LastSuccess,
			LastError:   status.LastError,
			Message:     status.Message,
		}
	}

	return nil
}

// GetAllStatuses returns all component statuses.
func (h *Health) GetAllStatuses() map[string]*HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for name, status := range h.components {
		result[name] = &HealthStatus{
			Healthy:     status.Healthy,
			LastCheck:   status.LastCheck,
			LastSuccess: status.LastSuccess,
			LastError:   status.LastError,
			Message:     status.Message,
		}
	}

	return result
}

// IsOverallHealthy returns true if all components are healthy.
func (h *Health) IsOverallHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, status := range h.components {
		if !status.Healthy {
			return false
		}
	}

	return true
}
