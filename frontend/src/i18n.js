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
    logout: 'Abmelden'
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
    username: 'Benutzername',
    password: 'Passwort',
    login: 'Anmelden',
    loginSuccess: 'Erfolgreich angemeldet',
    loginFailed: 'Anmeldung fehlgeschlagen',
    passwordRequirements: 'Passwort-Anforderungen',
    passwordMinLength: 'Mindestens 8 Zeichen lang',
    passwordComplexity: 'Mindestens 3 von: Großbuchstaben, Kleinbuchstaben, Zahlen, Sonderzeichen',
    passwordNotCommon: 'Nicht zu einfach oder häufig verwendet',
    currentPassword: 'Aktuelles Passwort',
    newPassword: 'Neues Passwort',
    changePassword: 'Passwort ändern',
    passwordChanged: 'Passwort erfolgreich geändert'
  },

  // Control
  control: {
    title: 'Steuerung',
    startAll: 'Alle starten',
    stopAll: 'Alle stoppen',
    start: 'Starten',
    stop: 'Stoppen',
    running: 'Läuft',
    stopped: 'Gestoppt',
    requests: 'Anfragen'
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
    language: 'Sprache',
    german: 'Deutsch',
    english: 'Englisch'
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
    logout: 'Logout'
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
    username: 'Username',
    password: 'Password',
    login: 'Login',
    loginSuccess: 'Login successful',
    loginFailed: 'Login failed',
    passwordRequirements: 'Password Requirements',
    passwordMinLength: 'At least 8 characters',
    passwordComplexity: 'At least 3 of: Uppercase, Lowercase, Numbers, Special characters',
    passwordNotCommon: 'Not too simple or commonly used',
    currentPassword: 'Current Password',
    newPassword: 'New Password',
    changePassword: 'Change Password',
    passwordChanged: 'Password changed successfully'
  },

  // Control
  control: {
    title: 'Control',
    startAll: 'Start All',
    stopAll: 'Stop All',
    start: 'Start',
    stop: 'Stop',
    running: 'Running',
    stopped: 'Stopped',
    requests: 'requests'
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
    english: 'English'
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
