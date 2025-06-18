import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { renderWithProviders } from '../test-utils';
import QRCodeModal from '@/components/QRCodeModal';
import type { Client } from '@/services/clients';

// Mock the client service
vi.mock('@/services/clients', () => ({
  clientService: {
    getQRCode: vi.fn(),
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

describe('QRCodeModal', () => {
  const mockOnClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render modal with client name', () => {
    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    expect(screen.getByText('QR Code for Test Client')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Base64' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'PNG Image' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Terminal' })).toBeInTheDocument();
  });

  it('should fetch QR code on mount', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'base64',
      data: 'base64-qr-data',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    await waitFor(() => {
      expect(clientService.getQRCode).toHaveBeenCalledWith(1, 'base64');
    });
  });

  it('should display base64 QR code', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'base64',
      data: 'base64-qr-data',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    await waitFor(() => {
      expect(screen.getByText('Base64 Encoded')).toBeInTheDocument();
      expect(screen.getByText('base64-qr-data')).toBeInTheDocument();
    });
  });

  it('should display PNG QR code', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'png',
      data: 'data:image/png;base64,png-data',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    const pngButton = screen.getByRole('button', { name: 'PNG Image' });
    fireEvent.click(pngButton);

    await waitFor(() => {
      expect(clientService.getQRCode).toHaveBeenCalledWith(1, 'png');
      expect(screen.getByText('QR Code Image')).toBeInTheDocument();
      expect(screen.getByRole('img', { name: 'QR Code' })).toBeInTheDocument();
    });
  });

  it('should display terminal QR code', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'terminal',
      data: '██████████████\n██          ██\n██████████████',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    const terminalButton = screen.getByRole('button', { name: 'Terminal' });
    fireEvent.click(terminalButton);

    await waitFor(() => {
      expect(clientService.getQRCode).toHaveBeenCalledWith(1, 'terminal');
      expect(screen.getByText('Terminal Display')).toBeInTheDocument();
    });
  });

  it('should handle copy to clipboard', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'base64',
      data: 'base64-qr-data',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    await waitFor(() => {
      expect(screen.getByText('base64-qr-data')).toBeInTheDocument();
    });

    const copyButton = screen.getByLabelText('Copy to clipboard');
    fireEvent.click(copyButton);

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('base64-qr-data');
  });

  it('should handle download', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockResolvedValue({
      format: 'base64',
      data: 'base64-qr-data',
    });

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    await waitFor(() => {
      expect(screen.getByText('base64-qr-data')).toBeInTheDocument();
    });

    const downloadButton = screen.getByLabelText('Download');
    fireEvent.click(downloadButton);

    expect(document.createElement).toHaveBeenCalledWith('a');
  });

  it('should handle API errors', async () => {
    const { clientService } = await import('@/services/clients');
    vi.mocked(clientService.getQRCode).mockRejectedValue(new Error('QR generation failed'));

    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    await waitFor(() => {
      expect(screen.getByText('QR generation failed')).toBeInTheDocument();
    });
  });

  it('should close modal when close button is clicked', () => {
    renderWithProviders(() => (
      <QRCodeModal
        open={true}
        client={mockClient}
        onClose={mockOnClose}
      />
    ));

    const closeButton = screen.getByRole('button', { name: 'Close' });
    fireEvent.click(closeButton);

    expect(mockOnClose).toHaveBeenCalled();
  });
});