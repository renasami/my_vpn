import { Component, For, Show } from 'solid-js';
import {
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  TextField,
  Box,
  Button,
} from '@suid/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
  QrCode as QrCodeIcon,
} from '@suid/icons-material';
import type { Client } from '@/services/clients';

interface ClientListProps {
  clients: Client[];
  searchTerm: string;
  onSearchChange: (term: string) => void;
  sortBy: keyof Client;
  sortOrder: 'asc' | 'desc';
  onSort: (column: keyof Client) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rows: number) => void;
  onEdit: (client: Client) => void;
  onDelete: (client: Client) => void;
  onShowQR: (client: Client) => void;
}

const ClientList: Component<ClientListProps> = (props) => {
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getStatusColor = (enabled: boolean) => {
    return enabled ? 'success' : 'default';
  };

  return (
    <Paper>
      <Box sx={{ p: 2 }}>
        <TextField
          label="Search clients..."
          variant="outlined"
          size="small"
          value={props.searchTerm}
          onChange={(e) => props.onSearchChange(e.target.value)}
          sx={{ mb: 2 }}
        />
      </Box>
      
      <TableContainer>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>
                <Button onClick={() => props.onSort('name')}>
                  Name
                </Button>
              </TableCell>
              <TableCell>IP Address</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Last Handshake</TableCell>
              <TableCell>Data Sent</TableCell>
              <TableCell>Data Received</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <For each={props.clients}>
              {(client) => (
                <TableRow>
                  <TableCell>{client.name}</TableCell>
                  <TableCell>{client.ip_address}</TableCell>
                  <TableCell>
                    <Chip
                      label={client.enabled ? 'Active' : 'Inactive'}
                      color={getStatusColor(client.enabled)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Show when={client.last_handshake} fallback="Never">
                      {formatDate(client.last_handshake!)}
                    </Show>
                  </TableCell>
                  <TableCell>{formatBytes(client.bytes_sent || 0)}</TableCell>
                  <TableCell>{formatBytes(client.bytes_received || 0)}</TableCell>
                  <TableCell>
                    <IconButton onClick={() => props.onEdit(client)}>
                      <EditIcon />
                    </IconButton>
                    <IconButton onClick={() => props.onDelete(client)}>
                      <DeleteIcon />
                    </IconButton>
                    <IconButton onClick={() => props.onShowQR(client)}>
                      <QrCodeIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              )}
            </For>
          </TableBody>
        </Table>
      </TableContainer>
      
      <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          Showing {props.clients.length} clients
        </div>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button 
            disabled={props.page === 0}
            onClick={() => props.onPageChange(props.page - 1)}
          >
            Previous
          </Button>
          <Button 
            onClick={() => props.onPageChange(props.page + 1)}
          >
            Next
          </Button>
        </Box>
      </Box>
    </Paper>
  );
};

export default ClientList;