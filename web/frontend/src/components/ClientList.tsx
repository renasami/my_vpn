import { Component, For, Show } from 'solid-js';
import {
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  IconButton,
  Chip,
  TextField,
  Box,
  Tooltip,
  TableSortLabel,
} from '@suid/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
  QrCode as QrCodeIcon,
  Download as DownloadIcon,
  PowerSettingsNew as PowerIcon,
} from '@suid/icons-material';
import { 
  clientsState, 
  paginatedClients, 
  totalPages,
  setSearchTerm,
  setSorting,
  setPage,
  setPageSize,
} from '@/stores/clients';
import type { Client } from '@/services/clients';

interface ClientListProps {
  onEdit: (client: Client) => void;
  onDelete: (client: Client) => void;
  onToggle: (client: Client) => void;
  onQRCode: (client: Client) => void;
  onDownload: (client: Client) => void;
  disabled?: boolean;
}

const ClientList: Component<ClientListProps> = (props) => {
  const formatDate = (date: string) => {
    return new Date(date).toLocaleString();
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getLastHandshakeStatus = (lastHandshake?: string) => {
    if (!lastHandshake) return 'never';
    
    const diff = Date.now() - new Date(lastHandshake).getTime();
    const minutes = Math.floor(diff / 60000);
    
    if (minutes < 5) return 'active';
    if (minutes < 30) return 'recent';
    return 'inactive';
  };

  return (
    <Paper sx={{ width: '100%', overflow: 'hidden' }}>
      <Box sx={{ p: 2 }}>
        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search by name or IP address..."
          value={clientsState.searchTerm}
          onChange={(e) => setSearchTerm(e.currentTarget.value)}
        />
      </Box>

      <TableContainer sx={{ maxHeight: 600 }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell>
                <TableSortLabel
                  active={clientsState.sortBy === 'name'}
                  direction={clientsState.sortBy === 'name' ? clientsState.sortOrder : 'asc'}
                  onClick={() => setSorting('name')}
                >
                  Name
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={clientsState.sortBy === 'ip'}
                  direction={clientsState.sortBy === 'ip' ? clientsState.sortOrder : 'asc'}
                  onClick={() => setSorting('ip')}
                >
                  IP Address
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={clientsState.sortBy === 'status'}
                  direction={clientsState.sortBy === 'status' ? clientsState.sortOrder : 'asc'}
                  onClick={() => setSorting('status')}
                >
                  Status
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={clientsState.sortBy === 'lastHandshake'}
                  direction={clientsState.sortBy === 'lastHandshake' ? clientsState.sortOrder : 'asc'}
                  onClick={() => setSorting('lastHandshake')}
                >
                  Last Handshake
                </TableSortLabel>
              </TableCell>
              <TableCell>Data Transfer</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <For each={paginatedClients()}>
              {(client) => (
                <TableRow hover>
                  <TableCell>{client.name}</TableCell>
                  <TableCell>{client.ip_address}</TableCell>
                  <TableCell>
                    <Box display="flex" alignItems="center" gap={1}>
                      <Chip
                        label={client.enabled ? 'Enabled' : 'Disabled'}
                        color={client.enabled ? 'success' : 'default'}
                        size="small"
                      />
                      <Show when={client.enabled && client.last_handshake}>
                        <Chip
                          label={getLastHandshakeStatus(client.last_handshake)}
                          color={
                            getLastHandshakeStatus(client.last_handshake) === 'active' ? 'success' :
                            getLastHandshakeStatus(client.last_handshake) === 'recent' ? 'warning' :
                            'error'
                          }
                          size="small"
                          variant="outlined"
                        />
                      </Show>
                    </Box>
                  </TableCell>
                  <TableCell>
                    {client.last_handshake ? formatDate(client.last_handshake) : 'Never'}
                  </TableCell>
                  <TableCell>
                    ↓ {formatBytes(client.bytes_received)} / ↑ {formatBytes(client.bytes_sent)}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title={client.enabled ? 'Disable' : 'Enable'}>
                      <IconButton
                        size="small"
                        onClick={() => props.onToggle(client)}
                        disabled={props.disabled}
                        color={client.enabled ? 'success' : 'default'}
                      >
                        <PowerIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Edit">
                      <IconButton
                        size="small"
                        onClick={() => props.onEdit(client)}
                        disabled={props.disabled}
                      >
                        <EditIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="QR Code">
                      <IconButton
                        size="small"
                        onClick={() => props.onQRCode(client)}
                        disabled={props.disabled}
                      >
                        <QrCodeIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Download Config">
                      <IconButton
                        size="small"
                        onClick={() => props.onDownload(client)}
                        disabled={props.disabled}
                      >
                        <DownloadIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete">
                      <IconButton
                        size="small"
                        onClick={() => props.onDelete(client)}
                        disabled={props.disabled}
                        color="error"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              )}
            </For>
          </TableBody>
        </Table>
      </TableContainer>

      <TablePagination
        rowsPerPageOptions={[5, 10, 25, 50]}
        component="div"
        count={clientsState.clients.length}
        rowsPerPage={clientsState.pageSize}
        page={clientsState.currentPage - 1}
        onPageChange={(event, newPage) => setPage(newPage + 1)}
        onRowsPerPageChange={(event) => setPageSize(parseInt(event.target.value, 10))}
      />
    </Paper>
  );
};

export default ClientList;