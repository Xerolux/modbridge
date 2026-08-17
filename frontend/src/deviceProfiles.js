// Device profiles: starting points for devices with known Modbus behaviour.
//
// Applying a profile only fills the proxy form — nothing is applied globally
// and existing proxies keep their settings until a profile is chosen for them.
//
// HOW THIS IS BUILT
//
// The settings live in `profileClasses` below, not on the individual devices.
// A class describes how a device behaves on the wire: how many Modbus sessions
// it serves, how much room it needs between requests, how long an answer may
// take, and whether it speaks Modbus TCP or raw RTU. Each device entry maps to
// one class, optionally with a note.
//
// That split is deliberate. Which class a device belongs to is knowable from
// how it behaves in the field; exact per-model timings are not, and inventing
// them per device would dress up guesses as vendor specifications. So a
// profile is a starting point that keeps a device stable — tune from there.
//
// Devices are listed only when they actually expose Modbus (natively or via a
// documented adapter). Entries marked with the `viaGateway` note reach Modbus
// through a separate adapter rather than on their own network port.
//
// Class values in full — every class sets every field it manages, so switching
// profiles is idempotent and 'standard' restores the ModBridge defaults.

export const profileClasses = {
    // ModBridge defaults.
    standard: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 30,
        max_retries: 3,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 0,
        min_request_gap_ms: 0,
        request_timeout_ms: 0
    },
    // Fast TCP servers built for many clients.
    multiSession: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 5,
        max_retries: 3,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 0,
        min_request_gap_ms: 0,
        request_timeout_ms: 0
    },
    // Handles a couple of parallel sessions, slows down beyond that.
    fewSessions: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 5,
        max_retries: 2,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 2,
        min_request_gap_ms: 50,
        request_timeout_ms: 0
    },
    // Serves exactly one Modbus session and ignores the rest.
    singleSession: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 5,
        max_retries: 1,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 1,
        min_request_gap_ms: 100,
        request_timeout_ms: 0
    },
    // One session, and the budget stays below the 3s timeout that common Home
    // Assistant integrations use.
    singleSessionFast: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 2,
        max_retries: 1,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 1,
        min_request_gap_ms: 100,
        request_timeout_ms: 2500
    },
    // One session on a controller that takes its time — heating controllers and
    // inverter logger sticks.
    singleSessionSlow: {
        protocol: 'tcp',
        connection_timeout: 15,
        read_timeout: 10,
        max_retries: 1,
        max_read_size: 0,
        connect_delay_ms: 0,
        max_target_conns: 1,
        min_request_gap_ms: 250,
        request_timeout_ms: 0
    },
    // Drops requests that arrive right after the TCP handshake.
    connectDelay: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 10,
        max_retries: 1,
        max_read_size: 0,
        connect_delay_ms: 3000,
        max_target_conns: 1,
        min_request_gap_ms: 200,
        request_timeout_ms: 0
    },
    // TCP-to-RTU gateway: every request shares one serial line. 125 registers is
    // the Modbus limit for a single read of holding/input registers.
    serialGateway: {
        protocol: 'tcp',
        connection_timeout: 10,
        read_timeout: 5,
        max_retries: 2,
        max_read_size: 125,
        connect_delay_ms: 0,
        max_target_conns: 1,
        min_request_gap_ms: 50,
        request_timeout_ms: 0
    },
    // Serial adapter forwarding raw RTU frames without an MBAP header.
    rtuOverTcp: {
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
};

