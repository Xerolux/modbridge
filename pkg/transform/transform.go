package transform

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// TransformType defines the type of transformation
type TransformType int

const (
	// TransformNone - no transformation
	TransformNone TransformType = iota
	// TransformScale - multiply by scale factor
	TransformScale
	// TransformLinear - y = mx + b
	TransformLinear
	// TransformMap - map discrete values
	TransformMap
	// TransformSwap - swap byte order
	TransformSwap
	// TransformToInt - convert to integer
	TransformToInt
	// TransformToFloat - convert to float
	TransformToFloat
	// TransformCustom - custom transformation function
	TransformCustom
)

// TransformConfig holds configuration for a single transformation
type TransformConfig struct {
	// Type of transformation
	Type TransformType `json:"type" yaml:"type"`

	// For TransformScale: multiplier
	Scale float64 `json:"scale,omitempty" yaml:"scale,omitempty"`

	// For TransformLinear: y = mx + b
	Slope     float64 `json:"slope,omitempty" yaml:"slope,omitempty"`
	Intercept float64 `json:"intercept,omitempty" yaml:"intercept,omitempty"`

	// For TransformMap: discrete value mappings
	// Map[rawValue] = transformedValue
	Map map[uint64]float64 `json:"map,omitempty" yaml:"map,omitempty"`

	// For TransformSwap: swap byte order (big/little endian)
	SwapBytes bool `json:"swap_bytes,omitempty" yaml:"swap_bytes,omitempty"`

	// For TransformToInt/ToFloat: precision
	Precision int `json:"precision,omitempty" yaml:"precision,omitempty"`

	// For TransformCustom: function name (must be registered)
	CustomFunc string `json:"custom_func,omitempty" yaml:"custom_func,omitempty"`

	// Clamping
	MinValue *float64 `json:"min_value,omitempty" yaml:"min_value,omitempty"`
	MaxValue *float64 `json:"max_value,omitempty" yaml:"max_value,omitempty"`
}

// Registry holds custom transformation functions
type Registry struct {
	functions map[string]func(uint64) (float64, error)
}

// NewRegistry creates a new transformation registry
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]func(uint64) (float64, error)),
	}
}

// Register registers a custom transformation function
func (r *Registry) Register(name string, fn func(uint64) (float64, error)) error {
	if name == "" {
		return errors.New("function name cannot be empty")
	}
	if fn == nil {
		return errors.New("function cannot be nil")
	}
	r.functions[name] = fn
	return nil
}

