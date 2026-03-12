package mapping

import (
	"encoding/json"
	"math"
	"sync"
)

// Mapping represents a register mapping configuration
type Mapping struct {
	ID               string            `json:"id"`
	ProxyID          string            `json:"proxy_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	RegisterAddress  int               `json:"register_address"`
	RegisterCount    int               `json:"register_count"`
	DataType         string            `json:"data_type"` // uint16, int16, uint32, int32, float32
	ScaleFactor      float64           `json:"scale_factor"`
	Offset           float64           `json:"offset"`
	Unit             string            `json:"unit"`
	Tags             []string          `json:"tags"`
	Enabled          bool              `json:"enabled"`
	Metadata         map[string]string `json:"metadata"`
}

// Manager manages register mappings
type Manager struct {
	mu       sync.RWMutex
	mappings map[string][]*Mapping // proxy_id -> mappings
}

// NewManager creates a new mapping manager
func NewManager() *Manager {
	return &Manager{
		mappings: make(map[string][]*Mapping),
	}
}

// AddMapping adds a register mapping
func (m *Manager) AddMapping(mapping *Mapping) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mappings := m.mappings[mapping.ProxyID]
	mappings = append(mappings, mapping)
	m.mappings[mapping.ProxyID] = mappings
}

// RemoveMapping removes a register mapping
func (m *Manager) RemoveMapping(mappingID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for proxyID, mappings := range m.mappings {
		newMappings := make([]*Mapping, 0, len(mappings))
		for _, mapping := range mappings {
			if mapping.ID != mappingID {
				newMappings = append(newMappings, mapping)
			}
		}
		m.mappings[proxyID] = newMappings
	}
}

// GetMappings returns all mappings for a proxy
func (m *Manager) GetMappings(proxyID string) []*Mapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mappings := m.mappings[proxyID]
	result := make([]*Mapping, len(mappings))
	copy(result, mappings)
	return result
}

// TransformValue transforms a raw register value according to the mapping
func (m *Manager) TransformValue(mapping *Mapping, rawValue []uint16) (interface{}, error) {
	if !mapping.Enabled {
		return nil, nil
	}

	var value float64

	switch mapping.DataType {
	case "uint16":
		if len(rawValue) > 0 {
			value = float64(rawValue[0])
		}
	case "int16":
		if len(rawValue) > 0 {
			value = float64(int16(rawValue[0]))
		}
	case "uint32":
		if len(rawValue) >= 2 {
			value = float64(uint32(rawValue[0])<<16 | uint32(rawValue[1]))
		}
	case "int32":
		if len(rawValue) >= 2 {
			value = float64(int32(rawValue[0])<<16 | int32(rawValue[1]))
		}
	case "float32":
		if len(rawValue) >= 2 {
			bits := uint32(rawValue[0])<<16 | uint32(rawValue[1])
			value = float64(math.Float32frombits(bits))
		}
	}

	// Apply scale and offset
	value = value*mapping.ScaleFactor + mapping.Offset

	return value, nil
}

// TransformMultiple transforms multiple register values
func (m *Manager) TransformMultiple(proxyID string, registerAddress int, values []uint16) map[string]interface{} {
	result := make(map[string]interface{})

	mappings := m.GetMappings(proxyID)
	for _, mapping := range mappings {
		if mapping.RegisterAddress == registerAddress && mapping.Enabled {
			transformed, err := m.TransformValue(mapping, values)
			if err == nil && transformed != nil {
				result[mapping.Name] = map[string]interface{}{
					"value":    transformed,
					"unit":     mapping.Unit,
					"tags":     mapping.Tags,
					"metadata": mapping.Metadata,
				}
			}
		}
	}

	return result
}

// GetMappingsJSON returns mappings as JSON
func (m *Manager) GetMappingsJSON(proxyID string) (string, error) {
	mappings := m.GetMappings(proxyID)
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
