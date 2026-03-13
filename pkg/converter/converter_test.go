package converter

import (
	"bytes"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultTransactionID != 0x0001 {
		t.Errorf("Expected default transaction ID 0x0001, got 0x%04X", cfg.DefaultTransactionID)
	}

	if cfg.DefaultUnitID != 0x01 {
		t.Errorf("Expected default unit ID 0x01, got 0x%02X", cfg.DefaultUnitID)
	}

	if cfg.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", cfg.Timeout)
	}
}

func TestNewConverter(t *testing.T) {
	c := NewConverter(nil)

	if c == nil {
		t.Fatal("Converter should not be nil")
	}

	if c.config.DefaultTransactionID != 0x0001 {
		t.Errorf("Expected default transaction ID 0x0001, got 0x%04X", c.config.DefaultTransactionID)
	}
}

func TestTCPToRTU_ReadHoldingRegisters(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame: TransactionID(0001) + ProtocolID(0000) + Length(0006) + UnitID(01) + Function(03) + StartAddr(0000) + Quantity(000A)
	tcpFrame := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x0A}

	rtuFrame, err := c.TCPToRTU(tcpFrame)
	if err != nil {
		t.Fatalf("Failed to convert TCP to RTU: %v", err)
	}

	// Expected RTU: SlaveID(01) + Function(03) + StartAddr(0000) + Quantity(000A) + CRC
	expectedLength := 1 + 1 + 4 + 2 // 8 bytes
	if len(rtuFrame) != expectedLength {
		t.Errorf("Expected RTU frame length %d, got %d", expectedLength, len(rtuFrame))
	}

	// Check SlaveID
	if rtuFrame[0] != 0x01 {
		t.Errorf("Expected SlaveID 0x01, got 0x%02X", rtuFrame[0])
	}

	// Check Function
	if rtuFrame[1] != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", rtuFrame[1])
	}
}

func TestTCPToRTU_WriteSingleRegister(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame for write single register
	tcpFrame := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x06, 0x00, 0x00, 0x00, 0x0A}

	rtuFrame, err := c.TCPToRTU(tcpFrame)
	if err != nil {
		t.Fatalf("Failed to convert TCP to RTU: %v", err)
	}

	if len(rtuFrame) < 8 {
		t.Errorf("Expected RTU frame length >= 8, got %d", len(rtuFrame))
	}

	if rtuFrame[0] != 0x01 {
		t.Errorf("Expected SlaveID 0x01, got 0x%02X", rtuFrame[0])
	}

	if rtuFrame[1] != 0x06 {
		t.Errorf("Expected Function 0x06, got 0x%02X", rtuFrame[1])
	}
}

func TestTCPToRTU_InvalidProtocol(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame with invalid protocol ID
	tcpFrame := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x06, 0x01, 0x03}

	_, err := c.TCPToRTU(tcpFrame)
	if err == nil {
		t.Error("Expected error for invalid protocol ID")
	}
}

func TestTCPToRTU_TooShort(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame too short
	tcpFrame := []byte{0x00, 0x01}

	_, err := c.TCPToRTU(tcpFrame)
	if err == nil {
		t.Error("Expected error for too short frame")
	}
}

func TestTCPToRTU_LengthMismatch(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame with incorrect length in header
	tcpFrame := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x0A, 0x01, 0x03} // Length says 10 but actual is 2

	_, err := c.TCPToRTU(tcpFrame)
	if err == nil {
		t.Error("Expected error for length mismatch")
	}
}

