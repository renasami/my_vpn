import { api } from './api';

export interface Client {
  id: number;
  name: string;
  public_key: string;
  ip_address: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_handshake?: string;
  bytes_sent: number;
  bytes_received: number;
}

export interface CreateClientRequest {
  name: string;
}

export interface UpdateClientRequest {
  name?: string;
  enabled?: boolean;
}

export interface ClientConfig {
  content: string;
  filename: string;
}

export interface QRCodeResponse {
  format: 'png' | 'base64' | 'terminal';
  data: string;
}

export const fetchClients = async (): Promise<Client[]> => {
  return api.get<Client[]>('/clients');
};

export const clientService = {
  list: fetchClients,
  
  get: async (id: number): Promise<Client> => {
    return api.get<Client>(`/clients/${id}`);
  },
  
  create: async (data: CreateClientRequest): Promise<Client> => {
    return api.post<Client>('/clients', data);
  },
  
  update: async (id: number, data: UpdateClientRequest): Promise<Client> => {
    return api.put<Client>(`/clients/${id}`, data);
  },
  
  delete: async (id: number): Promise<void> => {
    return api.delete(`/clients/${id}`);
  },
  
  getConfig: async (id: number): Promise<ClientConfig> => {
    return api.get<ClientConfig>(`/clients/${id}/config`);
  },
  
  getQRCode: async (id: number, format: 'png' | 'base64' | 'terminal' = 'base64'): Promise<QRCodeResponse> => {
    return api.get<QRCodeResponse>(`/clients/${id}/qr`, { format });
  },
};