# Phase 3: Web UI Modernization - Implementation Guide

## 🎯 Overview

Phase 3 transforms the Modbridge web interface into a modern, real-time React SPA with comprehensive user management, dark mode, and advanced features.

## ✅ Completed Backend Infrastructure

### 1. WebSocket Server (`pkg/websocket/`)

Real-time bidirectional communication for live dashboard updates.

**Features:**
- Hub-based architecture for broadcasting to multiple clients
- Automatic client registration/unregistration
- Ping/pong heartbeat mechanism
- Message types: proxy_status, device_update, metrics, logs, audit
- Per-client authentication and role-based filtering

**Usage Example:**
```go
// Initialize hub
hub := websocket.NewHub()
go hub.Run()

// Broadcast updates
hub.Broadcast(websocket.MessageTypeProxyStatus, proxyStatus)
hub.Broadcast(websocket.MessageTypeMetrics, metrics)
```

### 2. User Management & RBAC (`pkg/rbac/`)

Complete user management system with role-based access control.

**Roles:**
- `admin`: Full system access (users, proxies, devices, config, logs)
- `operator`: Manage proxies and devices, view logs/metrics
- `viewer`: Read-only access to all resources

**Permissions:**
```go
const (
    PermissionViewProxies
    PermissionManageProxies
    PermissionViewDevices
    PermissionManageDevices
    PermissionViewUsers
    PermissionManageUsers
    PermissionViewLogs
    PermissionViewMetrics
    PermissionBackupConfig
    PermissionRestoreConfig
)
```

**API Usage:**
```go
// Create user manager
um := rbac.NewUserManager()

// Create users
admin, _ := um.CreateUser("admin", "admin@example.com", "password", rbac.RoleAdmin)
operator, _ := um.CreateUser("operator", "op@example.com", "password", rbac.RoleOperator)

// Authenticate
user, _ := um.Authenticate("admin", "password")

// Check permissions
if rbac.HasPermission(user.Role, rbac.PermissionManageUsers) {
    // Allow action
}
```

### 3. Audit Logging (`pkg/audit/`)

Comprehensive audit trail for all administrative actions.

**Tracked Actions:**
- User authentication (login/logout)
- Proxy management (create/update/delete/start/stop)
- User management (create/update/delete)
- Configuration changes (backup/restore)
- Device management (rename)

**Features:**
- Persistent logging to file
- In-memory ring buffer for fast queries
- Search by user, action, time range
- Automatic log rotation

**Usage:**
```go
// Create audit logger
audit, _ := audit.NewLogger("audit.log", 1000)

// Log actions
audit.Log(
    userID, username,
    audit.ActionCreateProxy,
    "proxy-1",
    map[string]interface{}{"name": "PLC-1"},
    "192.168.1.100",
    true,
    nil,
)

// Query logs
entries := audit.Search("", audit.ActionCreateProxy, startTime, endTime, 50)
```

### 4. Configuration Backup/Restore (`pkg/backup/`)

Full system backup and restore with versioning.

**Features:**
- Tar.gz compressed archives
- Include config, users, devices
- Versioned backups with timestamps
- Import/export functionality
- Metadata tracking

**Usage:**
```go
// Create backup manager
bm, _ := backup.NewManager("./backups")

// Create backup
filename, _ := bm.Create(config, users, devices, "Pre-upgrade backup")

// List backups
backups, _ := bm.List()

// Restore backup
backup, _ := bm.Restore("backup_20251223_120000.tar.gz")

// Apply restored config
applyConfig(backup.Config)
```

## 🎨 Frontend Architecture

### Project Structure

