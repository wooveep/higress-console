import type { ChangePasswordParams, LoginParams, UserInfo } from '@/interfaces/user';
import request from './request';

interface UserPayload extends Partial<UserInfo> {
  name?: string;
}

function normalizeUserInfo(payload: UserPayload | null | undefined): UserInfo {
  return {
    username: payload?.username || payload?.name || '',
    displayName: payload?.displayName || payload?.username || payload?.name || '',
    type: payload?.type,
    avatarUrl: payload?.avatarUrl,
  };
}

export async function login(data: LoginParams): Promise<UserInfo> {
  const response = await request.post<LoginParams, UserPayload>('/session/login', data);
  return normalizeUserInfo(response);
}

export async function logout() {
  return await request.get('/session/logout');
}

export async function fetchUserInfo(): Promise<UserInfo> {
  const response = await request.get<UserPayload>('/user/info');
  return normalizeUserInfo(response);
}

export async function changePassword(data: ChangePasswordParams): Promise<any> {
  return await request.post('/user/changePassword', data);
}
