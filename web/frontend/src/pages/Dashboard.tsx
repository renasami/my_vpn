import { Component, createSignal, onMount, Show } from 'solid-js';
import {
  Container,
  Grid,
  Paper,
  Typography,
  Box,
  CircularProgress,
  Alert,
} from '@suid/material';
import {
  People as PeopleIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  NetworkCheck as NetworkCheckIcon,
} from '@suid/icons-material';
import { api } from '@/services/api';

interface ServerMetrics {
  server_status: string;
  connection_stats: {
    total_clients: number;
    active_clients: number;
  };
  network_stats: {
    bytes_received: number;
    bytes_sent: number;
    ip_pool_utilization: number;
  };
  alerts: Array<{
    id: string;
    title: string;
    severity: string;
  }>;
}

const Dashboard: Component = () => {
  const [metrics, setMetrics] = createSignal<ServerMetrics | null>(null);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal('');

  const fetchMetrics = async () => {
    try {
      const data = await api.get<ServerMetrics>('/monitoring/metrics');
      setMetrics(data);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
    } finally {
      setLoading(false);
    }
  };

  onMount(() => {
    fetchMetrics();
    // Refresh every 30 seconds
    const interval = setInterval(fetchMetrics, 30000);
    return () => clearInterval(interval);
  });

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircleIcon color="success" sx={{ fontSize: 40 }} />;
      case 'degraded':
        return <ErrorIcon color="warning" sx={{ fontSize: 40 }} />;
      default:
        return <ErrorIcon color="error" sx={{ fontSize: 40 }} />;
    }
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
      <Typography variant="h4" gutterBottom>
        Dashboard
      </Typography>

      <Show when={error()}>
        <Alert severity="error" sx={{ mb: 2 }}>
          {error()}
        </Alert>
      </Show>

      <Show when={loading()} fallback={
        <Show when={metrics()}>
          {(data) => (
            <Grid container spacing={3}>
              {/* Server Status */}
              <Grid item xs={12} sm={6} md={3}>
                <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                  {getStatusIcon(data().server_status)}
                  <Typography component="h2" variant="h6" gutterBottom>
                    Server Status
                  </Typography>
                  <Typography variant="h5" color="text.secondary">
                    {data().server_status}
                  </Typography>
                </Paper>
              </Grid>

              {/* Active Clients */}
              <Grid item xs={12} sm={6} md={3}>
                <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                  <PeopleIcon sx={{ fontSize: 40, color: 'primary.main' }} />
                  <Typography component="h2" variant="h6" gutterBottom>
                    Active Clients
                  </Typography>
                  <Typography variant="h5" color="text.secondary">
                    {data().connection_stats.active_clients} / {data().connection_stats.total_clients}
                  </Typography>
                </Paper>
              </Grid>

              {/* Data Transfer */}
              <Grid item xs={12} sm={6} md={3}>
                <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                  <NetworkCheckIcon sx={{ fontSize: 40, color: 'info.main' }} />
                  <Typography component="h2" variant="h6" gutterBottom>
                    Data Transfer
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    ↓ {formatBytes(data().network_stats.bytes_received)}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    ↑ {formatBytes(data().network_stats.bytes_sent)}
                  </Typography>
                </Paper>
              </Grid>

              {/* IP Pool Usage */}
              <Grid item xs={12} sm={6} md={3}>
                <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                  <Box sx={{ position: 'relative', display: 'inline-flex' }}>
                    <CircularProgress
                      variant="determinate"
                      value={data().network_stats.ip_pool_utilization}
                      size={60}
                      thickness={4}
                    />
                    <Box
                      sx={{
                        top: 0,
                        left: 0,
                        bottom: 0,
                        right: 0,
                        position: 'absolute',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <Typography variant="caption" component="div" color="text.secondary">
                        {`${Math.round(data().network_stats.ip_pool_utilization)}%`}
                      </Typography>
                    </Box>
                  </Box>
                  <Typography component="h2" variant="h6" gutterBottom sx={{ mt: 1 }}>
                    IP Pool Usage
                  </Typography>
                </Paper>
              </Grid>

              {/* Alerts */}
              <Show when={data().alerts.length > 0}>
                <Grid item xs={12}>
                  <Paper sx={{ p: 2 }}>
                    <Typography variant="h6" gutterBottom>
                      Active Alerts
                    </Typography>
                    {data().alerts.map((alert) => (
                      <Alert 
                        severity={
                          alert.severity === 'critical' ? 'error' : 
                          alert.severity === 'high' ? 'warning' : 
                          'info'
                        }
                        sx={{ mt: 1 }}
                      >
                        {alert.title}
                      </Alert>
                    ))}
                  </Paper>
                </Grid>
              </Show>
            </Grid>
          )}
        </Show>
      }>
        <Box display="flex" justifyContent="center" mt={4}>
          <CircularProgress />
        </Box>
      </Show>
    </Container>
  );
};

export default Dashboard;