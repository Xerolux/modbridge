package modbus

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// BenchmarkReadFrame benchmarks the ReadFrame function with pooling
func BenchmarkReadFrame(b *testing.B) {
	// Create a sample Modbus frame
	frame := make([]byte, 12) // 6 byte header + 6 byte payload
	binary.BigEndian.PutUint16(frame[0:2], 1)    // Transaction ID
	binary.BigEndian.PutUint16(frame[2:4], 0)    // Protocol ID
	binary.BigEndian.PutUint16(frame[4:6], 6)    // Length
	frame[6] = 1                                  // Unit ID
	frame[7] = 3                                  // Function code
	binary.BigEndian.PutUint16(frame[8:10], 100) // Starting address
	binary.BigEndian.PutUint16(frame[10:12], 10) // Quantity

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(frame)
		_, err := ReadFrame(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadFrameLarge benchmarks with larger frames
func BenchmarkReadFrameLarge(b *testing.B) {
	// Create a larger Modbus frame (max read response)
	payloadSize := 253
	frame := make([]byte, 6+payloadSize)
	binary.BigEndian.PutUint16(frame[0:2], 1)             // Transaction ID
	binary.BigEndian.PutUint16(frame[2:4], 0)             // Protocol ID
	binary.BigEndian.PutUint16(frame[4:6], uint16(payloadSize)) // Length
	frame[6] = 1                                           // Unit ID
	frame[7] = 3                                           // Function code
	frame[8] = byte(payloadSize - 2)                       // Byte count

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(frame)
		_, err := ReadFrame(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}
