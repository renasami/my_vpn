import { describe, it, expect, vi, beforeEach } from 'vitest';
import { clientService, fetchClients } from '@/services/clients';

// Mock the api module
vi.mock('@/services/api', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockClient = {
  id: 1,
  name: 'Test Client',
  public_key: 'test-key',
  ip_address: '10.0.0.2',
  enabled: true,
  created_at: '2023-01-01T00:00:00Z',
  updated_at: '2023-01-01T00:00:00Z',
  last_handshake: '2023-01-01T12:00:00Z',
  bytes_sent: 1024,
  bytes_received: 2048,
};

describe('clientService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('list', () => {
    it('should fetch all clients', async () => {
      const { api } = await import('@/services/api');
      const mockClients = [mockClient];
      
      vi.mocked(api.get).mockResolvedValue(mockClients);

      const result = await clientService.list();

      expect(api.get).toHaveBeenCalledWith('/clients');
      expect(result).toEqual(mockClients);
    });
  });

  describe('get', () => {
    it('should fetch a specific client', async () => {
      const { api } = await import('@/services/api');
      
      vi.mocked(api.get).mockResolvedValue(mockClient);

      const result = await clientService.get(1);

      expect(api.get).toHaveBeenCalledWith('/clients/1');
      expect(result).toEqual(mockClient);
    });
  });

  describe('create', () => {
    it('should create a new client', async () => {
      const { api } = await import('@/services/api');
      const createData = { name: 'New Client' };
      
      vi.mocked(api.post).mockResolvedValue(mockClient);

      const result = await clientService.create(createData);

      expect(api.post).toHaveBeenCalledWith('/clients', createData);
      expect(result).toEqual(mockClient);
    });
  });

  describe('update', () => {
    it('should update a client', async () => {
      const { api } = await import('@/services/api');
      const updateData = { name: 'Updated Client', enabled: false };
      
      vi.mocked(api.put).mockResolvedValue({ ...mockClient, ...updateData });

      const result = await clientService.update(1, updateData);

      expect(api.put).toHaveBeenCalledWith('/clients/1', updateData);
      expect(result).toEqual({ ...mockClient, ...updateData });
    });
  });

  describe('delete', () => {
    it('should delete a client', async () => {
      const { api } = await import('@/services/api');
      
      vi.mocked(api.delete).mockResolvedValue(undefined);

      await clientService.delete(1);

      expect(api.delete).toHaveBeenCalledWith('/clients/1');
    });
  });

  describe('getConfig', () => {
    it('should fetch client configuration', async () => {
      const { api } = await import('@/services/api');
      const mockConfig = {
        content: '[Interface]\nPrivateKey = ...',
        filename: 'client.conf',
      };
      
      vi.mocked(api.get).mockResolvedValue(mockConfig);

      const result = await clientService.getConfig(1);

      expect(api.get).toHaveBeenCalledWith('/clients/1/config');
      expect(result).toEqual(mockConfig);
    });
  });

  describe('getQRCode', () => {
    it('should fetch QR code with default format', async () => {
      const { api } = await import('@/services/api');
      const mockQR = {
        format: 'base64' as const,
        data: 'base64-data',
      };
      
      vi.mocked(api.get).mockResolvedValue(mockQR);

      const result = await clientService.getQRCode(1);

      expect(api.get).toHaveBeenCalledWith('/clients/1/qr', { format: 'base64' });
      expect(result).toEqual(mockQR);
    });

    it('should fetch QR code with specified format', async () => {
      const { api } = await import('@/services/api');
      const mockQR = {
        format: 'png' as const,
        data: 'png-data',
      };
      
      vi.mocked(api.get).mockResolvedValue(mockQR);

      const result = await clientService.getQRCode(1, 'png');

      expect(api.get).toHaveBeenCalledWith('/clients/1/qr', { format: 'png' });
      expect(result).toEqual(mockQR);
    });
  });

  describe('fetchClients', () => {
    it('should be an alias for list method', async () => {
      const { api } = await import('@/services/api');
      const mockClients = [mockClient];
      
      vi.mocked(api.get).mockResolvedValue(mockClients);

      const result = await fetchClients();

      expect(api.get).toHaveBeenCalledWith('/clients');
      expect(result).toEqual(mockClients);
    });
  });
});