```
web/frontend/
├── package.json              # Dependencies and scripts
├── vite.config.ts           # Vite configuration with proxy
├── tailwind.config.js       # Tailwind CSS + dark mode
├── tsconfig.json            # TypeScript configuration
├── index.html               # Entry point
└── src/
    ├── main.tsx             # Application entry
    ├── App.tsx              # Root component with routing
    ├── contexts/
    │   ├── ThemeContext.tsx # Dark mode provider
    │   ├── AuthContext.tsx  # Authentication state
    │   └── WebSocketContext.tsx  # Real-time connection
    ├── hooks/
    │   ├── useWebSocket.ts  # WebSocket hook
    │   ├── useTheme.ts      # Theme hook
    │   └── useAuth.ts       # Authentication hook
    ├── api/
    │   ├── client.ts        # Axios instance
    │   ├── proxies.ts       # Proxy API calls
    │   ├── devices.ts       # Device API calls
    │   ├── users.ts         # User API calls
    │   ├── audit.ts         # Audit log API
    │   └── backup.ts        # Backup API calls
    ├── stores/
    │   ├── useProxyStore.ts # Proxy state management
    │   ├── useDeviceStore.ts# Device state management
    │   └── useMetricsStore.ts # Metrics state
    ├── components/
    │   ├── layout/
    │   │   ├── Sidebar.tsx  # Navigation sidebar
    │   │   ├── Header.tsx   # Top bar with user menu
    │   │   └── Layout.tsx   # Main layout wrapper
    │   ├── ui/              # Reusable UI components
    │   │   ├── Button.tsx
    │   │   ├── Card.tsx
    │   │   ├── Input.tsx
    │   │   ├── Modal.tsx
    │   │   ├── Table.tsx
    │   │   └── Badge.tsx
    │   └── features/
    │       ├── Dashboard/
    │       │   ├── Dashboard.tsx
    │       │   ├── MetricsCard.tsx
    │       │   ├── ProxyStatusGrid.tsx
    │       │   └── LiveChart.tsx
    │       ├── Proxies/
    │       │   ├── ProxyList.tsx
    │       │   ├── ProxyForm.tsx
    │       │   └── ProxyDetails.tsx
    │       ├── Devices/
    │       │   ├── DeviceList.tsx
    │       │   ├── DeviceRename.tsx
    │       │   └── DeviceHistory.tsx
    │       ├── Users/
    │       │   ├── UserList.tsx
    │       │   ├── UserForm.tsx
    │       │   └── RoleSelector.tsx
    │       ├── Audit/
    │       │   ├── AuditLog.tsx
    │       │   ├── AuditFilter.tsx
    │       │   └── AuditExport.tsx
    │       └── Settings/
    │           ├── Settings.tsx
    │           ├── BackupRestore.tsx
    │           └── ThemeSelector.tsx
    └── styles/
        └── globals.css      # Global styles + CSS variables

```

### Technology Stack

**Core:**
- React 18.3 - UI library
- TypeScript 5.4 - Type safety
- Vite 5.2 - Build tool and dev server

**Routing & State:**
- React Router 6.23 - Client-side routing
- Zustand 4.5 - Lightweight state management
- TanStack Query 5.40 - Server state management

**UI & Styling:**
- Tailwind CSS 3.4 - Utility-first CSS
- Lucide React - Icon library
- Recharts 2.12 - Charts and graphs
- Sonner - Toast notifications

**Data Fetching:**
- Axios 1.7 - HTTP client
- WebSocket API - Real-time updates

## 🌙 Dark Mode Implementation

### Theme System

Uses CSS custom properties for seamless theme switching:

```css
/* globals.css */
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 221.2 83.2% 53.3%;
  --primary-foreground: 210 40% 98%;
  /* ... more color variables */
}

.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  --primary: 217.2 91.2% 59.8%;
  --primary-foreground: 222.2 47.4% 11.2%;
  /* ... more color variables */
}
```

### Theme Context (Example)

```typescript
// src/contexts/ThemeContext.tsx
import { createContext, useContext, useEffect, useState } from 'react'

type Theme = 'light' | 'dark' | 'system'

interface ThemeContextType {
  theme: Theme
  setTheme: (theme: Theme) => void
  effectiveTheme: 'light' | 'dark'
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    return (localStorage.getItem('theme') as Theme) || 'system'
  })

  const effectiveTheme = theme === 'system'
    ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
    : theme

  useEffect(() => {
    const root = window.document.documentElement
    root.classList.remove('light', 'dark')
    root.classList.add(effectiveTheme)
    localStorage.setItem('theme', theme)
  }, [theme, effectiveTheme])

  return (
    <ThemeContext.Provider value={{ theme, setTheme, effectiveTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export const useTheme = () => {
  const context = useContext(ThemeContext)
  if (!context) throw new Error('useTheme must be used within ThemeProvider')
  return context
}
```