func TestRTUToTCP_ReadHoldingRegistersResponse(t *testing.T) {
	c := NewConverter(nil)

	// RTU response: SlaveID(01) + Function(03) + ByteCount(02) + Data(000A) + CRC(3843)
	rtuFrame := []byte{0x01, 0x03, 0x02, 0x00, 0x0A, 0x38, 0x43}

	tcpFrame, err := c.RTUToTCP(rtuFrame, 0x0001)
	if err != nil {
		t.Fatalf("Failed to convert RTU to TCP: %v", err)
	}

	// Expected TCP: TransactionID(2) + ProtocolID(2) + Length(2) + UnitID(1) + Function(1) + Data(2)
	// Note: ByteCount is excluded from TCP data (only needed in RTU for CRC calculation)
	// Length field = UnitID + Function + Data = 1 + 1 + 2 = 4
	// Total: 7 + 4 = 11 bytes
	expectedLength := 11
	if len(tcpFrame) != expectedLength {
		t.Errorf("Expected TCP frame length %d, got %d", expectedLength, len(tcpFrame))
	}

	// Check Transaction ID
	transactionID := tcpFrame[0]<<8 | tcpFrame[1]
	if transactionID != 0x0001 {
		t.Errorf("Expected TransactionID 0x0001, got 0x%04X", transactionID)
	}

	// Check Protocol ID
	protocolID := tcpFrame[2]<<8 | tcpFrame[3]
	if protocolID != 0 {
		t.Errorf("Expected ProtocolID 0, got 0x%04X", protocolID)
	}

	// Check Unit ID
	if tcpFrame[6] != 0x01 {
		t.Errorf("Expected UnitID 0x01, got 0x%02X", tcpFrame[6])
	}

	// Check Function
	if tcpFrame[7] != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", tcpFrame[7])
	}

	// Check ByteCount (should be 2)
	if tcpFrame[8] != 0x02 {
		t.Errorf("Expected ByteCount 0x02, got 0x%02X", tcpFrame[8])
	}
}

func TestRTUToTCP_WriteSingleRegisterResponse(t *testing.T) {
	c := NewConverter(nil)

	// RTU response: SlaveID(01) + Function(06) + Address(0000) + Value(000A) + CRC(09CD)
	rtuFrame := []byte{0x01, 0x06, 0x00, 0x00, 0x00, 0x0A, 0x09, 0xCD}

	tcpFrame, err := c.RTUToTCP(rtuFrame, 0x0002)
	if err != nil {
		t.Fatalf("Failed to convert RTU to TCP: %v", err)
	}

	if len(tcpFrame) < 8 {
		t.Errorf("Expected TCP frame length >= 8, got %d", len(tcpFrame))
	}

	// Check Transaction ID
	transactionID := tcpFrame[0]<<8 | tcpFrame[1]
	if transactionID != 0x0002 {
		t.Errorf("Expected TransactionID 0x0002, got 0x%04X", transactionID)
	}
}

func TestRTUToTCP_CRCMismatch(t *testing.T) {
	c := NewConverter(nil)

	// RTU frame with invalid CRC
	rtuFrame := []byte{0x01, 0x03, 0x02, 0x00, 0x0A, 0xFF, 0xFF}

	_, err := c.RTUToTCP(rtuFrame, 0x0001)
	if err == nil {
		t.Error("Expected error for CRC mismatch")
	}
}

func TestRTUToTCP_TooShort(t *testing.T) {
	c := NewConverter(nil)

	// RTU frame too short
	rtuFrame := []byte{0x01, 0x03}

	_, err := c.RTUToTCP(rtuFrame, 0x0001)
	if err == nil {
		t.Error("Expected error for too short frame")
	}
}

func TestParseTCPFrame(t *testing.T) {
	c := NewConverter(nil)

	// TCP frame
	data := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x0A}

	frame, err := c.ParseTCPFrame(data)
	if err != nil {
		t.Fatalf("Failed to parse TCP frame: %v", err)
	}

	if frame.Header.TransactionID != 0x0001 {
		t.Errorf("Expected TransactionID 0x0001, got 0x%04X", frame.Header.TransactionID)
	}

	if frame.Header.UnitID != 0x01 {
		t.Errorf("Expected UnitID 0x01, got 0x%02X", frame.Header.UnitID)
	}

	if frame.Function != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", frame.Function)
	}
}

