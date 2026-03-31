import apiClient from './taskApi';
import type {
  LoginRequest,
  LoginResponse,
  User,
  UserToken,
  CreateTokenRequest,
  CreateTokenResponse,
  ListTokensResponse,
} from '../types/user';

export async function login(request: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/auth/login', request);
  // Store token in localStorage
  if (response.data.token) {
    window.localStorage.setItem('auth_token', response.data.token);
  }
  return response.data;
}

export async function getCurrentUser(): Promise<User> {
  const response = await apiClient.get<User>('/auth/me');
  return response.data;
}

// Token Management APIs
export async function createToken(request: CreateTokenRequest): Promise<CreateTokenResponse> {
  const response = await apiClient.post<CreateTokenResponse>('/users/tokens', request);
  return response.data;
}

export async function listTokens(): Promise<UserToken[]> {
  const response = await apiClient.get<ListTokensResponse>('/users/tokens');
  return response.data.tokens;
}

export async function deleteToken(tokenId: string): Promise<void> {
  await apiClient.delete(`/users/tokens/${tokenId}`);
}

// Clear auth token from localStorage
export function logout(): void {
  window.localStorage.removeItem('auth_token');
}