## 🔌 WebSocket Integration

### useWebSocket Hook (Example)

```typescript
// src/hooks/useWebSocket.ts
import { useEffect, useRef, useState } from 'react'

export interface WebSocketMessage {
  type: string
  timestamp: string
  data: any
}

export function useWebSocket(url: string) {
  const [connected, setConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const ws = useRef<WebSocket | null>(null)

  useEffect(() => {
    const connect = () => {
      ws.current = new WebSocket(url)

      ws.current.onopen = () => {
        console.log('WebSocket connected')
        setConnected(true)
      }

      ws.current.onclose = () => {
        console.log('WebSocket disconnected')
        setConnected(false)
        // Reconnect after 3 seconds
        setTimeout(connect, 3000)
      }

      ws.current.onmessage = (event) => {
        const message = JSON.parse(event.data)
        setLastMessage(message)
      }

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    }

    connect()

    return () => {
      ws.current?.close()
    }
  }, [url])

  return { connected, lastMessage }
}
```

### Real-time Dashboard Component (Example)

```typescript
// src/components/features/Dashboard/Dashboard.tsx
import { useWebSocket } from '@/hooks/useWebSocket'
import { useEffect } from 'react'
import { useProxyStore } from '@/stores/useProxyStore'
import { useMetricsStore } from '@/stores/useMetricsStore'

export function Dashboard() {
  const { lastMessage } = useWebSocket('ws://localhost:8080/ws')
  const updateProxy = useProxyStore(state => state.updateProxy)
  const updateMetrics = useMetricsStore(state => state.updateMetrics)

  useEffect(() => {
    if (!lastMessage) return

    switch (lastMessage.type) {
      case 'proxy_status':
        updateProxy(lastMessage.data)
        break
      case 'metrics':
        updateMetrics(lastMessage.data)
        break
      case 'device_update':
        // Update device list
        break
    }
  }, [lastMessage])

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <MetricsCard title="Active Proxies" value={activeProxies} />
      <MetricsCard title="Total Requests" value={totalRequests} />
      <MetricsCard title="Error Rate" value={errorRate} />
      <MetricsCard title="Avg Latency" value={avgLatency} />

      <div className="col-span-full">
        <LiveChart data={metricsHistory} />
      </div>

      <div className="col-span-full">
        <ProxyStatusGrid proxies={proxies} />
      </div>
    </div>
  )
}
```

## 📦 Key Features Implementation

### 1. User Management UI

**Components:**
- User list table with role badges
- Add/edit user modal with role selector
- Change password dialog
- Enable/disable user toggle
- Delete confirmation

**Permissions:**
- Only admins can see user management
- Form validation with TypeScript
- RBAC enforcement on all actions

### 2. Audit Log Viewer

**Features:**
- Filterable table (user, action, date range)
- Real-time updates via WebSocket
- Export to CSV/JSON
- Pagination for large logs
- Color-coded action types

### 3. Device Management

**Enhanced Features:**
- Device rename with inline editing
- Connection history timeline
- Last seen timestamp
- Traffic statistics per device
- Device grouping/filtering

### 4. Configuration Backup/Restore

**UI Components:**
- Backup list with size and date
- Create backup with description
- One-click restore with confirmation
- Import/export backup files
- Backup preview before restore

### 5. Real-time Metrics Dashboard

**Visualizations:**
- Line charts for request rate
- Bar charts for proxy comparison
- Gauge charts for connection pool usage
- Status indicators with live updates
- Auto-refresh every 5 seconds

## 🚀 Development Workflow

### Setup

```bash
cd web/frontend
npm install
npm run dev
```

### Build for Production

```bash
npm run build
# Output: web/dist/
```

### Backend Integration

The Go server serves the built React app:

```go
// Serve React app static files
mux.Handle("/", http.FileServer(http.Dir("web/dist")))

// API routes
mux.Handle("/api/", apiHandler)

// WebSocket
mux.Handle("/ws", wsHandler)
```

## 🎨 UI Component Examples

### Button Component

