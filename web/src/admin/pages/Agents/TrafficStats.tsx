import React, {useEffect, useState} from 'react';
import {
    Alert,
    App,
    Button,
    Card,
    Descriptions,
    Form,
    InputNumber,
    Progress,
    Select,
    Space,
    Statistic,
    Switch,
    Tag
} from 'antd';
import {Activity, RotateCcw, Save} from 'lucide-react';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {getTrafficStats, resetAgentTraffic, updateTrafficConfig} from '@/api/agent';
import {getErrorMessage} from '@/lib/utils';
import type {TrafficStats as TrafficStatsType, UpdateTrafficConfigRequest} from '@/types';
import dayjs from 'dayjs';

interface TrafficStatsProps {
    agentId: string;
}

const TrafficStats: React.FC<TrafficStatsProps> = ({agentId}) => {
    const {message, modal} = App.useApp();
    const [form] = Form.useForm();
    const queryClient = useQueryClient();
    const [enabled, setEnabled] = useState(false);

    // 获取流量统计
    const {data: stats, isLoading} = useQuery<TrafficStatsType>({
        queryKey: ['trafficStats', agentId],
        queryFn: async () => {
            const response = await getTrafficStats(agentId);
            return response.data;
        },
        enabled: agentId !== '',
    });

    // 保存配置 mutation
    const saveMutation = useMutation({
        mutationFn: (data: UpdateTrafficConfigRequest) => updateTrafficConfig(agentId, data),
        onSuccess: () => {
            message.success('配置已保存');
            queryClient.invalidateQueries({queryKey: ['trafficStats', agentId]});
        },
        onError: (error: unknown) => {
            console.error('Failed to save config:', error);
            message.error(getErrorMessage(error, '配置保存失败'));
        },
    });

    const handleSave = async () => {
        try {
            const values = enabled ? await form.validateFields() : form.getFieldsValue();
            const limitValue = typeof values.trafficLimit === 'number' ? values.trafficLimit : 0;
            const limitBytes = enabled ? limitValue * 1024 * 1024 * 1024 : 0;
            saveMutation.mutate({
                enabled: enabled,
                type: values.trafficType || 'recv',
                limit: limitBytes,
                resetDay: enabled ? (values.trafficResetDay || 0) : 0,
            });
        } catch (error) {
            // 表单验证失败
        }
    };

    // 重置流量 mutation
    const resetMutation = useMutation({
        mutationFn: () => resetAgentTraffic(agentId),
        onSuccess: () => {
            message.success('流量已重置');
            queryClient.invalidateQueries({queryKey: ['trafficStats', agentId]});
        },
        onError: (error: unknown) => {
            console.error('Failed to reset traffic:', error);
            message.error(getErrorMessage(error, '重置流量失败'));
        },
    });

    // 手动重置流量
    const handleResetTraffic = () => {
        modal.confirm({
            title: '确认重置流量',
            content: '确定要立即重置流量统计吗？此操作将清空当前周期的流量使用记录。',
            okText: '确认重置',
            cancelText: '取消',
            okButtonProps: {danger: true},
            centered: true,
            onOk: () => resetMutation.mutate(),
        });
    };

    // 初始化表单值
    useEffect(() => {
        if (stats) {
            // 从服务器返回的 enabled 字段读取启用状态
            setEnabled(stats.enabled);
            form.setFieldsValue({
                trafficType: stats.type || 'recv',
                trafficLimit: stats.limit > 0 ? stats.limit / (1024 * 1024 * 1024) : 0,
                trafficUsed: stats.used > 0 ? stats.used / (1024 * 1024 * 1024) : 0,
                trafficResetDay: stats.resetDay || 0,
            });
        } else {
            setEnabled(false);
            form.setFieldsValue({
                trafficType: 'recv',
                trafficLimit: 0,
                trafficUsed: 0,
                trafficResetDay: 0,
            });
        }
    }, [stats, form]);

    // 计算流量使用百分比
    const usagePercent = stats && stats.limit > 0
        ? Math.min(100, (stats.used / stats.limit) * 100)
        : 0;

    // 获取进度条颜色
    const getProgressColor = (percent: number) => {
        if (percent >= 100) return '#ff4d4f';
        if (percent >= 90) return '#ff7a45';
        if (percent >= 80) return '#ffa940';
        return '#52c41a';
    };

    // 格式化字节数
    const formatBytes = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
    };

    return (
        <Space orientation="vertical" size="large" style={{width: '100%'}}>
            <Alert
                title="流量统计功能可以设置流量限额并自动监控使用情况，超过阈值时会发送告警通知。"
                type="info"
                showIcon
                icon={<Activity size={16}/>}
            />

            {/* 当前流量使用情况 */}
            {stats && stats.enabled && stats.limit > 0 && (
                <Card
                    title="流量使用情况"
                    variant="outlined"
                >
                    <Space orientation="vertical" size="large" style={{width: '100%'}}>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <Statistic
                                title="已使用流量"
                                value={formatBytes(stats.used)}
                                styles={{
                                    content: {color: getProgressColor(usagePercent)}
                                }}
                            />
                            <Statistic
                                title="流量限额"
                                value={formatBytes(stats.limit)}
                            />
                            <Statistic
                                title="剩余流量"
                                value={formatBytes(stats.remaining)}
                                styles={{
                                    content: {color: stats.remaining > 0 ? '#52c41a' : '#ff4d4f'}
                                }}
                            />
                        </div>

                        <div>
                            <div className="mb-2 flex justify-between">
                                <span>使用率</span>
                                <span className="font-medium">{usagePercent.toFixed(2)}%</span>
                            </div>
                            <Progress
                                percent={usagePercent}
                                strokeColor={getProgressColor(usagePercent)}
                                showInfo={false}
                            />
                        </div>

                        <Descriptions bordered column={1} size="small">
                            <Descriptions.Item label="重置日期">
                                {stats.resetDay > 0 ? `每月 ${stats.resetDay} 号` : '不自动重置'}
                            </Descriptions.Item>
                            <Descriptions.Item label="当前周期">
                                {dayjs(stats.periodStart).format('YYYY-MM-DD')} ~ {dayjs(stats.periodEnd).format('YYYY-MM-DD')}
                            </Descriptions.Item>
                            <Descriptions.Item label="距离重置">
                                {stats.daysUntilReset > 0 ? `${stats.daysUntilReset} 天` : '今日重置'}
                            </Descriptions.Item>
                            <Descriptions.Item label="告警状态">
                                <Space>
                                    {stats.alerts.sent80 && <Tag color="orange">80%告警已发送</Tag>}
                                    {stats.alerts.sent90 && <Tag color="orange">90%告警已发送</Tag>}
                                    {stats.alerts.sent100 && <Tag color="red">100%告警已发送</Tag>}
                                    {!stats.alerts.sent80 && !stats.alerts.sent90 && !stats.alerts.sent100 && (
                                        <Tag color="green">正常</Tag>
                                    )}
                                </Space>
                            </Descriptions.Item>
                        </Descriptions>

                        <Button
                            icon={<RotateCcw size={16}/>}
                            onClick={handleResetTraffic}
                            loading={resetMutation.isPending}
                            danger
                        >
                            立即重置流量
                        </Button>
                    </Space>
                </Card>
            )}

            {/* 流量配置 */}
            <Card
                title={
                    <div className="flex items-center gap-2">
                        <Activity size={18}/>
                        <span>流量统计配置</span>
                    </div>
                }
                extra={
                    <Button
                        type="primary"
                        icon={<Save size={16}/>}
                        onClick={handleSave}
                        loading={saveMutation.isPending}
                    >
                        保存配置
                    </Button>
                }
                loading={isLoading}
            >
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{
                        trafficType: 'recv',
                        trafficLimit: 0,
                        trafficUsed: 0,
                        trafficResetDay: 0,
                    }}
                >
                    <Form.Item
                        label="启用流量统计"
                        extra="开启后将监控探针的流量使用情况"
                    >
                        <Switch
                            checked={enabled}
                            onChange={setEnabled}
                            checkedChildren="已启用"
                            unCheckedChildren="已禁用"
                        />
                    </Form.Item>

                    {enabled && (
                        <>
                            <Form.Item
                                label="统计类型"
                                name="trafficType"
                                rules={[{required: true, message: '请选择统计类型'}]}
                                extra="选择要统计的流量类型"
                            >
                                <Select
                                    placeholder="请选择统计类型"
                                    options={[
                                        {label: '进站流量 (下载)', value: 'recv'},
                                        {label: '出站流量 (上传)', value: 'send'},
                                        {label: '全部流量 (上传+下载)', value: 'both'},
                                    ]}
                                />
                            </Form.Item>

                            <Form.Item
                                label="流量限额"
                                name="trafficLimit"
                                rules={[
                                    {required: true, message: '请输入流量限额'},
                                    {type: 'number', min: 1, message: '流量限额必须大于0'},
                                ]}
                                extra="设置流量限额(GB)，必须大于0"
                            >
                                <InputNumber
                                    min={1}
                                    step={1}
                                    precision={0}
                                    placeholder="请输入流量限额(GB)"
                                    style={{width: '100%'}}
                                    addonAfter="GB"
                                />
                            </Form.Item>

                            <Form.Item
                                label="已使用流量"
                                name="trafficUsed"
                                extra="当前已使用的流量（只读）"
                            >
                                <InputNumber
                                    min={0}
                                    step={0.1}
                                    precision={2}
                                    placeholder="已使用流量(GB)"
                                    style={{width: '100%'}}
                                    addonAfter="GB"
                                    disabled
                                />
                            </Form.Item>

                            <Form.Item
                                label="流量重置日期"
                                name="trafficResetDay"
                                rules={[{required: true, message: '请选择流量重置日期'}]}
                                extra="每月的几号重置流量，0表示不自动重置"
                            >
                                <Select
                                    placeholder="请选择流量重置日期"
                                    options={[
                                        {label: '不自动重置', value: 0},
                                        ...Array.from({length: 31}, (_, i) => ({
                                            label: `每月${i + 1}号`,
                                            value: i + 1,
                                        })),
                                    ]}
                                />
                            </Form.Item>
                        </>
                    )}
                </Form>
            </Card>
        </Space>
    );
};

export default TrafficStats;
