import { get, post, put, del } from './request';
import type { ApiKey, GenerateApiKeyRequest, UpdateApiKeyNameRequest } from '../types';

export interface ListApiKeysResponse {
    items: ApiKey[];
    total: number;
}

// 生成管理 API Key
export const generateManageApiKey = (data: GenerateApiKeyRequest) => {
    return post<ApiKey>('/admin/admin-api-keys', data);
};

// 获取管理 API Key 列表
export const listManageApiKeys = (pageIndex: number = 1, pageSize: number = 10, name?: string) => {
    const params = new URLSearchParams();
    params.append('pageIndex', pageIndex.toString());
    params.append('pageSize', pageSize.toString());
    params.append('type', 'admin');
    if (name) {
        params.append('name', name);
    }
    return get<ListApiKeysResponse>(`/admin/api-keys?${params.toString()}`);
};

// 获取管理 API Key 详情
export const getManageApiKey = (id: string) => {
    return get<ApiKey>(`/admin/api-keys/${id}`);
};

// 更新管理 API Key 名称
export const updateManageApiKeyName = (id: string, data: UpdateApiKeyNameRequest) => {
    return put(`/admin/api-keys/${id}`, data);
};

// 启用管理 API Key
export const enableManageApiKey = (id: string) => {
    return post(`/admin/api-keys/${id}/enable`, {});
};

// 禁用管理 API Key
export const disableManageApiKey = (id: string) => {
    return post(`/admin/api-keys/${id}/disable`, {});
};

// 删除管理 API Key
export const deleteManageApiKey = (id: string) => {
    return del(`/admin/api-keys/${id}`);
};
