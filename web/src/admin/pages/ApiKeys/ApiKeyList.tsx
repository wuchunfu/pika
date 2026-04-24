import {useEffect, useState} from 'react';
import {useSearchParams, useNavigate} from 'react-router-dom';
import {App, Button, Divider, Input, Popconfirm, Space, Table, Tag} from 'antd';
import type {ColumnsType, TablePaginationConfig} from 'antd/es/table';
import {Copy, Edit, Loader2, Plus, RefreshCw, Terminal, Trash2} from 'lucide-react';
import {deleteApiKey, disableApiKey, enableApiKey, getApiKeyRaw, listApiKeys} from '@/api/apiKey.ts';
import type {ApiKey} from '@/types';
import dayjs from 'dayjs';
import {getErrorMessage} from '@/lib/utils';
import {PageHeader} from '@admin/components';
import copy from 'copy-to-clipboard';
import ApiKeyModal from './ApiKeyModal';
import ShowApiKeyModal from './ShowApiKeyModal';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';

const ApiKeyList = () => {
    const {message: messageApi} = App.useApp();
    const queryClient = useQueryClient();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const [searchValue, setSearchValue] = useState('');
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [editingApiKeyId, setEditingApiKeyId] = useState<string | undefined>(undefined);
    const [newApiKeyData, setNewApiKeyData] = useState<ApiKey | null>(null);
    const [showApiKeyModal, setShowApiKeyModal] = useState(false);
    const [copyingKeyId, setCopyingKeyId] = useState<string | null>(null);

    const pageIndex = Number(searchParams.get('pageIndex')) || 1;
    const pageSize = Number(searchParams.get('pageSize')) || 10;
    const name = searchParams.get('name') ?? '';

    const {
        data: apiKeyPaging,
        isLoading,
        isFetching,
        refetch,
    } = useQuery({
        queryKey: ['admin', 'api-keys', pageIndex, pageSize, name],
        queryFn: async () => {
            const response = await listApiKeys(pageIndex, pageSize, name || undefined);
            return response.data;
        },
    });

    const toggleMutation = useMutation({
        mutationFn: async (apiKey: ApiKey) => {
            if (apiKey.enabled) {
                await disableApiKey(apiKey.id);
            } else {
                await enableApiKey(apiKey.id);
            }
        },
        onSuccess: (_, apiKey) => {
            messageApi.success(apiKey.enabled ? '通信密钥已禁用' : '通信密钥已启用');
            queryClient.invalidateQueries({queryKey: ['admin', 'api-keys']});
        },
        onError: (error: unknown) => {
            messageApi.error(getErrorMessage(error, '操作失败'));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: (id: string) => deleteApiKey(id),
        onSuccess: () => {
            messageApi.success('删除成功');
            queryClient.invalidateQueries({queryKey: ['admin', 'api-keys']});
        },
        onError: (error: unknown) => {
            messageApi.error(getErrorMessage(error, '删除失败'));
        },
    });

    useEffect(() => {
        setSearchValue(name);
    }, [name]);

    // 处理表格变化
    const handleTableChange = (newPagination: TablePaginationConfig) => {
        const nextParams = new URLSearchParams(searchParams);
        nextParams.set('pageIndex', String(newPagination.current || 1));
        nextParams.set('pageSize', String(newPagination.pageSize || pageSize));
        setSearchParams(nextParams);
    };

    // 处理搜索
    const handleSearch = (value: string) => {
        const keyword = value.trim();
        setSearchValue(keyword);
        const nextParams = new URLSearchParams(searchParams);
        if (keyword) {
            nextParams.set('name', keyword);
        } else {
            nextParams.delete('name');
        }
        nextParams.set('pageIndex', '1');
        nextParams.set('pageSize', String(pageSize));
        setSearchParams(nextParams);
    };

    const handleCreate = () => {
        setEditingApiKeyId(undefined);
        setIsModalVisible(true);
    };

    const handleEdit = (apiKey: ApiKey) => {
        setEditingApiKeyId(apiKey.id);
        setIsModalVisible(true);
    };

    const handleToggleEnabled = (apiKey: ApiKey) => {
        toggleMutation.mutate(apiKey);
    };

    const handleDelete = (id: string) => {
        deleteMutation.mutate(id);
    };

    const handleModalSuccess = (apiKey?: ApiKey) => {
        setIsModalVisible(false);
        if (apiKey) {
            // 新建成功，显示生成的密钥
            setNewApiKeyData(apiKey);
            setShowApiKeyModal(true);
        }
        queryClient.invalidateQueries({queryKey: ['admin', 'api-keys']});
    };

    const handleCopyApiKey = async (id: string) => {
        setCopyingKeyId(id);
        try {
            const response = await getApiKeyRaw(id);
            copy(response.data.key || '');
            messageApi.success('复制成功');
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '复制密钥失败'));
        } finally {
            setCopyingKeyId(null);
        }
    };

    const columns: ColumnsType<ApiKey> = [
        {
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            width: 220,
            render: (text) => <span className="font-medium text-gray-900 dark:text-white">{text}</span>,
        },
        {
            title: '通信密钥',
            dataIndex: 'key',
            key: 'key',
            width: 260,
            render: (_, record) => {
                const isCopying = copyingKeyId === record.id;
                return (
                    <div className="flex items-center gap-2">
                        <code
                            className="text-xs bg-gray-100 dark:bg-gray-800 dark:text-gray-200 px-2 py-1 rounded font-mono">
                            {record.key || ''}
                        </code>
                        <Button
                            type="text"
                            size="small"
                            icon={isCopying ? <Loader2 size={14} className="animate-spin"/> : <Copy size={14}/>}
                            onClick={() => void handleCopyApiKey(record.id)}
                            disabled={isCopying}
                            title="复制完整密钥"
                        />
                    </div>
                );
            },
        },
        {
            title: '状态',
            dataIndex: 'enabled',
            key: 'enabled',
            render: (enabled) => (
                <Tag color={enabled ? 'green' : 'red'}>{enabled ? '启用' : '禁用'}</Tag>
            ),
            width: 80,
        },
        {
            title: '创建时间',
            dataIndex: 'createdAt',
            key: 'createdAt',
            render: (value: number) => (
                <span className="text-gray-600 dark:text-gray-400">{dayjs(value).format('YYYY-MM-DD HH:mm')}</span>
            ),
            width: 180,
        },
        {
            title: '更新时间',
            dataIndex: 'updatedAt',
            key: 'updatedAt',
            render: (value: number) => (
                <span className="text-gray-600 dark:text-gray-400">{dayjs(value).format('YYYY-MM-DD HH:mm')}</span>
            ),
            width: 180,
        },
        {
            title: '操作',
            key: 'action',
            width: 200,
            render: (_, record) => [
                <Button
                    key="edit"
                    type="link"
                    size="small"
                    onClick={() => handleEdit(record)}
                >
                    编辑
                </Button>,
                <Button
                    key="toggle"
                    type="link"
                    size="small"
                    onClick={() => handleToggleEnabled(record)}
                >
                    {record.enabled ? '禁用' : '启用'}
                </Button>,
                <Popconfirm
                    key="delete"
                    title="确定要删除这个通信密钥吗?"
                    description="删除后无法恢复,且使用该密钥的探针将无法连接"
                    onConfirm={() => handleDelete(record.id)}
                    okText="确定"
                    cancelText="取消"
                >
                    <Button type="link"
                            size="small"
                            danger
                    >
                        删除
                    </Button>
                </Popconfirm>,
            ],
        },
    ];

    return (
        <div className="space-y-6">
            <PageHeader
                title="通信密钥管理"
                description="管理探针连接所需的通信密钥，用于验证探针注册与数据上报"
                actions={[
                    {
                        key: 'deploy',
                        label: '部署指南',
                        icon: <Terminal size={16}/>,
                        onClick: () => navigate('/admin/agents-install/one-click'),
                    },
                    {
                        key: 'create',
                        label: '生成密钥',
                        icon: <Plus size={16}/>,
                        type: 'primary',
                        onClick: handleCreate,
                    },
                    {
                        key: 'refresh',
                        label: '刷新',
                        icon: <RefreshCw size={16}/>,
                        onClick: () => refetch(),
                    },
                ]}
            />

            <div className="bg-white dark:bg-[#1c1c21] rounded-2xl border border-gray-100 dark:border-white/5 shadow-sm p-4 sm:p-6 space-y-4">
                <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                    <Input.Search
                        placeholder="按名称搜索"
                        allowClear
                        value={searchValue}
                        onChange={(event) => {
                            const nextValue = event.target.value;
                            setSearchValue(nextValue);
                            if (!nextValue) {
                                handleSearch('');
                            }
                        }}
                        onSearch={handleSearch}
                        className="w-full max-w-md"
                    />
                </div>

                <Table<ApiKey>
                    columns={columns}
                    dataSource={apiKeyPaging?.items || []}
                    loading={isLoading || isFetching}
                    rowKey="id"
                    scroll={{x: 900}}
                    tableLayout="fixed"
                    pagination={{
                        current: pageIndex,
                        pageSize,
                        total: apiKeyPaging?.total || 0,
                        showSizeChanger: true,
                    }}
                    onChange={handleTableChange}
                />
            </div>

            <ApiKeyModal
                open={isModalVisible}
                apiKeyId={editingApiKeyId}
                onCancel={() => setIsModalVisible(false)}
                onSuccess={handleModalSuccess}
            />

            <ShowApiKeyModal
                open={showApiKeyModal}
                apiKey={newApiKeyData}
                onClose={() => {
                    setShowApiKeyModal(false);
                    setNewApiKeyData(null);
                }}
            />
        </div>
    );
};

export default ApiKeyList;
