import { Component, Show, createEffect } from 'solid-js';
import { Routes, Route, Navigate, useNavigate } from '@solidjs/router';
import { ThemeProvider, createTheme, CssBaseline } from '@suid/material';
import { isAuthenticated, logout } from '@/stores/auth';
import Navigation from '@/components/Navigation';
import Login from '@/pages/Login';
import Register from '@/pages/Register';
import Dashboard from '@/pages/Dashboard';
import Clients from '@/pages/Clients';

const theme = createTheme({
  palette: {
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
  },
});

const App: Component = () => {
  const navigate = useNavigate();

  createEffect(() => {
    // Check authentication on app load
    const token = localStorage.getItem('auth_token');
    if (!token) {
      navigate('/login', { replace: true });
    }
  });

  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Show when={isAuthenticated()}>
        <Navigation onLogout={handleLogout} />
      </Show>
      <Routes>
        <Route path="/login" component={Login} />
        <Route path="/register" component={Register} />
        <Route path="/" element={<Navigate href="/dashboard" />} />
        <Route
          path="/dashboard"
          component={() => (
            <Show when={isAuthenticated()} fallback={<Navigate href="/login" />}>
              <Dashboard />
            </Show>
          )}
        />
        <Route
          path="/clients"
          component={() => (
            <Show when={isAuthenticated()} fallback={<Navigate href="/login" />}>
              <Clients />
            </Show>
          )}
        />
      </Routes>
    </ThemeProvider>
  );
};

export default App;