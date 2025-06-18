import { api } from './api';
import type { AuthResponse, User } from '@/stores/auth';

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export const authService = {
  login: async (credentials: LoginRequest): Promise<AuthResponse> => {
    return api.post<AuthResponse>('/auth/login', credentials);
  },
  
  register: async (data: RegisterRequest): Promise<AuthResponse> => {
    return api.post<AuthResponse>('/auth/register', data);
  },
  
  refresh: async (): Promise<{ token: string }> => {
    return api.post<{ token: string }>('/auth/refresh');
  },
  
  getProfile: async (): Promise<User> => {
    return api.get<User>('/auth/profile');
  },
  
  changePassword: async (oldPassword: string, newPassword: string): Promise<void> => {
    return api.post('/auth/change-password', { 
      old_password: oldPassword,
      new_password: newPassword,
    });
  },
};