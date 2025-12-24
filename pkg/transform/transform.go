package transform

import (
	"encoding/binary"
	"errors"
	"math"
)

// DataType represents the data type for conversion.
type DataType string

const (
	DataTypeInt16   DataType = "int16"
	DataTypeUint16  DataType = "uint16"
	DataTypeInt32   DataType = "int32"
	DataTypeUint32  DataType = "uint32"
	DataTypeFloat32 DataType = "float32"
	DataTypeFloat64 DataType = "float64"
)

var (
	// ErrInvalidData is returned when data is invalid for conversion.
	ErrInvalidData = errors.New("invalid data for conversion")
)

// RegisterMapping defines a mapping between logical addresses and physical addresses.
type RegisterMapping struct {
	LogicalAddr  uint16
	PhysicalAddr uint16
	Count        uint16
}

// DataMapping defines data transformation rules.
type DataMapping struct {
	Address    uint16
	Type       DataType
	Scale      float64 // Scaling factor
	Offset     float64 // Offset to add
	ByteOrder  binary.ByteOrder
	SwapWords  bool // For 32-bit values
}

// Transformer handles data transformations.
type Transformer struct {
	addressMappings map[uint16]*RegisterMapping
	dataMappings    map[uint16]*DataMapping
}

// NewTransformer creates a new data transformer.
func NewTransformer() *Transformer {
	return &Transformer{
		addressMappings: make(map[uint16]*RegisterMapping),
		dataMappings:    make(map[uint16]*DataMapping),
	}
}

// AddAddressMapping adds a register address mapping.
func (t *Transformer) AddAddressMapping(logical, physical, count uint16) {
	t.addressMappings[logical] = &RegisterMapping{
		LogicalAddr:  logical,
		PhysicalAddr: physical,
		Count:        count,
	}
}

// AddDataMapping adds a data transformation mapping.
func (t *Transformer) AddDataMapping(mapping *DataMapping) {
	if mapping.ByteOrder == nil {
		mapping.ByteOrder = binary.BigEndian
	}
	if mapping.Scale == 0 {
		mapping.Scale = 1.0
	}

	t.dataMappings[mapping.Address] = mapping
}

// TransformAddress translates a logical address to a physical address.
func (t *Transformer) TransformAddress(logicalAddr uint16) (uint16, error) {
	if mapping, exists := t.addressMappings[logicalAddr]; exists {
		return mapping.PhysicalAddr, nil
	}

	// No mapping, return original
	return logicalAddr, nil
}

// TransformData applies data transformations to register values.
func (t *Transformer) TransformData(address uint16, registers []uint16) (interface{}, error) {
	mapping, exists := t.dataMappings[address]
	if !exists {
		// No mapping, return raw uint16 values
		return registers, nil
	}

	// Convert based on data type
	var value interface{}
	var err error

	switch mapping.Type {
	case DataTypeInt16:
		if len(registers) < 1 {
			return nil, ErrInvalidData
		}
		value = int16(registers[0])

	case DataTypeUint16:
		if len(registers) < 1 {
			return nil, ErrInvalidData
		}
		value = registers[0]

	case DataTypeInt32:
		if len(registers) < 2 {
			return nil, ErrInvalidData
		}
		value = t.registersToInt32(registers[:2], mapping.ByteOrder, mapping.SwapWords)

	case DataTypeUint32:
		if len(registers) < 2 {
			return nil, ErrInvalidData
		}
		value = t.registersToUint32(registers[:2], mapping.ByteOrder, mapping.SwapWords)

	case DataTypeFloat32:
		if len(registers) < 2 {
			return nil, ErrInvalidData
		}
		value = t.registersToFloat32(registers[:2], mapping.ByteOrder, mapping.SwapWords)

	case DataTypeFloat64:
		if len(registers) < 4 {
			return nil, ErrInvalidData
		}
		value = t.registersToFloat64(registers[:4], mapping.ByteOrder, mapping.SwapWords)

	default:
		return nil, errors.New("unsupported data type")
	}

	// Apply scaling and offset
	switch v := value.(type) {
	case int16:
		return float64(v)*mapping.Scale + mapping.Offset, nil
	case uint16:
		return float64(v)*mapping.Scale + mapping.Offset, nil
	case int32:
		return float64(v)*mapping.Scale + mapping.Offset, nil
	case uint32:
		return float64(v)*mapping.Scale + mapping.Offset, nil
	case float32:
		return float64(v)*mapping.Scale + mapping.Offset, nil
	case float64:
		return v*mapping.Scale + mapping.Offset, nil
	}

	return value, err
}

// registersToInt32 converts 2 registers to int32.
func (t *Transformer) registersToInt32(registers []uint16, byteOrder binary.ByteOrder, swapWords bool) int32 {
	var value uint32

	if swapWords {
		value = uint32(registers[1])<<16 | uint32(registers[0])
	} else {
		value = uint32(registers[0])<<16 | uint32(registers[1])
	}

	return int32(value)
}

// registersToUint32 converts 2 registers to uint32.
func (t *Transformer) registersToUint32(registers []uint16, byteOrder binary.ByteOrder, swapWords bool) uint32 {
	if swapWords {
		return uint32(registers[1])<<16 | uint32(registers[0])
	}
	return uint32(registers[0])<<16 | uint32(registers[1])
}

// registersToFloat32 converts 2 registers to float32.
func (t *Transformer) registersToFloat32(registers []uint16, byteOrder binary.ByteOrder, swapWords bool) float32 {
	bits := t.registersToUint32(registers, byteOrder, swapWords)
	return math.Float32frombits(bits)
}

// registersToFloat64 converts 4 registers to float64.
func (t *Transformer) registersToFloat64(registers []uint16, byteOrder binary.ByteOrder, swapWords bool) float64 {
	var bits uint64

	if swapWords {
		bits = uint64(registers[3])<<48 | uint64(registers[2])<<32 |
			uint64(registers[1])<<16 | uint64(registers[0])
	} else {
		bits = uint64(registers[0])<<48 | uint64(registers[1])<<32 |
			uint64(registers[2])<<16 | uint64(registers[3])
	}

	return math.Float64frombits(bits)
}

// BitMask defines bit manipulation operations.
type BitMask struct {
	Address uint16
	Mask    uint16
	Shift   uint8
}

// ApplyBitMask applies a bit mask to a register value.
func ApplyBitMask(value uint16, mask *BitMask) uint16 {
	return (value & mask.Mask) >> mask.Shift
}

// SetBits sets specific bits in a register value.
func SetBits(value uint16, mask *BitMask, newValue uint16) uint16 {
	// Clear the bits
	value &= ^mask.Mask

	// Set new bits
	value |= (newValue << mask.Shift) & mask.Mask

	return value
}
