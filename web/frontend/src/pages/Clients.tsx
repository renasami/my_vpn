import { Component, Show, createSignal } from 'solid-js';
import {
  Container,
  Typography,
  Box,
  Button,
  CircularProgress,
  Alert,
} from '@suid/material';
import { Add as AddIcon } from '@suid/icons-material';
import ClientList from '@/components/ClientList';
import ClientModal from '@/components/ClientModal';
import QRCodeModal from '@/components/QRCodeModal';
import { clientsState, refreshClients } from '@/stores/clients';
import { clientService, type Client } from '@/services/clients';

const Clients: Component = () => {
  const [createModalOpen, setCreateModalOpen] = createSignal(false);
  const [editModalOpen, setEditModalOpen] = createSignal(false);
  const [qrModalOpen, setQRModalOpen] = createSignal(false);
  const [selectedClient, setSelectedClient] = createSignal<Client | null>(null);
  const [actionLoading, setActionLoading] = createSignal(false);
  const [actionError, setActionError] = createSignal('');

  const handleCreate = () => {
    setSelectedClient(null);
    setCreateModalOpen(true);
  };

  const handleEdit = (client: Client) => {
    setSelectedClient(client);
    setEditModalOpen(true);
  };

  const handleQRCode = (client: Client) => {
    setSelectedClient(client);
    setQRModalOpen(true);
  };

  const handleDelete = async (client: Client) => {
    if (!confirm(`Are you sure you want to delete client "${client.name}"?`)) {
      return;
    }

    setActionLoading(true);
    setActionError('');

    try {
      await clientService.delete(client.id);
      refreshClients();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete client');
    } finally {
      setActionLoading(false);
    }
  };

  const handleToggle = async (client: Client) => {
    setActionLoading(true);
    setActionError('');

    try {
      await clientService.update(client.id, { enabled: !client.enabled });
      refreshClients();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to toggle client');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDownloadConfig = async (client: Client) => {
    try {
      const config = await clientService.getConfig(client.id);
      const blob = new Blob([config.content], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = config.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to download config');
    }
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">
          Clients
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreate}
          disabled={actionLoading()}
        >
          New Client
        </Button>
      </Box>

      <Show when={actionError()}>
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setActionError('')}>
          {actionError()}
        </Alert>
      </Show>

      <Show when={clientsState.error}>
        <Alert severity="error" sx={{ mb: 2 }}>
          {clientsState.error}
        </Alert>
      </Show>

      <Show when={clientsState.loading} fallback={
        <ClientList
          onEdit={handleEdit}
          onDelete={handleDelete}
          onToggle={handleToggle}
          onQRCode={handleQRCode}
          onDownload={handleDownloadConfig}
          disabled={actionLoading()}
        />
      }>
        <Box display="flex" justifyContent="center" mt={4}>
          <CircularProgress />
        </Box>
      </Show>

      <ClientModal
        open={createModalOpen()}
        onClose={() => setCreateModalOpen(false)}
        onSuccess={() => {
          setCreateModalOpen(false);
          refreshClients();
        }}
      />

      <Show when={selectedClient()}>
        <ClientModal
          open={editModalOpen()}
          client={selectedClient()!}
          onClose={() => setEditModalOpen(false)}
          onSuccess={() => {
            setEditModalOpen(false);
            refreshClients();
          }}
        />

        <QRCodeModal
          open={qrModalOpen()}
          client={selectedClient()!}
          onClose={() => setQRModalOpen(false)}
        />
      </Show>
    </Container>
  );
};

export default Clients;