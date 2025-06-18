import { createSignal, createResource } from 'solid-js';
import { createStore } from 'solid-js/store';
import { fetchClients, type Client } from '@/services/clients';

export interface ClientsState {
  clients: Client[];
  loading: boolean;
  error: string | null;
  searchTerm: string;
  sortBy: 'name' | 'ip' | 'status' | 'lastHandshake';
  sortOrder: 'asc' | 'desc';
  currentPage: number;
  pageSize: number;
}

const [state, setState] = createStore<ClientsState>({
  clients: [],
  loading: false,
  error: null,
  searchTerm: '',
  sortBy: 'name',
  sortOrder: 'asc',
  currentPage: 1,
  pageSize: 10,
});

// Create a resource for fetching clients
const [clients, { refetch }] = createResource(fetchClients);

// Update state when clients are fetched
createEffect(() => {
  const data = clients();
  if (data) {
    setState('clients', data);
    setState('loading', false);
    setState('error', null);
  }
});

// Handle loading state
createEffect(() => {
  if (clients.loading) {
    setState('loading', true);
  }
});

// Handle error state
createEffect(() => {
  if (clients.error) {
    setState('error', clients.error.message);
    setState('loading', false);
  }
});

export const refreshClients = () => refetch();

export const setSearchTerm = (term: string) => {
  setState('searchTerm', term);
  setState('currentPage', 1);
};

export const setSorting = (field: ClientsState['sortBy']) => {
  if (state.sortBy === field) {
    setState('sortOrder', state.sortOrder === 'asc' ? 'desc' : 'asc');
  } else {
    setState('sortBy', field);
    setState('sortOrder', 'asc');
  }
};

export const setPage = (page: number) => {
  setState('currentPage', page);
};

export const setPageSize = (size: number) => {
  setState('pageSize', size);
  setState('currentPage', 1);
};

// Computed values
export const filteredClients = () => {
  let filtered = state.clients;
  
  // Apply search filter
  if (state.searchTerm) {
    const term = state.searchTerm.toLowerCase();
    filtered = filtered.filter(client => 
      client.name.toLowerCase().includes(term) ||
      client.ip_address.toLowerCase().includes(term)
    );
  }
  
  // Apply sorting
  filtered = [...filtered].sort((a, b) => {
    let aVal: any, bVal: any;
    
    switch (state.sortBy) {
      case 'name':
        aVal = a.name;
        bVal = b.name;
        break;
      case 'ip':
        aVal = a.ip_address;
        bVal = b.ip_address;
        break;
      case 'status':
        aVal = a.enabled ? 1 : 0;
        bVal = b.enabled ? 1 : 0;
        break;
      case 'lastHandshake':
        aVal = a.last_handshake ? new Date(a.last_handshake).getTime() : 0;
        bVal = b.last_handshake ? new Date(b.last_handshake).getTime() : 0;
        break;
    }
    
    if (aVal < bVal) return state.sortOrder === 'asc' ? -1 : 1;
    if (aVal > bVal) return state.sortOrder === 'asc' ? 1 : -1;
    return 0;
  });
  
  return filtered;
};

export const paginatedClients = () => {
  const filtered = filteredClients();
  const start = (state.currentPage - 1) * state.pageSize;
  const end = start + state.pageSize;
  return filtered.slice(start, end);
};

export const totalPages = () => {
  return Math.ceil(filteredClients().length / state.pageSize);
};

export { state as clientsState };

// Import for createEffect
import { createEffect } from 'solid-js';