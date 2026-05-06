/**
 * Application constants for ModBridge
 * Centralized configuration values to avoid magic numbers
 */

// API Timeouts (in milliseconds)
export const API_TIMEOUT = {
  DEFAULT: 10000,        // Default API request timeout
  SHORT: 5000,           // Short timeout for quick requests
  LONG: 30000            // Long timeout for heavy operations
}

// Dashboard Configuration
export const DASHBOARD_CONFIG = {
  AUTO_REFRESH_INTERVAL: 10000,   // Auto-refresh interval (10 seconds)
  MIN_WIDGET_WIDTH: 80,            // Minimum widget width in pixels
  MIN_WIDGET_HEIGHT: 3,            // Minimum widget height in grid units
  COLUMN_COUNT: 12,                // Number of columns in grid
  FLOATING: true,                  // Enable floating widgets
  DRAGGABLE: { handle: '.drag-handle' },  // Drag handle selector
  RESIZABLE: { handles: 'e, se, s, sw, w' }  // Resizable handles
}

// GridStack Configuration
export const GRID_CONFIG = {
  ANIMATE: true,                   // Enable animations
  CELL_HEIGHT: 'auto',             // Auto cell height
  MARGIN: 5,                       // Margin between widgets
  MIN_ROW: 1,                      // Minimum row
  MAX_ROW: 100,                    // Maximum row
  DISABLE_DRAG: false,             // Enable dragging
  DISABLE_RESIZE: false            // Enable resizing
}

// EventSource Configuration
export const EVENT_SOURCE_CONFIG = {
  MAX_RECONNECT_ATTEMPTS: 10000,   // Retry indefinitely to recover from backend restarts
  INITIAL_DELAY: 1000,             // Initial delay before reconnect
  MAX_DELAY: 10000                 // Maximum delay between reconnects (10s)
}

// Breakpoints (in pixels)
export const BREAKPOINTS = {
  MOBILE: 640,        // Mobile screens
  TABLET: 768,        // Tablet screens
  DESKTOP: 1024,      // Desktop screens
  LARGE: 1280,        // Large screens
  XLARGE: 1536        // Extra large screens
}

// Animation Durations (in milliseconds)
export const ANIMATION_DURATION = {
  FAST: 150,          // Fast animation
  NORMAL: 300,        // Normal animation
  SLOW: 500           // Slow animation
}
