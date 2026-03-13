package transform

import (
	"math"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if reg.functions == nil {
		t.Error("registry functions map is nil")
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()

	// Test valid registration
	err := reg.Register("test", func(v uint64) (float64, error) {
		return float64(v) * 2, nil
	})
	if err != nil {
		t.Errorf("Failed to register function: %v", err)
	}

	// Test empty name
	err = reg.Register("", func(v uint64) (float64, error) {
		return float64(v), nil
	})
	if err == nil {
		t.Error("Expected error for empty function name")
	}

	// Test nil function
	err = reg.Register("test2", nil)
	if err == nil {
		t.Error("Expected error for nil function")
	}
}

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()
	testFn := func(v uint64) (float64, error) {
		return float64(v) * 2, nil
	}
	reg.Register("double", testFn)

	// Test existing function
	fn, ok := reg.Get("double")
	if !ok {
		t.Error("Failed to get registered function")
	}
	if fn == nil {
		t.Error("Retrieved function is nil")
	}

	// Test non-existing function
	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Should not find non-existent function")
	}
}

func TestNewTransformer(t *testing.T) {
	trans := NewTransformer()
	if trans == nil {
		t.Fatal("NewTransformer() returned nil")
	}
	if trans.registry == nil {
		t.Error("transformer registry is nil")
	}
}

func TestTransformerRegister(t *testing.T) {
	trans := NewTransformer()
	err := trans.Register("square", func(v uint64) (float64, error) {
		return float64(v * v), nil
	})
	if err != nil {
		t.Errorf("Failed to register custom function: %v", err)
	}
}

func TestTransformRegister_None(t *testing.T) {
	trans := NewTransformer()

	result, err := trans.TransformRegister(100, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("Expected 100, got %f", result)
	}

	config := &TransformConfig{Type: TransformNone}
	result, err = trans.TransformRegister(100, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("Expected 100, got %f", result)
	}
}

func TestTransformRegister_Scale(t *testing.T) {
	trans := NewTransformer()

	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}

	// Test scaling
	result, err := trans.TransformRegister(1234, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 123.4 {
		t.Errorf("Expected 123.4, got %f", result)
	}

	// Test zero scale (should fail validation)
	config.Scale = 0
	err = config.Validate()
	if err == nil {
		t.Error("Expected error for zero scale")
	}
}

