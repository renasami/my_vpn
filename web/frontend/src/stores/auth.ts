import { createSignal, createEffect } from 'solid-js';

export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

const [isAuthenticated, setIsAuthenticated] = createSignal(false);
const [currentUser, setCurrentUser] = createSignal<User | null>(null);
const [authToken, setAuthToken] = createSignal<string | null>(null);

// Initialize auth state from localStorage
createEffect(() => {
  const token = localStorage.getItem('auth_token');
  const userStr = localStorage.getItem('user');
  
  if (token && userStr) {
    setAuthToken(token);
    setCurrentUser(JSON.parse(userStr));
    setIsAuthenticated(true);
  }
});

export const login = (response: AuthResponse) => {
  const { token, user } = response;
  
  localStorage.setItem('auth_token', token);
  localStorage.setItem('user', JSON.stringify(user));
  
  setAuthToken(token);
  setCurrentUser(user);
  setIsAuthenticated(true);
};

export const logout = () => {
  localStorage.removeItem('auth_token');
  localStorage.removeItem('user');
  
  setAuthToken(null);
  setCurrentUser(null);
  setIsAuthenticated(false);
};

export const getAuthToken = () => authToken();

export { isAuthenticated, currentUser };