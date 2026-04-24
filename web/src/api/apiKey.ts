import { get, post, put, del } from './request';
import type { ApiKey, GenerateApiKeyRequest, UpdateApiKeyNameRequest } from '../types';

export interface ListApiKeysResponse {
    items: ApiKey[];
    total: number;
}

// 生成 API Key
export const generateApiKey = (data: GenerateApiKeyRequest) => {
    return post<ApiKey>('/admin/api-keys', data);
};

// 获取 API Key 列表
export const listApiKeys = (pageIndex: number = 1, pageSize: number = 10, name?: string) => {
    const params = new URLSearchParams();
    params.append('pageIndex', pageIndex.toString());
    params.append('pageSize', pageSize.toString());
    params.append('type', 'agent');
    if (name) {
        params.append('name', name);
    }
    return get<ListApiKeysResponse>(`/admin/api-keys?${params.toString()}`);
};

// 获取 API Key 详情
export const getApiKey = (id: string) => {
    return get<ApiKey>(`/admin/api-keys/${id}`);
};

// 更新 API Key 名称
export const updateApiKeyName = (id: string, data: UpdateApiKeyNameRequest) => {
    return put(`/admin/api-keys/${id}`, data);
};

// 启用 API Key
export const enableApiKey = (id: string) => {
    return post(`/admin/api-keys/${id}/enable`, {});
};

// 禁用 API Key
export const disableApiKey = (id: string) => {
    return post(`/admin/api-keys/${id}/disable`, {});
};

// 删除 API Key
export const deleteApiKey = (id: string) => {
    return del(`/admin/api-keys/${id}`);
};

// 获取 API Key 完整值
export const getApiKeyRaw = (id: string) => {
    return get<{ key: string }>(`/admin/api-keys/${id}/raw`);
};
