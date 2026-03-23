import apiClient from './taskApi';
import type { LoginRequest, LoginResponse, User } from '../types/user';

export async function login(request: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/auth/login', request);
  return response.data;
}

export async function getCurrentUser(): Promise<User> {
  const response = await apiClient.get<User>('/auth/me');
  return response.data;
}
