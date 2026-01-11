#!/usr/bin/env python3
"""
Simple Modbus TCP test script to verify proxy functionality.
Tests connection to 192.168.178.103:502
"""

import socket
import struct
import sys

def create_modbus_read_request(transaction_id, unit_id, function_code, start_addr, quantity):
    """Create a Modbus TCP read request."""
    # MBAP Header: Transaction ID (2), Protocol ID (2), Length (2), Unit ID (1)
    # PDU: Function Code (1), Start Address (2), Quantity (2)

    protocol_id = 0
    length = 6  # Unit ID (1) + Function Code (1) + Start Addr (2) + Quantity (2)

    # Pack MBAP header
    mbap = struct.pack('>HHHB', transaction_id, protocol_id, length, unit_id)

    # Pack PDU
    pdu = struct.pack('>BHH', function_code, start_addr, quantity)

    return mbap + pdu

def parse_modbus_response(response):
    """Parse a Modbus TCP response."""
    if len(response) < 9:
        return None, "Response too short"

    # Parse MBAP header
    tx_id, proto_id, length, unit_id = struct.unpack('>HHHB', response[:8])

    # Parse PDU
    function_code = response[7]

    # Check for exception
    if function_code & 0x80:
        exception_code = response[8]
        return None, f"Modbus Exception: {exception_code}"

    # For read functions (03, 04), parse data
    if function_code in [0x03, 0x04]:
        byte_count = response[8]
        data = response[9:9+byte_count]
        # Convert to 16-bit registers
        registers = []
        for i in range(0, len(data), 2):
            reg_value = struct.unpack('>H', data[i:i+2])[0]
            registers.append(reg_value)
        return registers, None

    return response[8:], None

def test_modbus_connection(host, port):
    """Test Modbus TCP connection."""
    print(f"Testing Modbus connection to {host}:{port}")

    try:
        # Create socket
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5.0)

        # Connect
        print(f"Connecting to {host}:{port}...")
        sock.connect((host, port))
        print("[OK] Connected successfully")

        # Create a read holding registers request
        # Function Code 03 (Read Holding Registers)
        # Start Address: 0
        # Quantity: 10 registers
        transaction_id = 1
        unit_id = 1
        function_code = 0x03  # Read Holding Registers
        start_addr = 0
        quantity = 10

        request = create_modbus_read_request(transaction_id, unit_id, function_code, start_addr, quantity)

        print(f"Sending read request (FC={function_code}, Start={start_addr}, Qty={quantity})...")
        print(f"Request bytes: {request.hex()}")

        # Send request
        sock.sendall(request)

        # Receive response
        response = sock.recv(1024)
        print(f"Received {len(response)} bytes")
        print(f"Response bytes: {response.hex()}")

        # Parse response
        registers, error = parse_modbus_response(response)

        if error:
            print(f"[ERROR] {error}")
            return False
        else:
            print(f"[OK] Successfully read {len(registers)} registers:")
            for i, reg in enumerate(registers):
                print(f"  Register {start_addr + i}: {reg} (0x{reg:04X})")
            return True

    except socket.timeout:
        print("[ERROR] Connection timeout")
        return False
    except ConnectionRefusedError:
        print("[ERROR] Connection refused")
        return False
    except Exception as e:
        print(f"[ERROR] {e}")
        return False
    finally:
        sock.close()
        print("Connection closed")

if __name__ == "__main__":
    host = "192.168.178.103"
    port = 502

    if len(sys.argv) > 1:
        host = sys.argv[1]
    if len(sys.argv) > 2:
        port = int(sys.argv[2])

    success = test_modbus_connection(host, port)
    sys.exit(0 if success else 1)
