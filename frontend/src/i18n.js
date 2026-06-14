import { createI18n } from 'vue-i18n';

// German translations (default)
const de = {
  // Navigation
  nav: {
    dashboard: 'Dashboard',
    devices: 'Geräte',
    control: 'Steuerung',
    logs: 'Logs',
    config: 'Konfiguration',
    system: 'System',
    users: 'Benutzer',
    audit: 'Audit',
    settings: 'Einstellungen',
    logout: 'Abmelden',
    openNavigation: 'Navigation öffnen',
    closeNavigation: 'Schließen',
    lightMode: 'Heller Modus',
    darkMode: 'Dunkler Modus',
    theme: 'Theme'
  },

  // Dashboard
  dashboard: {
    title: 'Dashboard',
    addWidget: 'Widget hinzufügen',
    resetLayout: 'Layout zurücksetzen',
    loading: 'Lade Dashboard...',
    error: 'Fehler beim Laden',
    retry: 'Erneut versuchen',
    selectProxy: 'Proxy wählen',
    widgetRemove: 'Widget entfernen'
  },

  // Login
  login: {
    title: 'ModBridge Login',
    subtitle: 'Industrial Modbus Proxy Manager',
    username: 'Benutzername',
    password: 'Passwort',
    login: 'Anmelden',
    loginWithCredentials: 'Mit Zugangsdaten anmelden',
    enterPassword: 'Passwort eingeben um fortzufahren',
    usernamePlaceholder: 'benutzername',
    passwordPlaceholder: '••••••••',
    loginSuccess: 'Erfolgreich angemeldet',
    loginFailed: 'Anmeldung fehlgeschlagen',
    invalidCredentials: 'Ungültige Anmeldedaten',
    passwordRequirements: 'Passwort-Anforderungen',
    passwordMinLength: 'Mindestens 8 Zeichen lang',
    passwordComplexity: 'Mindestens 3 von: Großbuchstaben, Kleinbuchstaben, Zahlen, Sonderzeichen',
    passwordNotCommon: 'Nicht zu einfach oder häufig verwendet',
    currentPassword: 'Aktuelles Passwort',
    newPassword: 'Neues Passwort',
    changePassword: 'Passwort ändern',
    passwordChanged: 'Passwort erfolgreich geändert'
  },

  // System
  system: {
    title: 'Systeminformationen',
    system: 'System',
    memory: 'Speicher',
    proxies: 'Proxies',
    configuration: 'Konfiguration',
    security: 'Sicherheit',
    serverControl: 'Server-Steuerung',
    proxyControl: 'Proxy-Steuerung',
    portManagement: 'Port-Verwaltung',
    uptime: 'Betriebszeit',
    goroutines: 'Goroutines',
    memoryAlloc: 'Speicher (Alloc)',
    memorySys: 'Speicher (Sys)',
    memoryGc: 'Nächste GC',
    cpuCount: 'CPU-Kerne',
    totalProxies: 'Proxies gesamt',
    runningProxies: 'Proxies aktiv',
    stoppedProxies: 'Gestoppte Proxies',
    goVersion: 'Go-Version',
    os: 'Betriebssystem',
    arch: 'Architektur',
    refresh: 'Aktualisieren',
    restart: 'System neu starten',
    restartConfirm: 'Soll das System wirklich neu gestartet werden?',
    startAllProxies: 'Alle Proxies starten',
    stopAllProxies: 'Alle Proxies stoppen',
    restartAllProxies: 'Alle Proxies neu starten',
    downloadLogs: 'Logs herunterladen',
    checkPorts: 'Ports prüfen',
    total: 'Gesamt',
    free: 'Frei',
    inUse: 'Belegt',
    blockedPorts: 'Blockierte Ports',
    allPortsFree: 'Alle Ports sind frei',
    releasePort: 'Port freigeben',
    logLevel: 'Log-Level',
    debugMode: 'Debug-Modus',
    metrics: 'Metriken',
    tls: 'TLS',
    rateLimiting: 'Rate-Limiting',
    ipWhitelist: 'IP-Whitelist',
    ipBlacklist: 'IP-Blacklist',
    emailAlerts: 'E-Mail-Benachrichtigungen'
  },

  // Control
  control: {
    title: 'Steuerung',
    subtitle: 'Modbus TCP Proxies verwalten und überwachen',
    startAll: 'Alle starten',
    stopAll: 'Alle stoppen',
    start: 'Starten',
    stop: 'Stoppen',
    restart: 'Neustarten',
    pause: 'Pausieren',
    resume: 'Fortsetzen',
    edit: 'Bearbeiten',
    lock: 'Sperren',
    addProxy: 'Proxy hinzufügen',
    searchPlaceholder: 'Proxy suchen…',
    loading: 'Proxies werden geladen…',
    noProxies: 'Keine Proxies konfiguriert',
    noProxiesHint: 'Erstelle deinen ersten Proxy über den Button oben.',
    noResults: 'Keine Ergebnisse für „{query}"',
    total: 'Gesamt',
    running: 'Läuft',
    stopped: 'Gestoppt',
    error: 'Fehler',
    requests: 'Anfragen',
    reachable: 'Erreichbar',
    notReachable: 'Nicht erreichbar',
    deleteConfirm: 'Soll dieser Proxy wirklich gelöscht werden?',
    startAllConfirm: 'Sollen wirklich alle Proxies gestartet werden?',
    stopAllConfirm: 'Sollen wirklich alle Proxies gestoppt werden?',
    form: {
      name: 'Name',
      listenAddr: 'Listen-Adresse',
      targetAddr: 'Ziel-Adresse',
      description: 'Beschreibung',
      connectionTimeout: 'Verbindungs-Timeout (s)',
      readTimeout: 'Lese-Timeout (s)',
      maxRetries: 'Max. Wiederholungen',
      maxReadSize: 'Max. Lese-Größe (0=unbegrenzt)',
      enabled: 'Aktiviert',
      paused: 'Pausiert',
      tags: 'Tags'
    }
  },

  // Config
  config: {
    title: 'Konfiguration',
    save: 'Speichern',
    cancel: 'Abbrechen',
    delete: 'Löschen',
    edit: 'Bearbeiten',
    add: 'Hinzufügen',
    name: 'Name',
    enabled: 'Aktiviert',
    listenAddr: 'Listen-Adresse',
    targetAddr: 'Ziel-Adresse',
    description: 'Beschreibung',
    tags: 'Tags',
    proxy: 'Proxy',
    proxies: 'Proxies',
    logging: 'Logging',
    security: 'Sicherheit',
    email: 'E-Mail',
    backup: 'Backup',
    advanced: 'Erweitert',

    // Logging
    logLevel: 'Log-Level',
    logMaxSize: 'Max. Dateigröße (MB)',
    logMaxFiles: 'Max. Dateien',
    logMaxAgeDays: 'Max. Alter (Tage)',

    // Security
    enableTLS: 'SSL/TLS aktivieren',
    certFile: 'Zertifikatsdatei',
    keyFile: 'Schlüsseldatei',
    sessionTimeout: 'Session-Timeout (Stunden)',
    corsOrigins: 'Erlaubte Origins',
    corsMethods: 'Erlaubte Methoden',
    corsHeaders: 'Erlaubte Header',
    rateLimitEnabled: 'Rate-Limiting aktivieren',
    rateLimitRequests: 'Anfragen pro Minute',
    rateLimitBurst: 'Burst-Größe',
    ipWhitelistEnabled: 'IP-Whitelist aktivieren',
    ipWhitelist: 'IP-Whitelist',
    ipBlacklistEnabled: 'IP-Blacklist aktivieren',
    ipBlacklist: 'IP-Blacklist',

    // Email
    emailEnabled: 'E-Mail-Benachrichtigungen aktivieren',
    smtpServer: 'SMTP-Server',
    smtpPort: 'SMTP-Port',
    emailFrom: 'Absender',
    emailTo: 'Empfänger',
    emailUsername: 'Benutzername',
    emailPassword: 'Passwort',
    alertOnError: 'Bei Fehler benachrichtigen',
    alertOnWarning: 'Bei Warnung benachrichtigen',

    // Backup
    backupEnabled: 'Backups aktivieren',
    backupInterval: 'Backup-Intervall',
    backupRetention: 'Aufbewahrung (Anzahl)',
    backupPath: 'Backup-Pfad',
    backupDatabase: 'Datenbank sichern',
    backupConfig: 'Konfiguration sichern',

    // Advanced
    debugMode: 'Debug-Modus',
    maxConnections: 'Max. Verbindungen',
    metricsEnabled: 'Metriken aktivieren',
    metricsPort: 'Metriken-Port',
    exportConfig: 'Konfiguration exportieren',
    importConfig: 'Konfiguration importieren',
    restartSystem: 'System neu starten',
    changePassword: 'Passwort ändern'
  },

  // Common
  common: {
    save: 'Speichern',
    cancel: 'Abbrechen',
    delete: 'Löschen',
    edit: 'Bearbeiten',
    add: 'Hinzufügen',
    close: 'Schließen',
    confirm: 'Bestätigen',
    yes: 'Ja',
    no: 'Nein',
    loading: 'Laden...',
    saving: 'Speichern...',
    saved: 'Gespeichert',
    error: 'Fehler',
    success: 'Erfolg',
    warning: 'Warnung',
    info: 'Information',
    enabled: 'Aktiviert',
    disabled: 'Deaktiviert',
    language: 'Sprache',
    german: 'Deutsch',
    english: 'Englisch',
    lastRefreshed: 'Zuletzt aktualisiert',
    refreshNow: 'Jetzt aktualisieren',
    autoRefresh: 'Auto-Aktualisierung',
    live: 'Live',
    connected: 'Verbunden',
    disconnected: 'Getrennt',
    justNow: 'Gerade eben',
    secondsAgo: 'vor {n}s',
    minuteAgo: 'vor 1 Min.',
    minutesAgo: 'vor {n} Min.'
  },

  // Units
  units: {
    requests: 'Anfragen',
    bytes: 'Bytes',
    seconds: 'Sekunden',
    minutes: 'Minuten',
    hours: 'Stunden'
  }
};

