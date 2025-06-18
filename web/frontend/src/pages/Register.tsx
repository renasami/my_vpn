import { Component, createSignal, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import {
  Container,
  Paper,
  TextField,
  Button,
  Typography,
  Box,
  Alert,
  Link,
  CircularProgress,
} from '@suid/material';
import { PersonAdd as PersonAddIcon } from '@suid/icons-material';
import { authService } from '@/services/auth';
import { login } from '@/stores/auth';

const Register: Component = () => {
  const navigate = useNavigate();
  const [username, setUsername] = createSignal('');
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [error, setError] = createSignal('');
  const [loading, setLoading] = createSignal(false);

  const validateForm = () => {
    if (password() !== confirmPassword()) {
      setError('Passwords do not match');
      return false;
    }
    if (password().length < 6) {
      setError('Password must be at least 6 characters');
      return false;
    }
    return true;
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');
    
    if (!validateForm()) return;
    
    setLoading(true);

    try {
      const response = await authService.register({
        username: username(),
        email: email(),
        password: password(),
      });
      
      login(response);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container component="main" maxWidth="xs">
      <Box
        sx={{
          marginTop: 8,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
        }}
      >
        <Paper
          elevation={3}
          sx={{
            padding: 4,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <PersonAddIcon sx={{ fontSize: 48, color: 'primary.main', mb: 2 }} />
          <Typography component="h1" variant="h5">
            Create Account
          </Typography>
          
          <Show when={error()}>
            <Alert severity="error" sx={{ mt: 2, width: '100%' }}>
              {error()}
            </Alert>
          </Show>

          <Box component="form" onSubmit={handleSubmit} sx={{ mt: 1, width: '100%' }}>
            <TextField
              margin="normal"
              required
              fullWidth
              id="username"
              label="Username"
              name="username"
              autoComplete="username"
              autoFocus
              value={username()}
              onChange={(e) => setUsername(e.currentTarget.value)}
              disabled={loading()}
            />
            <TextField
              margin="normal"
              required
              fullWidth
              id="email"
              label="Email Address"
              name="email"
              type="email"
              autoComplete="email"
              value={email()}
              onChange={(e) => setEmail(e.currentTarget.value)}
              disabled={loading()}
            />
            <TextField
              margin="normal"
              required
              fullWidth
              name="password"
              label="Password"
              type="password"
              id="password"
              autoComplete="new-password"
              value={password()}
              onChange={(e) => setPassword(e.currentTarget.value)}
              disabled={loading()}
              helperText="At least 6 characters"
            />
            <TextField
              margin="normal"
              required
              fullWidth
              name="confirmPassword"
              label="Confirm Password"
              type="password"
              id="confirmPassword"
              value={confirmPassword()}
              onChange={(e) => setConfirmPassword(e.currentTarget.value)}
              disabled={loading()}
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              sx={{ mt: 3, mb: 2 }}
              disabled={loading() || !username() || !email() || !password() || !confirmPassword()}
            >
              <Show when={loading()} fallback="Sign Up">
                <CircularProgress size={24} color="inherit" />
              </Show>
            </Button>
            <Box textAlign="center">
              <Link
                component="button"
                variant="body2"
                onClick={(e) => {
                  e.preventDefault();
                  navigate('/login');
                }}
              >
                Already have an account? Sign In
              </Link>
            </Box>
          </Box>
        </Paper>
      </Box>
    </Container>
  );
};

export default Register;