func TestTransformRegister_Linear(t *testing.T) {
	trans := NewTransformer()

	config := &TransformConfig{
		Type:      TransformLinear,
		Slope:     0.5,
		Intercept: 32,
	}

	// y = 0.5 * x + 32
	result, err := trans.TransformRegister(100, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := 0.5*100 + 32
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestTransformRegister_Map(t *testing.T) {
	trans := NewTransformer()

	config := &TransformConfig{
		Type: TransformMap,
		Map: map[uint64]float64{
			0: 10.5,
			1: 20.5,
			2: 30.5,
		},
	}

	// Test existing mapping
	result, err := trans.TransformRegister(1, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 20.5 {
		t.Errorf("Expected 20.5, got %f", result)
	}

	// Test non-existing mapping
	_, err = trans.TransformRegister(5, config)
	if err == nil {
		t.Error("Expected error for non-existent mapping")
	}
}

func TestTransformRegister_Swap(t *testing.T) {
	trans := NewTransformer()

	config := &TransformConfig{
		Type:      TransformSwap,
		SwapBytes: true,
	}

	// 0x1234 as big endian, swapped to little endian = 0x3412
	result, err := trans.TransformRegister(0x1234, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 0x3412 {
		t.Errorf("Expected %d, got %f", 0x3412, result)
	}
}

func TestTransformRegister_ToInt(t *testing.T) {
	trans := NewTransformer()

	config := &TransformConfig{
		Type:      TransformToInt,
		Precision: 2,
	}

	// Raw value 1234 with 2 decimal places = 12.34
	result, err := trans.TransformRegister(1234, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 12.34 {
		t.Errorf("Expected 12.34, got %f", result)
	}
}

func TestTransformRegister_Custom(t *testing.T) {
	trans := NewTransformer()

	// Register custom function
	trans.Register("celsius_to_fahrenheit", func(v uint64) (float64, error) {
		celsius := float64(v) * 0.1
		return celsius*1.8 + 32, nil
	})

	config := &TransformConfig{
		Type:       TransformCustom,
		CustomFunc: "celsius_to_fahrenheit",
	}

	// 250 raw = 25.0°C = 77.0°F
	result, err := trans.TransformRegister(250, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := 25.0*1.8 + 32
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}

	// Test non-existent function
	config.CustomFunc = "nonexistent"
	_, err = trans.TransformRegister(100, config)
	if err == nil {
		t.Error("Expected error for non-existent custom function")
	}
}

func TestTransformRegister_Clamping(t *testing.T) {
	trans := NewTransformer()

	minVal := 10.0
	maxVal := 100.0
	config := &TransformConfig{
		Type:     TransformScale,
		Scale:    1.0,
		MinValue: &minVal,
		MaxValue: &maxVal,
	}

	// Test below minimum (value 5 < min 10)
	result, err := trans.TransformRegister(5, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 10 {
		t.Errorf("Expected 10 (clamped to min), got %f", result)
	}

	// Test above maximum (value 150 > max 100)
	result, err = trans.TransformRegister(150, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("Expected 100 (clamped to max), got %f", result)
	}

	// Test within range
	result, err = trans.TransformRegister(50, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 50 {
		t.Errorf("Expected 50, got %f", result)
	}
}

func TestTransformRegisterArray(t *testing.T) {
	trans := NewTransformer()

	values := []uint16{100, 200, 300}
	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}

	result, err := trans.TransformRegisterArray(values, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result[0] != 10 || result[1] != 20 || result[2] != 30 {
		t.Errorf("Unexpected results: %v", result)
	}
}

func TestTransformRegisterBatch(t *testing.T) {
	trans := NewTransformer()

	values := []uint16{100, 200, 300}
	configs := []*TransformConfig{
		{Type: TransformScale, Scale: 0.1},
		{Type: TransformScale, Scale: 0.01},
		{Type: TransformNone},
	}

	result, err := trans.TransformRegisterBatch(values, configs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result[0] != 10 || result[1] != 2 || result[2] != 300 {
		t.Errorf("Unexpected results: %v", result)
	}

	// Test length mismatch
	_, err = trans.TransformRegisterBatch(values, configs[:2])
	if err == nil {
		t.Error("Expected error for length mismatch")
	}
}

func TestInverseTransform(t *testing.T) {
	trans := NewTransformer()

	// Test inverse scale
	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}

	result, err := trans.InverseTransform(12.34, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 123 { // 12.34 / 0.1 = 123.4, rounded = 123
		t.Errorf("Expected 123, got %d", result)
	}

	// Test inverse linear
	config2 := &TransformConfig{
		Type:      TransformLinear,
		Slope:     2,
		Intercept: 10,
	}

	// y = 2x + 10, so x = (y - 10) / 2
	// For y = 50, x = 20
	result, err = trans.InverseTransform(50, config2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 20 {
		t.Errorf("Expected 20, got %d", result)
	}

	// Test unsupported type
	config3 := &TransformConfig{Type: TransformMap}
	_, err = trans.InverseTransform(100, config3)
	if err == nil {
		t.Error("Expected error for unsupported inverse transform")
	}
}

func TestTransformCoil(t *testing.T) {
	trans := NewTransformer()

	// Test no transform
	result, err := trans.TransformCoil(true, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result {
		t.Error("Expected true, got false")
	}

	// Test mapping
	config := &TransformConfig{
		Type: TransformMap,
		Map: map[uint64]float64{
			0: 1, // false maps to true
			1: 0, // true maps to false
		},
	}

	result, err = trans.TransformCoil(true, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result {
		t.Error("Expected false (mapped), got true")
	}

	// Test unsupported type
	config2 := &TransformConfig{Type: TransformScale}
	_, err = trans.TransformCoil(true, config2)
	if err == nil {
		t.Error("Expected error for unsupported coil transform")
	}
}

func TestTransformCoilArray(t *testing.T) {
	trans := NewTransformer()

	values := []bool{true, false, true}
	result, err := trans.TransformCoilArray(values, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
}

func TestDeviceTransformConfig(t *testing.T) {
	config := &DeviceTransformConfig{
		DeviceID: "test-device",
		Registers: []RegisterTransformConfig{
			{
				StartAddr: 0,
				EndAddr:   9,
				Transform: &TransformConfig{
					Type:  TransformScale,
					Scale: 0.1,
				},
			},
			{
				StartAddr: 100,
				Transform: &TransformConfig{
					Type:  TransformScale,
					Scale: 0.01,
				},
			},
		},
	}

	// Test finding transform for register in range
	transform := config.TransformForRegister(5)
	if transform == nil {
		t.Error("Expected transform for register 5")
	}
	if transform.Scale != 0.1 {
		t.Errorf("Expected scale 0.1, got %f", transform.Scale)
	}

	// Test finding transform for single register
	transform = config.TransformForRegister(100)
	if transform == nil {
		t.Error("Expected transform for register 100")
	}
	if transform.Scale != 0.01 {
		t.Errorf("Expected scale 0.01, got %f", transform.Scale)
	}

	// Test no transform found
	transform = config.TransformForRegister(50)
	if transform != nil {
		t.Error("Expected no transform for register 50")
	}
}

func TestValidateTransformConfig(t *testing.T) {
	// Test valid scale config
	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid scale (zero)
	config.Scale = 0
	err = config.Validate()
	if err == nil {
		t.Error("Expected error for zero scale")
	}

	// Test valid linear config
	config2 := &TransformConfig{
		Type:  TransformLinear,
		Slope: 2.5,
	}
	err = config2.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid linear (zero slope)
	config2.Slope = 0
	err = config2.Validate()
	if err == nil {
		t.Error("Expected error for zero slope")
	}

	// Test valid map config
	config3 := &TransformConfig{
		Type: TransformMap,
		Map: map[uint64]float64{
			0: 1.0,
		},
	}
	err = config3.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid map (empty)
	config3.Map = map[uint64]float64{}
	err = config3.Validate()
	if err == nil {
		t.Error("Expected error for empty map")
	}

	// Test valid custom config
	config4 := &TransformConfig{
		Type:       TransformCustom,
		CustomFunc: "test_func",
	}
	err = config4.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid custom (no function name)
	config4.CustomFunc = ""
	err = config4.Validate()
	if err == nil {
		t.Error("Expected error for empty custom_func")
	}

	// Test invalid precision
	config5 := &TransformConfig{
		Type:      TransformToInt,
		Precision: -1,
	}
	err = config5.Validate()
	if err == nil {
		t.Error("Expected error for negative precision")
	}

	// Test invalid min/max
	config6 := &TransformConfig{
		Type:     TransformScale,
		Scale:    1.0,
		MinValue: ptrFloat64(100),
		MaxValue: ptrFloat64(50),
	}
	err = config6.Validate()
	if err == nil {
		t.Error("Expected error for min > max")
	}
}

func TestPresets(t *testing.T) {
	trans := NewTransformer()

	// Test temperature preset
	result, err := trans.TransformRegister(250, PresetTemperatureC)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 25.0 {
		t.Errorf("Expected 25.0°C, got %f", result)
	}

	// Test pressure preset
	result, err = trans.TransformRegister(5000, PresetPressureBar)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 50.0 {
		t.Errorf("Expected 50.0 bar, got %f", result)
	}

	// Test humidity preset
	result, err = trans.TransformRegister(650, PresetHumidity)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 65.0 {
		t.Errorf("Expected 65.0%%, got %f", result)
	}

	// Test energy preset
	result, err = trans.TransformRegister(5000, PresetEnergyKWh)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5.0 {
		t.Errorf("Expected 5.0 kWh, got %f", result)
	}

	// Test voltage preset
	result, err = trans.TransformRegister(2300, PresetVoltage)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 230.0 {
		t.Errorf("Expected 230.0 V, got %f", result)
	}

	// Test current preset
	result, err = trans.TransformRegister(5000, PresetCurrent)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5.0 {
		t.Errorf("Expected 5.0 A, got %f", result)
	}

	// Test power preset
	result, err = trans.TransformRegister(5000, PresetPowerKW)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5.0 {
		t.Errorf("Expected 5.0 kW, got %f", result)
	}
}

func TestClampingWithPresets(t *testing.T) {
	trans := NewTransformer()

	// Humidity should be clamped to 0-100%
	// Test value above 100%
	result, err := trans.TransformRegister(1200, PresetHumidity)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("Expected 100%% (clamped), got %f", result)
	}
}

func TestInverseTransformWithPresets(t *testing.T) {
	trans := NewTransformer()

	// Test inverse temperature: 25.0°C -> 250 raw
	result, err := trans.InverseTransform(25.0, PresetTemperatureC)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 250 {
		t.Errorf("Expected 250, got %d", result)
	}

	// Test inverse voltage: 230.0V -> 2300 raw
	result, err = trans.InverseTransform(230.0, PresetVoltage)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 2300 {
		t.Errorf("Expected 2300, got %d", result)
	}
}

func TestTransformRegister_RoundTrip(t *testing.T) {
	trans := NewTransformer()

	originalValues := []uint16{100, 200, 300, 400, 500}
	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}

	// Forward transform
	transformed, err := trans.TransformRegisterArray(originalValues, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Inverse transform
	reconstructed := make([]uint16, len(transformed))
	for i, v := range transformed {
		reconstructed[i], err = trans.InverseTransform(v, config)
		if err != nil {
			t.Errorf("Unexpected error in inverse transform: %v", err)
		}
	}

	// Should match original
	for i, orig := range originalValues {
		if reconstructed[i] != orig {
			t.Errorf("Round trip failed at index %d: original=%d, reconstructed=%d",
				i, orig, reconstructed[i])
		}
	}
}

func TestTransformRegister_EdgeCases(t *testing.T) {
	trans := NewTransformer()

	// Test zero value
	config := &TransformConfig{
		Type:  TransformScale,
		Scale: 0.1,
	}
	result, err := trans.TransformRegister(0, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0, got %f", result)
	}

	// Test max uint16 value
	result, err = trans.TransformRegister(math.MaxUint16, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := float64(math.MaxUint16) * 0.1
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}

	// Test very small scale
	config.Scale = 0.001
	result, err = trans.TransformRegister(1000, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 1.0 {
		t.Errorf("Expected 1.0, got %f", result)
	}
}

func TestTransformRegister_ArrayNilConfig(t *testing.T) {
	trans := NewTransformer()

	values := []uint16{100, 200, 300}
	result, err := trans.TransformRegisterArray(values, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	for i, v := range values {
		if result[i] != float64(v) {
			t.Errorf("Result[%d] = %f, want %f", i, result[i], float64(v))
		}
	}
}

func TestTransformCoil_NilConfig(t *testing.T) {
	trans := NewTransformer()

	// Test true
	result, err := trans.TransformCoil(true, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result {
		t.Error("Expected true, got false")
	}

	// Test false
	result, err = trans.TransformCoil(false, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result {
		t.Error("Expected false, got true")
	}
}

func TestInverseTransform_Clamping(t *testing.T) {
	trans := NewTransformer()

	minVal := 10.0
	maxVal := 100.0
	config := &TransformConfig{
		Type:     TransformScale,
		Scale:    1.0,
		MinValue: &minVal,
		MaxValue: &maxVal,
	}

	// Test below range (inverse transform doesn't apply clamping)
	// This is expected behavior - clamping only applies in forward direction
	result, err := trans.InverseTransform(5.0, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}

	// Test above range (should clamp to MaxUint16)
	_, err = trans.InverseTransform(100000.0, config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// Result should be MaxUint16 (65535)
}

func TestDeviceTransformConfig_Coils(t *testing.T) {
	config := &DeviceTransformConfig{
		DeviceID: "test-device",
		Coils: []RegisterTransformConfig{
			{
				StartAddr: 0,
				EndAddr:   7,
				Transform: &TransformConfig{
					Type: TransformMap,
					Map: map[uint64]float64{
						0: 1,
						1: 0,
					},
				},
			},
		},
	}

	// Test finding transform for coil in range
	transform := config.TransformForCoil(3)
	if transform == nil {
		t.Error("Expected transform for coil 3")
	}

	// Test no transform found
	transform = config.TransformForCoil(10)
	if transform != nil {
		t.Error("Expected no transform for coil 10")
	}
}
