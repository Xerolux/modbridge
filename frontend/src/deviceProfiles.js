// Device profiles: starting points for devices with known Modbus quirks.
//
// Applying a profile only fills the proxy form — nothing is applied globally
// and existing proxies keep their settings until a profile is chosen for them.
// Labels and hints live in i18n under `control.profiles.<id>`.
//
// Every profile sets every field it manages, so switching between profiles is
// idempotent and 'standard' always restores the ModBridge defaults.
//
// The values describe a behaviour class, not a vendor specification: how many
// Modbus sessions the device serves, how much breathing room it needs between
// requests, and how long an answer may take. Devices of the same class behave
// alike even when the firmware differs, so treat a profile as a starting point
// and adjust from there.

export const deviceProfiles = [
    {
        // ModBridge defaults.
        id: 'standard',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 30,
            max_retries: 3,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 0,
            min_request_gap_ms: 0,
            request_timeout_ms: 0
        }
    },
    {
        // SolarEdge and other SunSpec inverters serve a single Modbus session
        // and need requests spaced out. The request budget stays below the 3s
        // timeout used by common Home Assistant integrations.
        id: 'sunspec',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 2,
            max_retries: 1,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 1,
            min_request_gap_ms: 100,
            request_timeout_ms: 2500
        }
    },
    {
        // Huawei inverters/sDongles ignore requests that arrive right after the
        // TCP handshake and also accept only one session.
        id: 'huawei',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 10,
            max_retries: 1,
            max_read_size: 0,
            connect_delay_ms: 3000,
            max_target_conns: 1,
            min_request_gap_ms: 200,
            request_timeout_ms: 0
        }
    },
    {
        // SMA Speedwire/Modbus serves very few clients; one session with a
        // small gap keeps it stable.
        id: 'sma',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 1,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 1,
            min_request_gap_ms: 50,
            request_timeout_ms: 0
        }
    },
    {
        // Fronius Datamanager/GEN24 handles a couple of parallel sessions but
        // gets slow under more.
        id: 'fronius',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 2,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 2,
            min_request_gap_ms: 50,
            request_timeout_ms: 0
        }
    },
    {
        // Kostal Plenticore/PIKO answers slowly and only on one session.
        id: 'kostal',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 1,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 1,
            min_request_gap_ms: 100,
            request_timeout_ms: 0
        }
    },
    {
        // Victron GX/Venus OS is a fast TCP server that serves many clients.
        id: 'victron',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 3,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 0,
            min_request_gap_ms: 0,
            request_timeout_ms: 0
        }
    },
    {
        // TCP-to-RTU gateways share one serial line across all requests, so
        // parallel sessions only queue up behind each other. 125 registers is
        // the Modbus limit for a single read of holding/input registers.
        id: 'rtuGateway',
        values: {
            protocol: 'tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 2,
            max_read_size: 125,
            connect_delay_ms: 0,
            max_target_conns: 1,
            min_request_gap_ms: 50,
            request_timeout_ms: 0
        }
    },
    {
        // Serial-to-WiFi/Ethernet adapters that forward raw RTU frames without
        // an MBAP header (Waveshare, USR, Elfin and similar).
        id: 'rtuOverTcp',
        values: {
            protocol: 'rtu-tcp',
            connection_timeout: 10,
            read_timeout: 5,
            max_retries: 2,
            max_read_size: 125,
            connect_delay_ms: 0,
            max_target_conns: 1,
            min_request_gap_ms: 50,
            request_timeout_ms: 0
        }
    },
    {
        // PLCs and automation controllers: fast, and built for many clients.
        id: 'plc',
        values: {
            protocol: 'tcp',
            connection_timeout: 5,
            read_timeout: 5,
            max_retries: 2,
            max_read_size: 0,
            connect_delay_ms: 0,
            max_target_conns: 0,
            min_request_gap_ms: 0,
            request_timeout_ms: 0
        }
    }
];

export const findDeviceProfile = (id) => deviceProfiles.find((profile) => profile.id === id);
