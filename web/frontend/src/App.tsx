import { ThemeProvider } from '@/contexts/ThemeContext'
import { Dashboard } from '@/components/features/Dashboard/Dashboard'
import { Button } from '@/components/ui/Button'
import { useTheme } from '@/contexts/ThemeContext'
import { Sun, Moon, Monitor } from 'lucide-react'

function App() {
  return (
    <ThemeProvider>
      <div className="min-h-screen bg-background">
        <Header />
        <main>
          <Dashboard />
        </main>
      </div>
    </ThemeProvider>
  )
}

function Header() {
  const { theme, setTheme } = useTheme()

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-14 items-center">
        <div className="mr-4 flex">
          <a className="mr-6 flex items-center space-x-2" href="/">
            <span className="font-bold text-xl">Modbridge</span>
          </a>
          <nav className="flex items-center space-x-6 text-sm font-medium">
            <a href="#dashboard" className="transition-colors hover:text-foreground/80 text-foreground">
              Dashboard
            </a>
            <a href="#proxies" className="transition-colors hover:text-foreground/80 text-foreground/60">
              Proxies
            </a>
            <a href="#devices" className="transition-colors hover:text-foreground/80 text-foreground/60">
              Devices
            </a>
            <a href="#users" className="transition-colors hover:text-foreground/80 text-foreground/60">
              Users
            </a>
            <a href="#audit" className="transition-colors hover:text-foreground/80 text-foreground/60">
              Audit Log
            </a>
            <a href="#settings" className="transition-colors hover:text-foreground/80 text-foreground/60">
              Settings
            </a>
          </nav>
        </div>
        <div className="ml-auto flex items-center space-x-4">
          <ThemeToggle theme={theme} setTheme={setTheme} />
          <Button size="sm">Sign Out</Button>
        </div>
      </div>
    </header>
  )
}

function ThemeToggle({ theme, setTheme }: { theme: string, setTheme: (theme: 'light' | 'dark' | 'system') => void }) {
  return (
    <div className="flex items-center gap-1 border rounded-md p-1">
      <button
        onClick={() => setTheme('light')}
        className={`p-1.5 rounded ${theme === 'light' ? 'bg-accent' : ''}`}
        title="Light mode"
      >
        <Sun className="w-4 h-4" />
      </button>
      <button
        onClick={() => setTheme('dark')}
        className={`p-1.5 rounded ${theme === 'dark' ? 'bg-accent' : ''}`}
        title="Dark mode"
      >
        <Moon className="w-4 h-4" />
      </button>
      <button
        onClick={() => setTheme('system')}
        className={`p-1.5 rounded ${theme === 'system' ? 'bg-accent' : ''}`}
        title="System"
      >
        <Monitor className="w-4 h-4" />
      </button>
    </div>
  )
}

export default App
