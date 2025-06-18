import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { renderWithProviders } from '../test-utils';
import ClientModal from '@/components/ClientModal';
import type { Client } from '@/services/clients';

// Mock the client service
vi.mock('@/services/clients', () => ({
  clientService: {
    create: vi.fn(),
    update: vi.fn(),
  },
}));

const mockClient: Client = {
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

describe('ClientModal', () => {
  const mockOnClose = vi.fn();
  const mockOnSuccess = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render create modal correctly', () => {
    renderWithProviders(() => (
      <ClientModal
        open={true}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    expect(screen.getByText('Create New Client')).toBeInTheDocument();
    expect(screen.getByLabelText('Client Name')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create' })).toBeInTheDocument();
  });

  it('should render edit modal correctly', () => {
    renderWithProviders(() => (
      <ClientModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    expect(screen.getByText('Edit Client')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Test Client')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Update' })).toBeInTheDocument();
    expect(screen.getByText('IP Address: 10.0.0.2')).toBeInTheDocument();
  });

  it('should handle create client submission', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.create).mockResolvedValue(mockClient);

    renderWithProviders(() => (
      <ClientModal
        open={true}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    const nameInput = screen.getByLabelText('Client Name');
    const submitButton = screen.getByRole('button', { name: 'Create' });

    fireEvent.input(nameInput, { target: { value: 'New Client' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(clientService.create).toHaveBeenCalledWith({ name: 'New Client' });
      expect(mockOnSuccess).toHaveBeenCalled();
    });
  });

  it('should handle update client submission', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.update).mockResolvedValue(mockClient);

    renderWithProviders(() => (
      <ClientModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    const nameInput = screen.getByDisplayValue('Test Client');
    const submitButton = screen.getByRole('button', { name: 'Update' });

    fireEvent.input(nameInput, { target: { value: 'Updated Client' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(clientService.update).toHaveBeenCalledWith(1, {
        name: 'Updated Client',
        enabled: true,
      });
      expect(mockOnSuccess).toHaveBeenCalled();
    });
  });

  it('should show validation error for empty name', async () => {
    renderWithProviders(() => (
      <ClientModal
        open={true}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    const submitButton = screen.getByRole('button', { name: 'Create' });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Name is required')).toBeInTheDocument();
    });
  });

  it('should handle API errors', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.create).mockRejectedValue(new Error('API Error'));

    renderWithProviders(() => (
      <ClientModal
        open={true}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    const nameInput = screen.getByLabelText('Client Name');
    const submitButton = screen.getByRole('button', { name: 'Create' });

    fireEvent.input(nameInput, { target: { value: 'Test Client' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('API Error')).toBeInTheDocument();
    });
  });

  it('should close modal when cancel is clicked', () => {
    renderWithProviders(() => (
      <ClientModal
        open={true}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    const cancelButton = screen.getByRole('button', { name: 'Cancel' });
    fireEvent.click(cancelButton);

    expect(mockOnClose).toHaveBeenCalled();
  });

  it('should not render when closed', () => {
    renderWithProviders(() => (
      <ClientModal
        open={false}
        onClose={mockOnClose}
        onSuccess={mockOnSuccess}
      />
    ));

    expect(screen.queryByText('Create New Client')).not.toBeInTheDocument();
  });
});