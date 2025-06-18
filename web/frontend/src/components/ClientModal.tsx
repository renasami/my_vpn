import { Component, createSignal, Show } from 'solid-js';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  FormControlLabel,
  Switch,
  CircularProgress,
  Alert,
} from '@suid/material';
import { clientService, type Client } from '@/services/clients';

interface ClientModalProps {
  open: boolean;
  client?: Client;
  onClose: () => void;
  onSuccess: () => void;
}

const ClientModal: Component<ClientModalProps> = (props) => {
  const [name, setName] = createSignal(props.client?.name || '');
  const [enabled, setEnabled] = createSignal(props.client?.enabled ?? true);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');

  const isEdit = () => !!props.client;

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    if (!name().trim()) {
      setError('Name is required');
      return;
    }

    setLoading(true);
    setError('');

    try {
      if (isEdit()) {
        await clientService.update(props.client!.id, {
          name: name(),
          enabled: enabled(),
        });
      } else {
        await clientService.create({
          name: name(),
        });
      }
      props.onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Operation failed');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading()) {
      setName(props.client?.name || '');
      setEnabled(props.client?.enabled ?? true);
      setError('');
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={handleClose} maxWidth="sm" fullWidth>
      <form onSubmit={handleSubmit}>
        <DialogTitle>
          {isEdit() ? 'Edit Client' : 'Create New Client'}
        </DialogTitle>
        
        <DialogContent>
          <Show when={error()}>
            <Alert severity="error" sx={{ mb: 2 }}>
              {error()}
            </Alert>
          </Show>

          <TextField
            autoFocus
            margin="dense"
            label="Client Name"
            fullWidth
            variant="outlined"
            value={name()}
            onChange={(e) => setName(e.currentTarget.value)}
            disabled={loading()}
            required
            sx={{ mb: 2 }}
          />

          <Show when={isEdit()}>
            <FormControlLabel
              control={
                <Switch
                  checked={enabled()}
                  onChange={(e) => setEnabled(e.currentTarget.checked)}
                  disabled={loading()}
                />
              }
              label="Enabled"
            />
          </Show>

          <Show when={isEdit()}>
            <Alert severity="info" sx={{ mt: 2 }}>
              IP Address: {props.client?.ip_address}
            </Alert>
          </Show>
        </DialogContent>

        <DialogActions>
          <Button onClick={handleClose} disabled={loading()}>
            Cancel
          </Button>
          <Button 
            type="submit" 
            variant="contained" 
            disabled={loading() || !name().trim()}
          >
            <Show when={loading()} fallback={isEdit() ? 'Update' : 'Create'}>
              <CircularProgress size={20} color="inherit" />
            </Show>
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default ClientModal;