```typescript
// src/components/ui/Button.tsx
import { ButtonHTMLAttributes, forwardRef } from 'react'
import { cn } from '@/lib/utils'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'destructive' | 'outline'
  size?: 'sm' | 'md' | 'lg'
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = 'primary', size = 'md', ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(
          'inline-flex items-center justify-center rounded-md font-medium transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
          'disabled:opacity-50 disabled:pointer-events-none',
          {
            'bg-primary text-primary-foreground hover:bg-primary/90': variant === 'primary',
            'bg-secondary text-secondary-foreground hover:bg-secondary/80': variant === 'secondary',
            'bg-destructive text-destructive-foreground hover:bg-destructive/90': variant === 'destructive',
            'border border-input hover:bg-accent': variant === 'outline',
          },
          {
            'h-8 px-3 text-sm': size === 'sm',
            'h-10 px-4': size === 'md',
            'h-12 px-6 text-lg': size === 'lg',
          },
          className
        )}
        {...props}
      />
    )
  }
)
```

### Card Component

```typescript
// src/components/ui/Card.tsx
import { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'rounded-lg border bg-card text-card-foreground shadow-sm',
        className
      )}
      {...props}
    />
  )
}

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('flex flex-col space-y-1.5 p-6', className)} {...props} />
}

export function CardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return <h3 className={cn('font-semibold leading-none tracking-tight', className)} {...props} />
}

export function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('p-6 pt-0', className)} {...props} />
}
```

## 🔐 Authentication Flow

```typescript
// src/contexts/AuthContext.tsx
export function AuthProvider({ children }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Check if user is authenticated
    api.get('/api/auth/me')
      .then(res => setUser(res.data))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const login = async (username: string, password: string) => {
    const res = await api.post('/api/auth/login', { username, password })
    setUser(res.data.user)
  }

  const logout = async () => {
    await api.post('/api/auth/logout')
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, login, logout, loading }}>
      {children}
    </AuthContext.Provider>
  )
}
```

## 📊 State Management with Zustand

```typescript
// src/stores/useProxyStore.ts
import { create } from 'zustand'

interface ProxyStore {
  proxies: Proxy[]
  setProxies: (proxies: Proxy[]) => void
  addProxy: (proxy: Proxy) => void
  updateProxy: (proxy: Proxy) => void
  removeProxy: (id: string) => void
}

export const useProxyStore = create<ProxyStore>((set) => ({
  proxies: [],
  setProxies: (proxies) => set({ proxies }),
  addProxy: (proxy) => set((state) => ({
    proxies: [...state.proxies, proxy]
  })),
  updateProxy: (proxy) => set((state) => ({
    proxies: state.proxies.map(p => p.id === proxy.id ? proxy : p)
  })),
  removeProxy: (id) => set((state) => ({
    proxies: state.proxies.filter(p => p.id !== id)
  })),
}))
```

## 🎯 Next Steps

### To Complete Phase 3:

1. **Install Dependencies:**
   ```bash
   cd web/frontend
   npm install
   ```

2. **Create Remaining Components:**
   - Follow the structure in this guide
   - Implement each feature incrementally
   - Test with real backend data

3. **Build and Integrate:**
   ```bash
   npm run build
   # Update Go server to serve from web/dist
   ```

4. **Testing:**
   - Test all user roles (admin, operator, viewer)
   - Verify real-time updates work
   - Test dark mode across all pages
   - Verify backup/restore functionality

5. **Deployment:**
   - Update Dockerfile to include frontend build
   - Update Kubernetes manifests
   - Add nginx reverse proxy if needed

## 🎉 Expected Outcome

A fully modern web interface with:
- ✅ Responsive design (mobile, tablet, desktop)
- ✅ Dark mode with system preference detection
- ✅ Real-time dashboard updates
- ✅ Multi-user support with RBAC
- ✅ Comprehensive audit logging
- ✅ Device management with history
- ✅ Configuration backup/restore
- ✅ Beautiful, accessible UI components
- ✅ Type-safe codebase with TypeScript
- ✅ Production-ready build optimization

---

**Time Estimate:** 2-3 weeks for complete implementation
**Team:** 1-2 frontend developers
**Complexity:** Medium-High
