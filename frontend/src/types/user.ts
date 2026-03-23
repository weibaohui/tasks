export interface User {
  id: string;
  user_code: string;
  username: string;
  email: string;
  display_name: string;
  is_active: boolean;
  created_at: number;
  updated_at: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  expires_at: number;
  user: User;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  display_name: string;
  password: string;
}

export interface UpdateUserRequest {
  email: string;
  display_name: string;
  is_active?: boolean;
}
