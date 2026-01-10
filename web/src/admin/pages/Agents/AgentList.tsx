import {useEffect, useRef, useState} from 'react';
import {Link, useNavigate} from 'react-router-dom';
import type {ActionType, ProColumns} from '@ant-design/pro-components';
import {ProTable} from '@ant-design/pro-components';
import type {MenuProps} from 'antd';
import {App, Button, DatePicker, Divider, Dropdown, Form, Input, InputNumber, Modal, Radio, Select, Space, Tag} from 'antd';
import {Edit, Eye, MoreVertical, Plus, RefreshCw, Shield, Tags, Trash2} from 'lucide-react';
import {batchUpdateTags, deleteAgent, getAgentPaging, getTags, updateAgentInfo} from '@/api/agent.ts';
import type {Agent} from '@/types';
import {getErrorMessage} from '@/lib/utils';
import dayjs from 'dayjs';
import {PageHeader} from '@admin/components';

const AgentList = () => {
    const navigate = useNavigate();
    const {message: messageApi, modal} = App.useApp();
    const actionRef = useRef<ActionType>(null);
    const [form] = Form.useForm();
    const [batchForm] = Form.useForm();
    const [editModalVisible, setEditModalVisible] = useState(false);
    const [batchTagModalVisible, setBatchTagModalVisible] = useState(false);
    const [currentAgent, setCurrentAgent] = useState<Agent | null>(null);
    const [loading, setLoading] = useState(false);
    const [batchLoading, setBatchLoading] = useState(false);
    const [existingTags, setExistingTags] = useState<string[]>([]);
    const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

    // 加载已有的标签
    useEffect(() => {
        const loadTags = async () => {
            try {
                const response = await getTags();
                setExistingTags(response.data.tags || []);
            } catch (error) {
                console.error('加载标签失败:', error);
            }
        };
        loadTags();
    }, []);

    // 打开编辑模态框
    const handleEdit = (agent: Agent) => {
        setCurrentAgent(agent);
        form.setFieldsValue({
            name: agent.name,
            tags: agent.tags || [],
            expireTime: agent.expireTime ? dayjs(agent.expireTime) : null,
            visibility: agent.visibility || 'public',
            weight: agent.weight || 0,
            remark: agent.remark || '',
        });
        setEditModalVisible(true);
    };


    // 保存探针信息
    const handleSave = async () => {
        if (!currentAgent) return;

        try {
            const values = await form.validateFields();
            setLoading(true);

            // 转换到期时间为时间戳（设置为当天的23:59:59）
            const data: any = {
                name: values.name,
                visibility: values.visibility || 'public',
                tags: values.tags || [],
                weight: values.weight || 0,
                remark: values.remark || '',
            };

            if (values.expireTime) {
                data.expireTime = values.expireTime.endOf('day').valueOf();
            }

            await updateAgentInfo(currentAgent.id, data);

            messageApi.success('探针信息更新成功');
            setEditModalVisible(false);
            actionRef.current?.reload();
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '更新探针信息失败'));
        } finally {
            setLoading(false);
        }
    };

    // 删除探针
    const handleDelete = (agent: Agent) => {
        modal.confirm({
            title: '删除探针',
            content: (
                <div>
                    <p>确定要删除探针「{agent.name || agent.hostname}」吗？</p>
                    <p className="text-red-500 text-sm mt-2">
                        警告：此操作将删除探针及其所有相关数据（指标数据、监控统计、审计结果等），且不可恢复！
                    </p>
                </div>
            ),
            okText: '确认删除',
            cancelText: '取消',
            okButtonProps: {danger: true},
            centered: true,
            onOk: async () => {
                try {
                    await deleteAgent(agent.id);
                    messageApi.success('探针删除成功');
                    actionRef.current?.reload();
                } catch (error: unknown) {
                    messageApi.error(getErrorMessage(error, '删除探针失败'));
                }
            },
        });
    };

    // 打开批量操作标签模态框
    const handleBatchTags = () => {
        if (selectedRowKeys.length === 0) {
            messageApi.warning('请先选择要操作的探针');
            return;
        }
        batchForm.setFieldsValue({
            operation: 'add',
            tags: [],
        });
        setBatchTagModalVisible(true);
    };

    // 批量更新标签
    const handleBatchSave = async () => {
        try {
            const values = await batchForm.validateFields();
            setBatchLoading(true);

            await batchUpdateTags({
                agentIds: selectedRowKeys as string[],
                tags: values.tags || [],
                operation: values.operation,
            });

            messageApi.success(`成功${values.operation === 'add' ? '添加' : values.operation === 'remove' ? '移除' : '替换'}了 ${selectedRowKeys.length} 个探针的标签`);
            setBatchTagModalVisible(false);
            setSelectedRowKeys([]);
            actionRef.current?.reload();
        } catch (error: unknown) {
            messageApi.error(getErrorMessage(error, '批量更新标签失败'));
        } finally {
            setBatchLoading(false);
        }
    };

    const columns: ProColumns<Agent>[] = [
        {
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            fixed: 'left',
            render: (_, record) => (
                <div className="space-y-1">
                    <div className="font-medium">
                        <Link to={`/admin/agents/${record.id}`}>
                            {record.name || record.hostname}
                        </Link>
                    </div>
                    <Tag color="geekblue" variant={'filled'}>{record.os} · {record.arch}</Tag>
                </div>
            ),
        },
        {
            title: '标签',
            dataIndex: 'tags',
            key: 'tags',
            hideInSearch: true,
            width: 200,
            render: (_, record) => (
                <div className={'flex items-center gap-1'}>
                    {record.tags?.map((tag, index) => (
                        <Tag key={index} color="blue" variant={'filled'}>
                            {tag}
                        </Tag>
                    ))}
                </div>
            ),
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            hideInSearch: true,
            width: 80,
            render: (_, record) => (
                <Tag color={record.status === 1 ? 'success' : 'default'}>
                    {record.status === 1 ? '在线' : '离线'}
                </Tag>
            ),
        },
        {
            title: '可见性',
            dataIndex: 'visibility',
            key: 'visibility',
            hideInSearch: true,
            width: 100,
            render: (visibility) => (
                <Tag color={visibility === 'public' ? 'green' : 'orange'}>
                    {visibility === 'public' ? '匿名可见' : '登录可见'}
                </Tag>
            ),
        },
        {
            title: '主机名',
            dataIndex: 'hostname',
            key: 'hostname',
            ellipsis: true,
            width: 150,
        },
        {
            title: '状态筛选',
            dataIndex: 'status',
            valueType: 'select',
            hideInTable: true,
            valueEnum: {
                online: {text: '在线'},
                offline: {text: '离线'},
            },
        },
        {
            title: '版本',
            dataIndex: 'version',
            key: 'version',
            hideInSearch: true,
        },
        {
            title: '到期时间',
            dataIndex: 'expireTime',
            key: 'expireTime',
            hideInSearch: true,
            width: 100,
            render: (val) => {
                if (!val) return '-';
                const expireDate = new Date(val as number);
                const now = new Date();
                const isExpired = expireDate < now;
                const daysLeft = Math.ceil((expireDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

                return (
                    <div className="flex flex-col gap-1">
                        <div>{expireDate.toLocaleDateString('zh-CN')}</div>
                        {isExpired ? (
                            <Tag color="red" variant={'filled'}>已过期</Tag>
                        ) : daysLeft <= 7 ? (
                            <Tag color="orange" variant={'filled'}>{daysLeft}天后到期</Tag>
                        ) : daysLeft <= 30 ? (
                            <Tag color="gold" variant={'filled'}>{daysLeft}天后到期</Tag>
                        ) : null}
                    </div>
                );
            },
        },
        {
            title: '流量统计',
            key: 'trafficStats',
            hideInSearch: true,
            width: 120,
            render: (_, record) => {
                const trafficStats = record.trafficStats;
                if (!trafficStats || !trafficStats.enabled) {
                    return <Tag variant={'filled'}>未启用</Tag>;
                }
                return (
                    <div className="flex flex-col gap-1">
                        <Tag color="green" variant={'filled'}>已启用</Tag>
                        {trafficStats.limit > 0 && (
                            <span className="text-xs text-gray-500">
                                {(trafficStats.used / (1024 * 1024 * 1024)).toFixed(2)}GB / {(trafficStats.limit / (1024 * 1024 * 1024)).toFixed(0)}GB
                            </span>
                        )}
                    </div>
                );
            },
        },
        {
            title: '防篡改保护',
            key: 'tamperProtect',
            hideInSearch: true,
            width: 120,
            render: (_, record) => {
                const config = record.tamperProtectConfig;
                if (!config || !config.enabled) {
                    return <Tag variant={'filled'}>未启用</Tag>;
                }
                return (
                    <div className="flex flex-col gap-1">
                        <Tag color="green" variant={'filled'}>已启用</Tag>
                        {config.paths && config.paths.length > 0 && (
                            <span className="text-xs text-gray-500">{config.paths.length} 个路径</span>
                        )}
                    </div>
                );
            },
        },
        {
            title: 'SSH登录监控',
            key: 'sshLogin',
            hideInSearch: true,
            width: 120,
            render: (_, record) => {
                const config = record.sshLoginConfig;
                if (!config || !config.enabled) {
                    return <Tag variant={'filled'}>未启用</Tag>;
                }
                return <Tag color="green" variant={'filled'}>已启用</Tag>;
            },
        },
        {
            title: '排序权重',
            dataIndex: 'weight',
            key: 'weight',
            hideInSearch: true,
            sorter: true,
        },
        {
            title: '最后活跃时间',
            dataIndex: 'lastSeenAt',
            key: 'lastSeenAt',
            hideInSearch: true,
            valueType: 'dateTime',
            width: 180,
        },
        {
            title: '操作',
            key: 'action',
            valueType: 'option',
            width: 150,
            fixed: 'right',
            render: (_, record) => {
                const menuItems: MenuProps['items'] = [
                    {
                        key: 'view',
                        label: '查看详情',
                        icon: <Eye size={14}/>,
                        onClick: () => navigate(`/admin/agents/${record.id}`),
                    },
                    {
                        key: 'audit',
                        label: '安全审计',
                        icon: <Shield size={14}/>,
                        onClick: () => navigate(`/admin/agents/${record.id}?tab=audit`),
                    },
                    {
                        key: 'edit',
                        label: '编辑信息',
                        icon: <Edit size={14}/>,
                        onClick: () => handleEdit(record),
                    },
                    {
                        type: 'divider',
                    },
                    {
                        key: 'delete',
                        label: '删除探针',
                        icon: <Trash2 size={14}/>,
                        danger: true,
                        onClick: () => handleDelete(record),
                    },
                ];

                return (
                    <Space size="small">
                        <Button
                            type="link"
                            icon={<Eye size={14}/>}
                            onClick={() => navigate(`/admin/agents/${record.id}`)}
                            style={{padding: 0}}
                        >
                            详情
                        </Button>
                        <Dropdown menu={{items: menuItems}} trigger={['click']}>
                            <Button
                                type="link"
                                icon={<MoreVertical size={14}/>}
                                style={{padding: 0}}
                            />
                        </Dropdown>
                    </Space>
                );
            },
        },
    ];

    return (
        <div className="space-y-6">
            {/* 页面头部 */}
            <PageHeader
                title="探针管理"
                description="管理和监控系统探针状态"
                actions={[
                    {
                        key: 'batch-tags',
                        label: `批量操作标签${selectedRowKeys.length > 0 ? ` (${selectedRowKeys.length})` : ''}`,
                        icon: <Tags size={16}/>,
                        onClick: handleBatchTags,
                        disabled: selectedRowKeys.length === 0,
                    },
                    {
                        key: 'register',
                        label: '注册探针',
                        icon: <Plus size={16}/>,
                        onClick: () => navigate('/admin/agents-install'),
                        type: 'primary',
                    },
                    {
                        key: 'refresh',
                        label: '刷新',
                        icon: <RefreshCw size={16}/>,
                        onClick: () => actionRef.current?.reload(),
                    },
                ]}
            />

            <Divider/>

            {/* 探针列表 */}
            <ProTable<Agent>
                actionRef={actionRef}
                rowKey="id"
                search={{labelWidth: 80}}
                columns={columns}
                scroll={{x: 'max-content'}}
                pagination={{
                    defaultPageSize: 10,
                    showSizeChanger: true,
                }}
                options={false}
                rowSelection={{
                    selectedRowKeys,
                    onChange: (keys) => setSelectedRowKeys(keys),
                    preserveSelectedRowKeys: true,
                }}
                tableAlertRender={({selectedRowKeys}) => (
                    <Space size={16}>
                        <span>已选择 <strong>{selectedRowKeys.length}</strong> 项</span>
                    </Space>
                )}
                tableAlertOptionRender={() => (
                    <Space size={16}>
                        <a onClick={() => setSelectedRowKeys([])}>取消选择</a>
                    </Space>
                )}
                request={async (params) => {
                    const {current = 1, pageSize = 10, name, hostname, ip, status} = params;
                    try {
                        const response = await getAgentPaging(
                            current,
                            pageSize,
                            name,
                            hostname,
                            ip,
                            status as string | undefined
                        );
                        const items = response.data.items || [];
                        return {
                            data: items,
                            success: true,
                            total: response.data.total,
                        };
                    } catch (error: unknown) {
                        messageApi.error(getErrorMessage(error, '获取探针列表失败'));
                        return {
                            data: [],
                            success: false,
                        };
                    }
                }}
            />

            {/* 编辑探针信息模态框 */}
            <Modal
                title="编辑探针信息"
                open={editModalVisible}
                onOk={handleSave}
                onCancel={() => setEditModalVisible(false)}
                confirmLoading={loading}
                width={600}
            >
                <Form form={form} layout="vertical">
                    <Form.Item
                        label="名称"
                        name="name"
                        rules={[{required: true, message: '请输入探针名称'}]}
                    >
                        <Input placeholder="请输入探针名称"/>
                    </Form.Item>
                    <Form.Item
                        label="标签"
                        name="tags"
                        extra="可以从已有标签中选择，或输入新标签后按回车添加"
                    >
                        <Select
                            mode="tags"
                            placeholder="请选择或输入标签"
                            options={existingTags?.map(tag => ({label: tag, value: tag}))}
                            tokenSeparators={[',']}
                        />
                    </Form.Item>
                    <Form.Item
                        label="到期时间"
                        name="expireTime"
                    >
                        <DatePicker
                            style={{width: '100%'}}
                            format="YYYY-MM-DD"
                            placeholder="请选择到期时间"
                        />
                    </Form.Item>
                    <Form.Item
                        label="可见性"
                        name="visibility"
                        rules={[{required: true, message: '请选择可见性'}]}
                        extra="控制探针在公开页面的可见性"
                    >
                        <Select
                            placeholder="请选择可见性"
                            options={[
                                {label: '匿名可见', value: 'public'},
                                {label: '登录可见', value: 'private'},
                            ]}
                        />
                    </Form.Item>
                    <Form.Item
                        label="权重排序"
                        name="weight"
                        extra="数字越大排序越靠前，默认为0"
                    >
                        <InputNumber
                            min={0}
                            step={1}
                            precision={0}
                            placeholder="请输入权重"
                            style={{width: '100%'}}
                        />
                    </Form.Item>
                    <Form.Item
                        label="备注"
                        name="remark"
                        extra="备注信息"
                    >
                        <Input.TextArea
                            rows={3}
                            placeholder="请输入备注信息"
                            maxLength={500}
                            showCount
                        />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 批量操作标签模态框 */}
            <Modal
                title={`批量操作标签 (已选择 ${selectedRowKeys.length} 个探针)`}
                open={batchTagModalVisible}
                onOk={handleBatchSave}
                onCancel={() => setBatchTagModalVisible(false)}
                confirmLoading={batchLoading}
                width={600}
            >
                <Form form={batchForm} layout="vertical">
                    <Form.Item
                        label="操作类型"
                        name="operation"
                        rules={[{required: true, message: '请选择操作类型'}]}
                    >
                        <Radio.Group>
                            <Radio value="add">添加标签（保留原有标签）</Radio>
                            <Radio value="remove">移除标签（从原有标签中移除）</Radio>
                            <Radio value="replace">替换标签（完全替换为新标签）</Radio>
                        </Radio.Group>
                    </Form.Item>
                    <Form.Item
                        label="标签"
                        name="tags"
                        rules={[{required: true, message: '请输入或选择标签'}]}
                        extra="可以从已有标签中选择，或输入新标签后按回车添加"
                    >
                        <Select
                            mode="tags"
                            placeholder="请选择或输入标签"
                            options={existingTags?.map(tag => ({label: tag, value: tag}))}
                            tokenSeparators={[',']}
                        />
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
};

export default AgentList;
