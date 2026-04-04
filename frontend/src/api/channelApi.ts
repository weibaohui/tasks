/**
 * 渠道（Channel）API 调用模块
 */
import apiClient from './client';
import type { Channel, ChannelTypeOption, CreateChannelRequest, UpdateChannelRequest } from '../types/channel';

/**
 * 获取渠道列表
 */
export async function listChannels(userCode: string): Promise<Channel[]> {
  const response = await apiClient.get<Channel[]>('/channels', { params: { user_code: userCode } });
  return response.data;
}

/**
 * 创建渠道
 */
export async function createChannel(request: CreateChannelRequest): Promise<Channel> {
  const response = await apiClient.post<Channel>('/channels', request);
  return response.data;
}

/**
 * 更新渠道
 */
export async function updateChannel(id: string, request: UpdateChannelRequest): Promise<Channel> {
  const response = await apiClient.put<Channel>('/channels', request, { params: { id } });
  return response.data;
}

/**
 * 删除渠道
 */
export async function deleteChannel(id: string): Promise<void> {
  await apiClient.delete('/channels', { params: { id } });
}

export async function listChannelTypes(): Promise<ChannelTypeOption[]> {
  const response = await apiClient.get<ChannelTypeOption[]>('/channels/types');
  return response.data;
}