// Device labels are brand names and stay untranslated. Category names, class
// descriptions and notes come from i18n (`control.profiles.*`).
export const deviceCategories = [
    {
        id: 'generic',
        devices: [
            { id: 'standard', label: 'Standard', class: 'standard' },
            { id: 'plc', label: 'SPS / PLC (Siemens, Beckhoff, WAGO)', class: 'multiSession' },
            { id: 'rtuGateway', label: 'Modbus-TCP → RTU Gateway (Wago, Moxa, USR)', class: 'serialGateway' },
            { id: 'rtuAdapter', label: 'Serieller Adapter (Waveshare, USR, Elfin)', class: 'rtuOverTcp', note: 'rawRtu' }
        ]
    },
    {
        id: 'inverter',
        devices: [
            { id: 'solaredge', label: 'SolarEdge (SunSpec)', class: 'singleSessionFast', note: 'haBudget' },
            { id: 'sma', label: 'SMA (Speedwire / Modbus)', class: 'fewSessions' },
            { id: 'fronius', label: 'Fronius (Symo, GEN24)', class: 'fewSessions' },
            { id: 'kostal', label: 'Kostal (Plenticore, PIKO)', class: 'singleSession' },
            { id: 'huawei', label: 'Huawei SUN2000 / sDongle', class: 'connectDelay', note: 'connectDelay' },
            { id: 'sungrow', label: 'Sungrow (SG, SH, WiNet-S)', class: 'singleSessionSlow', note: 'dongle' },
            { id: 'goodwe', label: 'GoodWe (ET, EH)', class: 'singleSession' },
            { id: 'growatt', label: 'Growatt (ShineLAN, ShineWiFi)', class: 'singleSessionSlow', note: 'dongle' },
            { id: 'solax', label: 'SolaX (Pocket LAN / WiFi)', class: 'singleSessionSlow', note: 'dongle' },
            { id: 'deye', label: 'Deye / Sunsynk', class: 'singleSessionSlow', note: 'dongle' },
            { id: 'sofar', label: 'Sofar Solar (LSW-3 / LSE)', class: 'singleSessionSlow', note: 'dongle' },
            { id: 'delta', label: 'Delta (RPI, M-Serie)', class: 'singleSession' },
            { id: 'kaco', label: 'KACO (blueplanet)', class: 'singleSession' },
            { id: 'fimer', label: 'FIMER / ABB (VSN300)', class: 'singleSession' },
            { id: 'e3dc', label: 'E3/DC (S10)', class: 'fewSessions', note: 'enableFirst' },
            { id: 'victron', label: 'Victron GX / Venus OS', class: 'multiSession' },
            { id: 'sunspecGeneric', label: 'SunSpec-Wechselrichter (allgemein)', class: 'singleSession' }
        ]
    },
    {
        id: 'heatpump',
        devices: [
            { id: 'idm', label: 'IDM (Navigator 2.0 / Navigator 10)', class: 'singleSessionSlow', note: 'pollSlowly' },
            { id: 'stiebel', label: 'Stiebel Eltron (ISG + Modbus)', class: 'singleSessionSlow', note: 'pollSlowly' },
            { id: 'tecalor', label: 'Tecalor (ISG + Modbus)', class: 'singleSessionSlow', note: 'pollSlowly' },
            { id: 'nibeS', label: 'NIBE S-Serie', class: 'singleSession' },
            { id: 'nibeModbus40', label: 'NIBE F-Serie (MODBUS 40)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'lambda', label: 'Lambda Wärmepumpen', class: 'singleSession' },
            { id: 'waterkotte', label: 'Waterkotte (EcoTouch)', class: 'singleSessionSlow' },
            { id: 'ochsner', label: 'Ochsner (OTE)', class: 'singleSessionSlow' },
            { id: 'nilan', label: 'Nilan (CTS602)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'daikin', label: 'Daikin Altherma (Modbus-Adapter)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'panasonic', label: 'Panasonic Aquarea (CZ-TAW1)', class: 'singleSessionSlow', note: 'viaGateway' },
            { id: 'lgThermaV', label: 'LG Therma V (PI485)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'ecodan', label: 'Mitsubishi Ecodan (Modbus-Gateway)', class: 'serialGateway', note: 'viaGateway' }
        ]
    },
    {
        id: 'ventilation',
        devices: [
            { id: 'helios', label: 'Helios KWL (easyControls)', class: 'singleSession' },
            { id: 'zehnder', label: 'Zehnder ComfoAir Q', class: 'singleSession' },
            { id: 'vallox', label: 'Vallox', class: 'singleSession' },
            { id: 'pluggit', label: 'Pluggit', class: 'singleSession' },
            { id: 'wolfCwl', label: 'Wolf CWL', class: 'serialGateway', note: 'viaGateway' }
        ]
    },
    {
        id: 'meter',
        devices: [
            { id: 'eastron', label: 'Eastron SDM (72, 120, 230, 630)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'carloGavazzi', label: 'Carlo Gavazzi EM24 / EM340', class: 'serialGateway', note: 'viaGateway' },
            { id: 'janitza', label: 'Janitza UMG', class: 'fewSessions' },
            { id: 'schneiderIem', label: 'Schneider iEM3000', class: 'fewSessions' },
            { id: 'siemensPac', label: 'Siemens SENTRON PAC', class: 'fewSessions' },
            { id: 'abbMeter', label: 'ABB B23 / B24', class: 'serialGateway', note: 'viaGateway' },
            { id: 'finder', label: 'Finder 7M', class: 'serialGateway', note: 'viaGateway' },
            { id: 'iskra', label: 'Iskra WM3', class: 'serialGateway', note: 'viaGateway' },
            { id: 'shellyPro', label: 'Shelly Pro EM / 3EM', class: 'singleSession' }
        ]
    },
    {
        id: 'battery',
        devices: [
            { id: 'byd', label: 'BYD Battery-Box', class: 'singleSession' },
            { id: 'pylontech', label: 'Pylontech (RS485)', class: 'serialGateway', note: 'viaGateway' },
            { id: 'varta', label: 'VARTA (Element, Pulse, One)', class: 'singleSession' }
        ]
    },
    {
        id: 'wallbox',
        devices: [
            { id: 'keba', label: 'KEBA P30 / P40', class: 'singleSession', note: 'singleClient' },
            { id: 'alfen', label: 'Alfen Eve', class: 'singleSessionSlow', note: 'singleClient' },
            { id: 'goE', label: 'go-e Charger', class: 'singleSession' },
            { id: 'wallboxPulsar', label: 'Wallbox Pulsar Plus', class: 'singleSession' },
            { id: 'mennekes', label: 'MENNEKES AMTRON', class: 'singleSession' },
            { id: 'webasto', label: 'Webasto Live / Next', class: 'singleSession' },
            { id: 'abl', label: 'ABL eMH', class: 'serialGateway', note: 'viaGateway' },
            { id: 'heidelberg', label: 'Heidelberg Energy Control', class: 'serialGateway', note: 'rtuOnly' }
        ]
    }
];

// Flat lookup by profile id.
const devicesById = new Map(
    deviceCategories.flatMap((category) =>
        category.devices.map((device) => [device.id, { ...device, category: category.id }])
    )
);

export const findDeviceProfile = (id) => {
    const device = devicesById.get(id);
    if (!device) return null;
    return { ...device, values: profileClasses[device.class] };
};

export const deviceProfileCount = devicesById.size;
