# Modbridge Frontend

Modern React SPA for Modbridge - High-Performance Modbus TCP Proxy

## 🚀 Quick Start

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## 📁 Project Structure

```
src/
├── components/      # React components
│   ├── layout/     # Layout components (Sidebar, Header)
│   ├── ui/         # Reusable UI components
│   └── features/   # Feature-specific components
├── contexts/       # React contexts (Theme, Auth, WebSocket)
├── hooks/          # Custom React hooks
├── api/            # API client and endpoints
├── stores/         # Zustand state stores
├── styles/         # Global styles
└── lib/            # Utility functions
```

## 🎨 Features

- **Dark Mode**: Automatic theme switching with system preference detection
- **Real-time Updates**: WebSocket integration for live dashboard
- **User Management**: RBAC with admin, operator, and viewer roles
- **Audit Logging**: Complete activity tracking and search
- **Device Management**: Enhanced device tracking and renaming
- **Backup/Restore**: Configuration management
- **Responsive Design**: Mobile, tablet, and desktop support

## 🛠️ Tech Stack

- **React 18.3** - UI framework
- **TypeScript 5.4** - Type safety
- **Vite 5.2** - Build tool
- **Tailwind CSS 3.4** - Styling
- **Zustand 4.5** - State management
- **TanStack Query 5.40** - Server state
- **React Router 6.23** - Routing
- **Axios 1.7** - HTTP client
- **Recharts 2.12** - Charts
- **Lucide React** - Icons

## 📝 Development Guidelines

### Code Style

- Use TypeScript for all new code
- Follow React Hooks best practices
- Use Tailwind CSS for styling
- Extract reusable components to `components/ui/`
- Keep components small and focused

### State Management

- Use Zustand for global state
- Use TanStack Query for server state
- Use React Context for cross-cutting concerns (theme, auth)
- Prefer local state when possible

### API Integration

```typescript
import { api } from '@/api/client'

// GET request
const proxies = await api.get('/api/proxies')

// POST request
await api.post('/api/proxies', proxyData)
```

### WebSocket Usage

```typescript
import { useWebSocket } from '@/hooks/useWebSocket'

function Component() {
  const { connected, lastMessage } = useWebSocket('ws://localhost:8080/ws')

  useEffect(() => {
    if (lastMessage) {
      // Handle message
    }
  }, [lastMessage])
}
```

### Dark Mode

```typescript
import { useTheme } from '@/contexts/ThemeContext'

function Component() {
  const { theme, setTheme, effectiveTheme } = useTheme()

  return (
    <button onClick={() => setTheme('dark')}>
      Switch to Dark Mode
    </button>
  )
}
```

## 🧪 Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run linter
npm run lint
```

## 📦 Building

```bash
# Production build
npm run build

# Output directory: ../dist/
```

The built files will be served by the Go backend.

## 🔐 Authentication

The frontend uses session-based authentication with cookies:

```typescript
import { useAuth } from '@/contexts/AuthContext'

function LoginForm() {
  const { login } = useAuth()

  const handleSubmit = async (e) => {
    await login(username, password)
    // Redirect to dashboard
  }
}
```

## 📊 Real-time Dashboard

The dashboard receives updates via WebSocket:

- **proxy_status**: Proxy state changes
- **device_update**: Device list changes
- **metrics**: Performance metrics
- **log**: New log entries
- **audit**: Audit log entries

## 🎨 Theming

Themes are defined using CSS custom properties in `src/styles/globals.css`:

```css
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  /* ... */
}

.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  /* ... */
}
```

## 🔧 Environment Variables

Create `.env` file:

```env
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

## 📚 Resources

- [React Documentation](https://react.dev/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Tailwind CSS](https://tailwindcss.com/)
- [Vite Guide](https://vitejs.dev/guide/)
- [Zustand](https://zustand-demo.pmnd.rs/)
- [TanStack Query](https://tanstack.com/query/latest)

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Run linter and tests
4. Submit a pull request

## 📄 License

MIT License - see main project LICENSE file
