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
  token_id: string;
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
  email?: string;
  display_name?: string;
  is_active?: boolean;
}

// API Token types
export interface UserToken {
  id: string;
  name: string;
  description: string;
  expires_at?: number; // Unix milliseconds, undefined means permanent
  created_at: number;
  last_used_at?: number;
  is_active: boolean;
  is_expired: boolean;
}

export interface CreateTokenRequest {
  name: string;
  description: string;
  expires_in_days: number; // 0 or negative means permanent
}

export interface CreateTokenResponse {
  token: string; // Only shown once when created
  id: string;
  name: string;
  description: string;
  expires_at?: number;
  created_at: number;
  is_active: boolean;
  is_expired: boolean;
}

export interface ListTokensResponse {
  tokens: UserToken[];
}
