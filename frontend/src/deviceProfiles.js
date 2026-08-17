// Device profiles: starting points for devices with known Modbus quirks.
//
// Applying a profile only fills the proxy form — nothing is applied globally
// and existing proxies keep their settings until a profile is chosen for them.
// Labels and hints live in i18n under `control.profiles.<id>`.

export const deviceProfiles = [
    {
        id: 'standard',
        values: {
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
            connection_timeout: 10,
            read_timeout: 10,
            max_retries: 1,
            max_read_size: 0,
            connect_delay_ms: 3000,
            max_target_conns: 1,
            min_request_gap_ms: 200,
            request_timeout_ms: 0
        }
    }
];

export const findDeviceProfile = (id) => deviceProfiles.find((profile) => profile.id === id);