func TestParseRTUFrame(t *testing.T) {
	c := NewConverter(nil)

	// RTU frame
	data := []byte{0x01, 0x03, 0x02, 0x00, 0x0A, 0x38, 0x43}

	frame, err := c.ParseRTUFrame(data)
	if err != nil {
		t.Fatalf("Failed to parse RTU frame: %v", err)
	}

	if frame.SlaveID != 0x01 {
		t.Errorf("Expected SlaveID 0x01, got 0x%02X", frame.SlaveID)
	}

	if frame.Function != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", frame.Function)
	}

	if len(frame.Data) != 2 {
		t.Errorf("Expected data length 2, got %d", len(frame.Data))
	}
}

func TestParseRTUFrame_CRCMismatch(t *testing.T) {
	c := NewConverter(nil)

	// RTU frame with invalid CRC
	data := []byte{0x01, 0x03, 0x02, 0x00, 0x0A, 0xFF, 0xFF}

	_, err := c.ParseRTUFrame(data)
	if err == nil {
		t.Error("Expected error for CRC mismatch")
	}
}

func TestBuildTCPFrame(t *testing.T) {
	c := NewConverter(nil)

	frame := c.BuildTCPFrame(0x0001, 0x01, 0x03, []byte{0x00, 0x00, 0x00, 0x0A})

	if len(frame) != 12 {
		t.Errorf("Expected frame length 12, got %d", len(frame))
	}

	// Check Transaction ID
	transactionID := frame[0]<<8 | frame[1]
	if transactionID != 0x0001 {
		t.Errorf("Expected TransactionID 0x0001, got 0x%04X", transactionID)
	}

	// Check Protocol ID
	protocolID := frame[2]<<8 | frame[3]
	if protocolID != 0 {
		t.Errorf("Expected ProtocolID 0, got 0x%04X", protocolID)
	}

	// Check Unit ID
	if frame[6] != 0x01 {
		t.Errorf("Expected UnitID 0x01, got 0x%02X", frame[6])
	}

	// Check Function
	if frame[7] != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", frame[7])
	}
}

func TestBuildRTUFrame(t *testing.T) {
	c := NewConverter(nil)

	frame := c.BuildRTUFrame(0x01, 0x03, []byte{0x00, 0x00, 0x00, 0x0A})

	if len(frame) != 8 { // 2 bytes header + 4 bytes data + 2 bytes CRC
		t.Errorf("Expected frame length 8, got %d", len(frame))
	}

	// Check SlaveID
	if frame[0] != 0x01 {
		t.Errorf("Expected SlaveID 0x01, got 0x%02X", frame[0])
	}

	// Check Function
	if frame[1] != 0x03 {
		t.Errorf("Expected Function 0x03, got 0x%02X", frame[1])
	}

	// Check CRC (last 2 bytes)
	if len(frame) < 2 {
		t.Fatal("Frame too short to check CRC")
	}

	crc := calculateCRC(frame[:len(frame)-2])
	if frame[len(frame)-2] != crc[0] || frame[len(frame)-1] != crc[1] {
		t.Errorf("CRC mismatch: expected %02X%02X", crc[0], crc[1])
	}
}

func TestGetNextTransactionID(t *testing.T) {
	c := NewConverter(nil)

	// Get first ID
	id1 := c.GetNextTransactionID()
	if id1 != 0x0001 {
		t.Errorf("Expected transaction ID 0x0001, got 0x%04X", id1)
	}

	// Get second ID
	id2 := c.GetNextTransactionID()
	if id2 != 0x0002 {
		t.Errorf("Expected transaction ID 0x0002, got 0x%04X", id2)
	}
}

func TestGetNextTransactionID_WrapAround(t *testing.T) {
	c := NewConverter(nil)
	c.transactionID = 0xFFFF

	// Should return 0xFFFF, then next call returns 0x0001
	id1 := c.GetNextTransactionID()
	if id1 != 0xFFFF {
		t.Errorf("Expected transaction ID 0xFFFF, got 0x%04X", id1)
	}

	id2 := c.GetNextTransactionID()
	if id2 != 0x0001 {
		t.Errorf("Expected transaction ID 0x0001 after wrap, got 0x%04X", id2)
	}
}

