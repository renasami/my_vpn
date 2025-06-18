import { Component, createSignal, Show, onMount } from 'solid-js';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  ToggleButton,
  ToggleButtonGroup,
  CircularProgress,
  Alert,
  Paper,
  IconButton,
  Tooltip,
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

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(qrData());
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
    }
  };

  const downloadQRCode = () => {
    if (format() === 'png') {
      // For PNG format, qrData should be a blob URL or base64
      const link = document.createElement('a');
      link.href = qrData().startsWith('data:') ? qrData() : `data:image/png;base64,${qrData()}`;
      link.download = `${props.client.name}-qr.png`;
      link.click();
    } else {
      // For base64 and terminal formats, save as text
      const blob = new Blob([qrData()], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${props.client.name}-qr.${format() === 'terminal' ? 'txt' : 'b64'}`;
      link.click();
      URL.revokeObjectURL(url);
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        QR Code for {props.client.name}
      </DialogTitle>
      
      <DialogContent>
        <Box sx={{ mb: 3 }}>
          <ToggleButtonGroup
            value={format()}
            exclusive
            onChange={(_, value) => value && handleFormatChange(value)}
            sx={{ mb: 2 }}
          >
            <ToggleButton value="base64">Base64</ToggleButton>
            <ToggleButton value="png">PNG Image</ToggleButton>
            <ToggleButton value="terminal">Terminal</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        <Show when={error()}>
          <Alert severity="error" sx={{ mb: 2 }}>
            {error()}
          </Alert>
        </Show>

        <Show when={loading()}>
          <Box display="flex" justifyContent="center" p={4}>
            <CircularProgress />
          </Box>
        </Show>

        <Show when={!loading() && qrData()}>
          <Paper variant="outlined" sx={{ p: 2, position: 'relative' }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
              <Typography variant="subtitle2">
                {format() === 'png' ? 'QR Code Image' : 
                 format() === 'base64' ? 'Base64 Encoded' : 
                 'Terminal Display'}
              </Typography>
              <Box>
                <Tooltip title="Copy to clipboard">
                  <IconButton size="small" onClick={copyToClipboard}>
                    <CopyIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Download">
                  <IconButton size="small" onClick={downloadQRCode}>
                    <DownloadIcon />
                  </IconButton>
                </Tooltip>
              </Box>
            </Box>

            <Show when={format() === 'png'}>
              <Box display="flex" justifyContent="center" p={2}>
                <img 
                  src={qrData().startsWith('data:') ? qrData() : `data:image/png;base64,${qrData()}`}
                  alt="QR Code"
                  style={{ "max-width": "300px", "max-height": "300px" }}
                />
              </Box>
            </Show>

            <Show when={format() === 'terminal'}>
              <Box
                component="pre"
                sx={{
                  fontFamily: 'monospace',
                  fontSize: '0.5rem',
                  lineHeight: '0.5rem',
                  overflow: 'auto',
                  backgroundColor: 'black',
                  color: 'white',
                  p: 1,
                  borderRadius: 1,
                  maxHeight: '400px',
                }}
              >
                {qrData()}
              </Box>
            </Show>

            <Show when={format() === 'base64'}>
              <Box
                component="pre"
                sx={{
                  fontFamily: 'monospace',
                  fontSize: '0.75rem',
                  overflow: 'auto',
                  backgroundColor: '#f5f5f5',
                  p: 1,
                  borderRadius: 1,
                  maxHeight: '200px',
                  wordBreak: 'break-all',
                }}
              >
                {qrData()}
              </Box>
            </Show>
          </Paper>
        </Show>
      </DialogContent>

      <DialogActions>
        <Button onClick={props.onClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default QRCodeModal;