// Get retrieves a custom transformation function
func (r *Registry) Get(name string) (func(uint64) (float64, error), bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

// Transformer applies transformations to data
type Transformer struct {
	registry *Registry
}

// NewTransformer creates a new transformer
func NewTransformer() *Transformer {
	return &Transformer{
		registry: NewRegistry(),
	}
}

// NewTransformerWithRegistry creates a transformer with a custom registry
func NewTransformerWithRegistry(registry *Registry) *Transformer {
	return &Transformer{
		registry: registry,
	}
}

// Register registers a custom transformation function
func (t *Transformer) Register(name string, fn func(uint64) (float64, error)) error {
	return t.registry.Register(name, fn)
}

// TransformRegister transforms a single register value
func (t *Transformer) TransformRegister(value uint16, config *TransformConfig) (float64, error) {
	if config == nil {
		return float64(value), nil
	}

	// Convert to uint64 for processing
	rawValue := uint64(value)

	var result float64
	var err error

	switch config.Type {
	case TransformNone:
		result = float64(rawValue)

	case TransformScale:
		result = float64(rawValue) * config.Scale

	case TransformLinear:
		result = config.Slope*float64(rawValue) + config.Intercept

	case TransformMap:
		if config.Map == nil {
			return 0, errors.New("map configuration is required for TransformMap")
		}
		mapped, ok := config.Map[rawValue]
		if !ok {
			return 0, fmt.Errorf("no mapping found for value %d", rawValue)
		}
		result = mapped

	case TransformSwap:
		// Swap byte order (big endian <-> little endian)
		bytes := make([]byte, 2)
		binary.BigEndian.PutUint16(bytes, value)
		swapped := binary.LittleEndian.Uint16(bytes)
		result = float64(swapped)

	case TransformToInt:
		if config.Precision > 0 {
			// Interpret as fixed-point decimal
			divisor := math.Pow(10, float64(config.Precision))
			result = float64(rawValue) / divisor
		} else {
			result = float64(rawValue)
		}

	case TransformToFloat:
		// Interpret uint16 as IEEE 754 half-precision float (16-bit)
		// For simplicity, just convert to float with precision
		result = float64(rawValue)
		if config.Precision > 0 {
			roundTo := math.Pow(10, float64(config.Precision))
			result = math.Round(result*roundTo) / roundTo
		}

	case TransformCustom:
		if config.CustomFunc == "" {
			return 0, errors.New("custom_func name is required for TransformCustom")
		}
		fn, ok := t.registry.Get(config.CustomFunc)
		if !ok {
			return 0, fmt.Errorf("custom function '%s' not found", config.CustomFunc)
		}
		result, err = fn(rawValue)
		if err != nil {
			return 0, fmt.Errorf("custom function failed: %w", err)
		}

	default:
		return 0, fmt.Errorf("unknown transformation type: %d", config.Type)
	}

	// Apply clamping
	if config.MinValue != nil && result < *config.MinValue {
		result = *config.MinValue
	}
	if config.MaxValue != nil && result > *config.MaxValue {
		result = *config.MaxValue
	}

	return result, nil
}

// TransformRegisterArray transforms multiple register values
func (t *Transformer) TransformRegisterArray(values []uint16, config *TransformConfig) ([]float64, error) {
	if config == nil {
		result := make([]float64, len(values))
		for i, v := range values {
			result[i] = float64(v)
		}
		return result, nil
	}

	result := make([]float64, len(values))
	for i, value := range values {
		transformed, err := t.TransformRegister(value, config)
		if err != nil {
			return nil, fmt.Errorf("failed to transform register %d: %w", i, err)
		}
		result[i] = transformed
	}

	return result, nil
}

// TransformRegisterBatch transforms multiple registers with individual configs
func (t *Transformer) TransformRegisterBatch(values []uint16, configs []*TransformConfig) ([]float64, error) {
	if len(values) != len(configs) {
		return nil, fmt.Errorf("values and configs length mismatch: %d vs %d", len(values), len(configs))
	}

	result := make([]float64, len(values))
	for i, value := range values {
		transformed, err := t.TransformRegister(value, configs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to transform register %d: %w", i, err)
		}
		result[i] = transformed
	}

	return result, nil
}

// InverseTransform performs inverse transformation (from engineering units to raw value)
func (t *Transformer) InverseTransform(value float64, config *TransformConfig) (uint16, error) {
	if config == nil {
		return uint16(value), nil
	}

	var result float64

	switch config.Type {
	case TransformNone:
		result = value

	case TransformScale:
		if config.Scale == 0 {
			return 0, errors.New("scale factor cannot be zero for inverse transform")
		}
		result = value / config.Scale

	case TransformLinear:
		if config.Slope == 0 {
			return 0, errors.New("slope cannot be zero for inverse transform")
		}
		result = (value - config.Intercept) / config.Slope

	case TransformToInt:
		if config.Precision > 0 {
			multiplier := math.Pow(10, float64(config.Precision))
			result = value * multiplier
		} else {
			result = value
		}

	case TransformSwap, TransformToFloat, TransformMap, TransformCustom:
		return 0, fmt.Errorf("inverse transform not supported for type %d", config.Type)

	default:
		return 0, fmt.Errorf("unknown transformation type: %d", config.Type)
	}

	// Clamp to valid range
	if result < 0 {
		result = 0
	}
	if result > math.MaxUint16 {
		result = math.MaxUint16
	}

	return uint16(math.Round(result)), nil
}

// TransformCoil transforms a coil value (boolean)
func (t *Transformer) TransformCoil(value bool, config *TransformConfig) (bool, error) {
	if config == nil {
		return value, nil
	}

	switch config.Type {
	case TransformNone:
		return value, nil

	case TransformMap:
		rawValue := uint64(0)
		if value {
			rawValue = 1
		}
		mapped, ok := config.Map[rawValue]
		if !ok {
			return false, fmt.Errorf("no mapping found for value %d", rawValue)
		}
		return mapped != 0, nil

	default:
		return value, fmt.Errorf("transformation type %d not supported for coils", config.Type)
	}
}

// TransformCoilArray transforms multiple coil values
func (t *Transformer) TransformCoilArray(values []bool, config *TransformConfig) ([]bool, error) {
	if config == nil {
		return values, nil
	}

	result := make([]bool, len(values))
	for i, value := range values {
		transformed, err := t.TransformCoil(value, config)
		if err != nil {
			return nil, fmt.Errorf("failed to transform coil %d: %w", i, err)
		}
		result[i] = transformed
	}

	return result, nil
}

// RegisterTransformConfig defines transformation for a range of registers
type RegisterTransformConfig struct {
	// Start address (inclusive)
	StartAddr uint16 `json:"start_addr" yaml:"start_addr"`
	// End address (inclusive), 0 means single register
	EndAddr uint16 `json:"end_addr,omitempty" yaml:"end_addr,omitempty"`
	// Transformation to apply
	Transform *TransformConfig `json:"transform" yaml:"transform"`
}

// DeviceTransformConfig defines transformations for a device
type DeviceTransformConfig struct {
	// Device ID
	DeviceID string `json:"device_id" yaml:"device_id"`
	// Register transformations
	Registers []RegisterTransformConfig `json:"registers" yaml:"registers"`
	// Coil transformations
	Coils []RegisterTransformConfig `json:"coils" yaml:"coils"`
}

// TransformForRegister finds the transformation config for a given register address
func (d *DeviceTransformConfig) TransformForRegister(addr uint16) *TransformConfig {
	for _, reg := range d.Registers {
		if addr >= reg.StartAddr && (reg.EndAddr == 0 || addr <= reg.EndAddr) {
			return reg.Transform
		}
	}
	return nil
}

// TransformForCoil finds the transformation config for a given coil address
func (d *DeviceTransformConfig) TransformForCoil(addr uint16) *TransformConfig {
	for _, coil := range d.Coils {
		if addr >= coil.StartAddr && (coil.EndAddr == 0 || addr <= coil.EndAddr) {
			return coil.Transform
		}
	}
	return nil
}

// Validate validates a transformation configuration
func (t *TransformConfig) Validate() error {
	switch t.Type {
	case TransformScale:
		if t.Scale == 0 {
			return errors.New("scale factor cannot be zero")
		}

	case TransformLinear:
		if t.Slope == 0 {
			return errors.New("slope cannot be zero")
		}

	case TransformMap:
		if t.Map == nil || len(t.Map) == 0 {
			return errors.New("map cannot be empty for TransformMap")
		}

	case TransformCustom:
		if t.CustomFunc == "" {
			return errors.New("custom_func name is required for TransformCustom")
		}

	case TransformToInt, TransformToFloat:
		if t.Precision < 0 || t.Precision > 10 {
			return errors.New("precision must be between 0 and 10")
		}
	}

	// Validate min/max
	if t.MinValue != nil && t.MaxValue != nil {
		if *t.MinValue > *t.MaxValue {
			return errors.New("min_value cannot be greater than max_value")
		}
	}

	return nil
}

// Common transformation presets
var (
	// Temperature: Convert raw value to Celsius (assuming 0.1°C resolution)
	PresetTemperatureC = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.1,
		Precision: 1,
	}

	// Temperature: Convert raw value to Fahrenheit
	PresetTemperatureF = &TransformConfig{
		Type:      TransformLinear,
		Slope:     0.18,
		Intercept: 32,
		Precision: 1,
	}

	// Pressure: Convert raw value to Bar (assuming 0.01 bar resolution)
	PresetPressureBar = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.01,
		Precision: 2,
	}

	// Humidity: Convert raw value to Percentage (assuming 0.1% resolution)
	PresetHumidity = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.1,
		Precision: 1,
		MinValue:  ptrFloat64(0),
		MaxValue:  ptrFloat64(100),
	}

	// Energy: Convert raw value to kWh (assuming 1 Wh resolution)
	PresetEnergyKWh = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.001,
		Precision: 3,
	}

	// Voltage: Convert raw value to Volts (assuming 0.1V resolution)
	PresetVoltage = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.1,
		Precision: 1,
	}

	// Current: Convert raw value to Amperes (assuming 0.001A resolution)
	PresetCurrent = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.001,
		Precision: 3,
	}

	// Power: Convert raw value to kW (assuming 1W resolution)
	PresetPowerKW = &TransformConfig{
		Type:      TransformScale,
		Scale:     0.001,
		Precision: 3,
	}
)

// Helper function to get pointer to float64
func ptrFloat64(v float64) *float64 {
	return &v
}