// English translations
const en = {
  // Navigation
  nav: {
    dashboard: 'Dashboard',
    devices: 'Devices',
    control: 'Control',
    logs: 'Logs',
    config: 'Configuration',
    system: 'System',
    users: 'Users',
    audit: 'Audit',
    settings: 'Settings',
    logout: 'Logout',
    openNavigation: 'Open navigation',
    closeNavigation: 'Close',
    lightMode: 'Light mode',
    darkMode: 'Dark mode',
    theme: 'Theme'
  },

  // Dashboard
  dashboard: {
    title: 'Dashboard',
    addWidget: 'Add Widget',
    resetLayout: 'Reset Layout',
    loading: 'Loading Dashboard...',
    error: 'Error loading',
    retry: 'Retry',
    selectProxy: 'Select Proxy',
    widgetRemove: 'Remove Widget'
  },

  // Login
  login: {
    title: 'ModBridge Login',
    subtitle: 'Industrial Modbus Proxy Manager',
    username: 'Username',
    password: 'Password',
    login: 'Login',
    loginWithCredentials: 'Login with credentials',
    enterPassword: 'Enter password to continue',
    usernamePlaceholder: 'username',
    passwordPlaceholder: '••••••••',
    loginSuccess: 'Login successful',
    loginFailed: 'Login failed',
    invalidCredentials: 'Invalid credentials',
    passwordRequirements: 'Password Requirements',
    passwordMinLength: 'At least 8 characters',
    passwordComplexity: 'At least 3 of: Uppercase, Lowercase, Numbers, Special characters',
    passwordNotCommon: 'Not too simple or commonly used',
    currentPassword: 'Current Password',
    newPassword: 'New Password',
    changePassword: 'Change Password',
    passwordChanged: 'Password changed successfully'
  },

  // System
  system: {
    title: 'System Information',
    system: 'System',
    memory: 'Memory',
    proxies: 'Proxies',
    configuration: 'Configuration',
    security: 'Security',
    serverControl: 'Server Control',
    proxyControl: 'Proxy Control',
    portManagement: 'Port Management',
    uptime: 'Uptime',
    goroutines: 'Goroutines',
    memoryAlloc: 'Memory (Alloc)',
    memorySys: 'Memory (Sys)',
    memoryGc: 'Next GC',
    cpuCount: 'CPU Cores',
    totalProxies: 'Total Proxies',
    runningProxies: 'Running Proxies',
    stoppedProxies: 'Stopped Proxies',
    goVersion: 'Go Version',
    os: 'Operating System',
    arch: 'Architecture',
    refresh: 'Refresh',
    restart: 'Restart System',
    restartConfirm: 'Are you sure you want to restart the system?',
    startAllProxies: 'Start All Proxies',
    stopAllProxies: 'Stop All Proxies',
    restartAllProxies: 'Restart All Proxies',
    downloadLogs: 'Download Logs',
    checkPorts: 'Check Ports',
    total: 'Total',
    free: 'Free',
    inUse: 'In Use',
    blockedPorts: 'Blocked Ports',
    allPortsFree: 'All ports are free',
    releasePort: 'Release Port',
    logLevel: 'Log Level',
    debugMode: 'Debug Mode',
    metrics: 'Metrics',
    tls: 'TLS',
    rateLimiting: 'Rate Limiting',
    ipWhitelist: 'IP Whitelist',
    ipBlacklist: 'IP Blacklist',
    emailAlerts: 'Email Alerts'
  },

  // Control
  control: {
    title: 'Control',
    subtitle: 'Manage and monitor Modbus TCP proxies',
    startAll: 'Start All',
    stopAll: 'Stop All',
    start: 'Start',
    stop: 'Stop',
    restart: 'Restart',
    pause: 'Pause',
    resume: 'Resume',
    edit: 'Edit',
    lock: 'Lock',
    addProxy: 'Add Proxy',
    searchPlaceholder: 'Search proxy…',
    loading: 'Loading proxies…',
    noProxies: 'No proxies configured',
    noProxiesHint: 'Create your first proxy using the button above.',
    noResults: 'No results for "{query}"',
    total: 'Total',
    running: 'Running',
    stopped: 'Stopped',
    error: 'Error',
    requests: 'Requests',
    reachable: 'Reachable',
    notReachable: 'Not reachable',
    deleteConfirm: 'Are you sure you want to delete this proxy?',
    startAllConfirm: 'Are you sure you want to start all proxies?',
    stopAllConfirm: 'Are you sure you want to stop all proxies?',
    form: {
      name: 'Name',
      listenAddr: 'Listen Address',
      targetAddr: 'Target Address',
      description: 'Description',
      connectionTimeout: 'Connection Timeout (s)',
      readTimeout: 'Read Timeout (s)',
      maxRetries: 'Max Retries',
      maxReadSize: 'Max Read Size (0=unlimited)',
      enabled: 'Enabled',
      paused: 'Paused',
      tags: 'Tags'
    }
  },

  // Config
  config: {
    title: 'Configuration',
    save: 'Save',
    cancel: 'Cancel',
    delete: 'Delete',
    edit: 'Edit',
    add: 'Add',
    name: 'Name',
    enabled: 'Enabled',
    listenAddr: 'Listen Address',
    targetAddr: 'Target Address',
    description: 'Description',
    tags: 'Tags',
    proxy: 'Proxy',
    proxies: 'Proxies',
    logging: 'Logging',
    security: 'Security',
    email: 'Email',
    backup: 'Backup',
    advanced: 'Advanced',

    // Logging
    logLevel: 'Log Level',
    logMaxSize: 'Max File Size (MB)',
    logMaxFiles: 'Max Files',
    logMaxAgeDays: 'Max Age (Days)',

    // Security
    enableTLS: 'Enable SSL/TLS',
    certFile: 'Certificate File',
    keyFile: 'Key File',
    sessionTimeout: 'Session Timeout (Hours)',
    corsOrigins: 'Allowed Origins',
    corsMethods: 'Allowed Methods',
    corsHeaders: 'Allowed Headers',
    rateLimitEnabled: 'Enable Rate Limiting',
    rateLimitRequests: 'Requests per Minute',
    rateLimitBurst: 'Burst Size',
    ipWhitelistEnabled: 'Enable IP Whitelist',
    ipWhitelist: 'IP Whitelist',
    ipBlacklistEnabled: 'Enable IP Blacklist',
    ipBlacklist: 'IP Blacklist',

    // Email
    emailEnabled: 'Enable Email Alerts',
    smtpServer: 'SMTP Server',
    smtpPort: 'SMTP Port',
    emailFrom: 'From Email',
    emailTo: 'To Email',
    emailUsername: 'Username',
    emailPassword: 'Password',
    alertOnError: 'Alert on Error',
    alertOnWarning: 'Alert on Warning',

    // Backup
    backupEnabled: 'Enable Backups',
    backupInterval: 'Backup Interval',
    backupRetention: 'Retention (Count)',
    backupPath: 'Backup Path',
    backupDatabase: 'Backup Database',
    backupConfig: 'Backup Configuration',

    // Advanced
    debugMode: 'Debug Mode',
    maxConnections: 'Max Connections',
    metricsEnabled: 'Enable Metrics',
    metricsPort: 'Metrics Port',
    exportConfig: 'Export Configuration',
    importConfig: 'Import Configuration',
    restartSystem: 'Restart System',
    changePassword: 'Change Password'
  },

  // Common
  common: {
    save: 'Save',
    cancel: 'Cancel',
    delete: 'Delete',
    edit: 'Edit',
    add: 'Add',
    close: 'Close',
    confirm: 'Confirm',
    yes: 'Yes',
    no: 'No',
    loading: 'Loading...',
    saving: 'Saving...',
    saved: 'Saved',
    error: 'Error',
    success: 'Success',
    warning: 'Warning',
    info: 'Information',
    language: 'Language',
    german: 'German',
    english: 'English',
    lastRefreshed: 'Last refreshed',
    refreshNow: 'Refresh now',
    autoRefresh: 'Auto-refresh',
    live: 'Live',
    connected: 'Connected',
    disconnected: 'Disconnected',
    justNow: 'Just now',
    secondsAgo: '{n}s ago',
    minuteAgo: '1 min ago',
    minutesAgo: '{n} min ago'
  },

  // Units
  units: {
    requests: 'requests',
    bytes: 'bytes',
    seconds: 'seconds',
    minutes: 'minutes',
    hours: 'hours'
  }
};

// Get saved language preference or default to German
function getSavedLanguage() {
  const saved = localStorage.getItem('modbridge_language');
  if (saved && (saved === 'de' || saved === 'en')) {
    return saved;
  }
  return 'de'; // Default to German
}

// Save language preference
function saveLanguage(lang) {
  localStorage.setItem('modbridge_language', lang);
}

// Create i18n instance
const i18n = createI18n({
  legacy: false,
  locale: getSavedLanguage(),
  fallbackLocale: 'de',
  messages: {
    de,
    en
  }
});

export { i18n, saveLanguage };
