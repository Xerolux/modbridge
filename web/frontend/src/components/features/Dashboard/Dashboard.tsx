import { useEffect, useState } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Activity, Server, AlertCircle, Zap } from 'lucide-react'

interface DashboardStats {
  activeProxies: number
  totalRequests: number
  errorRate: number
  avgLatency: number
}

export function Dashboard() {
  const { connected, lastMessage } = useWebSocket('ws://localhost:8080/ws')
  const [stats, setStats] = useState<DashboardStats>({
    activeProxies: 0,
    totalRequests: 0,
    errorRate: 0,
    avgLatency: 0,
  })

  useEffect(() => {
    if (!lastMessage) return

    // Update stats based on WebSocket messages
    if (lastMessage.type === 'metrics') {
      setStats(lastMessage.data)
    }
  }, [lastMessage])

  return (
    <div className="p-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">
            Real-time overview of your Modbus proxies
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-sm text-muted-foreground">
            {connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricsCard
          title="Active Proxies"
          value={stats.activeProxies}
          icon={<Server className="w-4 h-4" />}
          trend="+2 from last hour"
        />
        <MetricsCard
          title="Total Requests"
          value={stats.totalRequests.toLocaleString()}
          icon={<Activity className="w-4 h-4" />}
          trend="+12% from last hour"
        />
        <MetricsCard
          title="Error Rate"
          value={`${stats.errorRate.toFixed(2)}%`}
          icon={<AlertCircle className="w-4 h-4" />}
          trend="Normal"
          variant={stats.errorRate > 5 ? 'destructive' : 'default'}
        />
        <MetricsCard
          title="Avg Latency"
          value={`${stats.avgLatency.toFixed(1)}ms`}
          icon={<Zap className="w-4 h-4" />}
          trend="-5ms from last hour"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Latest proxy connections and requests</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Real-time activity feed will appear here
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>System Health</CardTitle>
            <CardDescription>Overall system status</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <HealthItem label="Connection Pool" status="healthy" />
              <HealthItem label="Metrics Collection" status="healthy" />
              <HealthItem label="WebSocket" status={connected ? 'healthy' : 'warning'} />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

interface MetricsCardProps {
  title: string
  value: string | number
  icon: React.ReactNode
  trend?: string
  variant?: 'default' | 'destructive'
}

function MetricsCard({ title, value, icon, trend, variant = 'default' }: MetricsCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        <div className={`text-2xl font-bold ${variant === 'destructive' ? 'text-destructive' : ''}`}>
          {value}
        </div>
        {trend && (
          <p className="text-xs text-muted-foreground mt-1">{trend}</p>
        )}
      </CardContent>
    </Card>
  )
}

interface HealthItemProps {
  label: string
  status: 'healthy' | 'warning' | 'error'
}

function HealthItem({ label, status }: HealthItemProps) {
  const colors = {
    healthy: 'bg-green-500',
    warning: 'bg-yellow-500',
    error: 'bg-red-500',
  }

  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm">{label}</span>
      <div className="flex items-center gap-2">
        <div className={`w-2 h-2 rounded-full ${colors[status]}`} />
        <span className="text-sm capitalize text-muted-foreground">{status}</span>
      </div>
    </div>
  )
}