func TestConvertExceptionToTCP(t *testing.T) {
	c := NewConverter(nil)

	// RTU exception: SlaveID(01) + Function(83) + ExceptionCode(02) + CRC
	rtuException := []byte{0x01, 0x83, 0x02, 0xC0, 0x81}

	tcpFrame, err := c.ConvertExceptionToTCP(rtuException, 0x0001)
	if err != nil {
		t.Fatalf("Failed to convert exception: %v", err)
	}

	if len(tcpFrame) < 8 {
		t.Errorf("Expected TCP frame length >= 8, got %d", len(tcpFrame))
	}

	// Check Transaction ID
	transactionID := tcpFrame[0]<<8 | tcpFrame[1]
	if transactionID != 0x0001 {
		t.Errorf("Expected TransactionID 0x0001, got 0x%04X", transactionID)
	}

	// Check Exception bit is set
	if tcpFrame[7]&0x80 == 0 {
		t.Error("Exception bit should be set in function code")
	}
}

func TestConvertExceptionToRTU(t *testing.T) {
	c := NewConverter(nil)

	// TCP exception: TransactionID(0001) + ProtocolID(0000) + Length(0002) + UnitID(01) + Function(83) + ExceptionCode(02)
	tcpException := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x01, 0x83, 0x02}

	rtuFrame, err := c.ConvertExceptionToRTU(tcpException)
	if err != nil {
		t.Fatalf("Failed to convert exception: %v", err)
	}

	if len(rtuFrame) != 5 { // SlaveID + Function + ExceptionCode + CRC
		t.Errorf("Expected RTU frame length 5, got %d", len(rtuFrame))
	}

	// Check SlaveID
	if rtuFrame[0] != 0x01 {
		t.Errorf("Expected SlaveID 0x01, got 0x%02X", rtuFrame[0])
	}

	// Check Exception bit is set
	if rtuFrame[1]&0x80 == 0 {
		t.Error("Exception bit should be set in function code")
	}

	// Check CRC
	crc := calculateCRC(rtuFrame[:len(rtuFrame)-2])
	if rtuFrame[3] != crc[0] || rtuFrame[4] != crc[1] {
		t.Errorf("CRC mismatch: expected %02X%02X", crc[0], crc[1])
	}
}

func TestValidateTCPFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   []byte
		wantErr bool
	}{
		{
			name:    "valid frame",
			frame:   []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x0A},
			wantErr: false,
		},
		{
			name:    "too short",
			frame:   []byte{0x00, 0x01},
			wantErr: true,
		},
		{
			name:    "invalid protocol ID",
			frame:   []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x06, 0x01, 0x03},
			wantErr: true,
		},
		{
			name:    "length too large",
			frame:   []byte{0x00, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x03},
			wantErr: true,
		},
		{
			name:    "length mismatch",
			frame:   []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x0A, 0x01, 0x03},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTCPFrame(tt.frame)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTCPFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRTUFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   []byte
		wantErr bool
	}{
		{
			name:    "valid frame",
			frame:   []byte{0x01, 0x03, 0x02, 0x00, 0x0A},
			wantErr: false,
		},
		{
			name:    "too short",
			frame:   []byte{0x01},
			wantErr: true,
		},
		{
			name:    "invalid function code too low",
			frame:   []byte{0x01, 0x00, 0x02, 0x00},
			wantErr: true,
		},
		{
			name:    "invalid function code too high",
			frame:   []byte{0x01, 0x80, 0x02, 0x00},
			wantErr: true,
		},
		{
			name:    "valid exception response",
			frame:   []byte{0x01, 0x83, 0x02},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRTUFrame(tt.frame)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRTUFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStreamConverter_ConvertTCPToRTUStream(t *testing.T) {
	sc := NewStreamConverter(nil)

	// Create TCP reader with two frames
	tcpData := []byte{
		// Frame 1: Read holding registers
		0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x0A,
		// Frame 2: Write single register
		0x00, 0x02, 0x00, 0x00, 0x00, 0x06, 0x01, 0x06, 0x00, 0x00, 0x00, 0x0A,
	}

	tcpReader := bytes.NewReader(tcpData)
	var rtuWriter bytes.Buffer

	err := sc.ConvertTCPToRTUStream(tcpReader, &rtuWriter)
	if err != nil {
		t.Fatalf("Failed to convert stream: %v", err)
	}

	rtuData := rtuWriter.Bytes()

	// Should have 2 RTU frames
	// Frame 1: 8 bytes (including CRC)
	// Frame 2: 8 bytes (including CRC)
	// Total: 16 bytes
	expectedLength := 16
	if len(rtuData) != expectedLength {
		t.Errorf("Expected RTU data length %d, got %d", expectedLength, len(rtuData))
	}

	// Check first frame starts with SlaveID 0x01
	if rtuData[0] != 0x01 {
		t.Errorf("Expected first frame SlaveID 0x01, got 0x%02X", rtuData[0])
	}
}

func TestRoundTrip_TCP_To_RTU_To_TCP(t *testing.T) {
	c := NewConverter(nil)

	// Use WRITE SINGLE REGISTER (function 0x06) which has same format in both directions
	// Original TCP frame
	originalTCP := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x06, 0x00, 0x01, 0x00, 0x0A}

	// Convert TCP to RTU
	rtuFrame, err := c.TCPToRTU(originalTCP)
	if err != nil {
		t.Fatalf("Failed to convert TCP to RTU: %v", err)
	}

	// Convert RTU back to TCP
	reconstructedTCP, err := c.RTUToTCP(rtuFrame, 0x0001)
	if err != nil {
		t.Fatalf("Failed to convert RTU to TCP: %v", err)
	}

	// Compare the data part (after MBAP header)
	// Both should be: UnitID(01) + Function(06) + Address(0001) + Value(000A)
	if len(reconstructedTCP) < 12 {
		t.Fatalf("Reconstructed TCP frame too short: %d bytes", len(reconstructedTCP))
	}

	if !bytes.Equal(originalTCP[6:], reconstructedTCP[6:]) {
		t.Errorf("Data mismatch after round trip:\nOriginal:      %v\nReconstructed: %v",
			originalTCP[6:], reconstructedTCP[6:])
	}
}

func TestRoundTrip_RTU_To_TCP_To_RTU(t *testing.T) {
	c := NewConverter(nil)

	// Use WRITE SINGLE REGISTER response (function 0x06) which has same format in both directions
	// Original RTU frame: SlaveID(01) + Function(06) + Address(0001) + Value(000A) + CRC(580D)
	originalRTU := []byte{0x01, 0x06, 0x00, 0x01, 0x00, 0x0A, 0x58, 0x0D}

	// Convert RTU to TCP
	tcpFrame, err := c.RTUToTCP(originalRTU, 0x0001)
	if err != nil {
		t.Fatalf("Failed to convert RTU to TCP: %v", err)
	}

	// Convert TCP back to RTU
	reconstructedRTU, err := c.TCPToRTU(tcpFrame)
	if err != nil {
		t.Fatalf("Failed to convert TCP to RTU: %v", err)
	}

	// Compare data (excluding CRC which is last 2 bytes)
	if len(originalRTU) < 5 || len(reconstructedRTU) < 5 {
		t.Fatalf("Frames too short for comparison")
	}

	// Compare SlaveID, Function, and Data
	if !bytes.Equal(originalRTU[:len(originalRTU)-2], reconstructedRTU[:len(reconstructedRTU)-2]) {
		t.Errorf("Data mismatch after round trip:\nOriginal:      %v\nReconstructed: %v",
			originalRTU[:len(originalRTU)-2], reconstructedRTU[:len(reconstructedRTU)-2])
	}
}
