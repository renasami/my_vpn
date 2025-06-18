import { Component, createSignal, Show, onMount } from 'solid-js';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  CircularProgress,
  Alert,
  Paper,
  IconButton,
} from '@suid/material';
import {
  ContentCopy as CopyIcon,
  Download as DownloadIcon,
} from '@suid/icons-material';
import { clientService, type Client } from '@/services/clients';

interface QRCodeModalProps {
  open: boolean;
  client: Client;
  onClose: () => void;
}

const QRCodeModal: Component<QRCodeModalProps> = (props) => {
  const [format, setFormat] = createSignal<'png' | 'base64' | 'terminal'>('base64');
  const [qrData, setQrData] = createSignal('');
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');

  const fetchQRCode = async () => {
    setLoading(true);
    setError('');

    try {
      const response = await clientService.getQRCode(props.client.id, format());
      setQrData(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate QR code');
    } finally {
      setLoading(false);
    }
  };

  onMount(() => {
    if (props.open) {
      fetchQRCode();
    }
  });

  const handleFormatChange = (newFormat: 'png' | 'base64' | 'terminal') => {
    setFormat(newFormat);
    fetchQRCode();
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(qrData());
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const handleDownload = () => {
    const blob = new Blob([qrData()], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${props.client.name}-qr.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Dialog open={props.open} onClose={props.onClose} maxWidth="md" fullWidth>
      <DialogTitle>QR Code for {props.client.name}</DialogTitle>
      <DialogContent>
        <Box sx={{ mb: 2 }}>
          <Button
            variant={format() === 'base64' ? 'contained' : 'outlined'}
            onClick={() => handleFormatChange('base64')}
            sx={{ mr: 1 }}
          >
            Base64
          </Button>
          <Button
            variant={format() === 'png' ? 'contained' : 'outlined'}
            onClick={() => handleFormatChange('png')}
            sx={{ mr: 1 }}
          >
            PNG Image
          </Button>
          <Button
            variant={format() === 'terminal' ? 'contained' : 'outlined'}
            onClick={() => handleFormatChange('terminal')}
          >
            Terminal
          </Button>
        </Box>

        <Show when={loading()}>
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
            <CircularProgress />
          </Box>
        </Show>

        <Show when={error()}>
          <Alert severity="error" sx={{ mb: 2 }}>
            {error()}
          </Alert>
        </Show>

        <Show when={!loading() && !error() && qrData()}>
          <Paper sx={{ p: 2, mb: 2 }}>
            <Show when={format() === 'base64'}>
              <Typography variant="h6" gutterBottom>
                Base64 Encoded
              </Typography>
              <Box sx={{ wordBreak: 'break-all', fontFamily: 'monospace', fontSize: '0.875rem' }}>
                {qrData()}
              </Box>
            </Show>

            <Show when={format() === 'png'}>
              <Typography variant="h6" gutterBottom>
                QR Code Image
              </Typography>
              <img src={qrData()} alt="QR Code" style={{ 'max-width': '100%' }} />
            </Show>

            <Show when={format() === 'terminal'}>
              <Typography variant="h6" gutterBottom>
                Terminal Display
              </Typography>
              <Box sx={{ fontFamily: 'monospace', fontSize: '0.75rem', lineHeight: 1 }}>
                <pre>{qrData()}</pre>
              </Box>
            </Show>

            <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
              <IconButton onClick={handleCopy} aria-label="Copy to clipboard">
                <CopyIcon />
              </IconButton>
              <IconButton onClick={handleDownload} aria-label="Download">
                <DownloadIcon />
              </IconButton>
            </Box>
          </Paper>
        </Show>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default QRCodeModal;