import apiClient from './taskApi';
import type { CreateUserRequest, UpdateUserRequest, User } from '../types/user';

export async function listUsers(): Promise<User[]> {
  const response = await apiClient.get<User[]>('/users');
  return response.data;
}

export async function createUser(request: CreateUserRequest): Promise<User> {
  const response = await apiClient.post<User>('/users', request);
  return response.data;
}

export async function updateUser(id: string, request: UpdateUserRequest): Promise<User> {
  const response = await apiClient.put<User>('/users', request, {
    params: { id },
  });
  return response.data;
}

export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete('/users', { params: { id } });
}
