import { describe, it, expect, vi, beforeEach } from 'vitest';
import { authService } from '@/services/auth';

// Mock the api module
vi.mock('@/services/api', () => ({
  api: {
    post: vi.fn(),
    get: vi.fn(),
  },
}));

describe('authService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('login', () => {
    it('should call API with correct credentials', async () => {
      const { api } = await import('@/services/api');
      const mockResponse = {
        token: 'test-token',
        user: { id: 1, username: 'testuser', email: 'test@example.com', role: 'user' },
      };
      
      vi.mocked(api.post).mockResolvedValue(mockResponse);

      const credentials = { username: 'testuser', password: 'password' };
      const result = await authService.login(credentials);

      expect(api.post).toHaveBeenCalledWith('/auth/login', credentials);
      expect(result).toEqual(mockResponse);
    });

    it('should handle login errors', async () => {
      const { api } = await import('@/services/api');
      const error = new Error('Invalid credentials');
      
      vi.mocked(api.post).mockRejectedValue(error);

      const credentials = { username: 'testuser', password: 'wrong' };
      
      await expect(authService.login(credentials)).rejects.toThrow('Invalid credentials');
    });
  });

  describe('register', () => {
    it('should call API with correct registration data', async () => {
      const { api } = await import('@/services/api');
      const mockResponse = {
        token: 'test-token',
        user: { id: 1, username: 'newuser', email: 'new@example.com', role: 'user' },
      };
      
      vi.mocked(api.post).mockResolvedValue(mockResponse);

      const data = { username: 'newuser', email: 'new@example.com', password: 'password' };
      const result = await authService.register(data);

      expect(api.post).toHaveBeenCalledWith('/auth/register', data);
      expect(result).toEqual(mockResponse);
    });
  });

  describe('refresh', () => {
    it('should call refresh endpoint', async () => {
      const { api } = await import('@/services/api');
      const mockResponse = { token: 'new-token' };
      
      vi.mocked(api.post).mockResolvedValue(mockResponse);

      const result = await authService.refresh();

      expect(api.post).toHaveBeenCalledWith('/auth/refresh');
      expect(result).toEqual(mockResponse);
    });
  });

  describe('getProfile', () => {
    it('should fetch user profile', async () => {
      const { api } = await import('@/services/api');
      const mockUser = { id: 1, username: 'testuser', email: 'test@example.com', role: 'user' };
      
      vi.mocked(api.get).mockResolvedValue(mockUser);

      const result = await authService.getProfile();

      expect(api.get).toHaveBeenCalledWith('/auth/profile');
      expect(result).toEqual(mockUser);
    });
  });

  describe('changePassword', () => {
    it('should call change password endpoint with correct data', async () => {
      const { api } = await import('@/services/api');
      
      vi.mocked(api.post).mockResolvedValue(undefined);

      await authService.changePassword('oldpass', 'newpass');

      expect(api.post).toHaveBeenCalledWith('/auth/change-password', {
        old_password: 'oldpass',
        new_password: 'newpass',
      });
    });
  });
});