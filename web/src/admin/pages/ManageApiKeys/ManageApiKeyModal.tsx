import {useEffect} from 'react';
import {App, Form, Input, Modal} from 'antd';
import {generateManageApiKey, getManageApiKey, updateManageApiKeyName} from '@/api/manageApiKey.ts';
import type {ApiKey, GenerateApiKeyRequest, UpdateApiKeyNameRequest} from '@/types';
import {getErrorMessage} from '@/lib/utils';

interface ManageApiKeyModalProps {
    open: boolean;
    apiKeyId?: string;
    onCancel: () => void;
    onSuccess: (apiKey?: ApiKey) => void;
}

const ManageApiKeyModal = ({open, apiKeyId, onCancel, onSuccess}: ManageApiKeyModalProps) => {
    const {message: messageApi} = App.useApp();
    const [form] = Form.useForm();
    const isEditMode = !!apiKeyId;

    useEffect(() => {
        if (open && apiKeyId) {
            const loadApiKey = async () => {
                try {
                    const response = await getManageApiKey(apiKeyId);
                    form.setFieldsValue({
                        name: response.data.name,
                    });
                } catch (error) {
                    messageApi.error(getErrorMessage(error, '加载 API 密钥详情失败'));
                }
            };
            loadApiKey();
        } else if (open) {
            form.resetFields();
        }
    }, [open, apiKeyId, form, messageApi]);

    const handleOk = async () => {
        try {
            const values = await form.validateFields();
            const name = values.name?.trim();

            if (!name) {
                messageApi.warning('名称不能为空');
                return;
            }

            if (isEditMode) {
                const updateData: UpdateApiKeyNameRequest = {name};
                await updateManageApiKeyName(apiKeyId, updateData);
                messageApi.success('更新成功');
                onSuccess();
            } else {
                const createData: GenerateApiKeyRequest = {name};
                const response = await generateManageApiKey(createData);
                messageApi.success('API密钥已生成');
                onSuccess(response.data);
            }
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'errorFields' in error) {
                return;
            }
            messageApi.error(getErrorMessage(error, '操作失败'));
        }
    };

    return (
        <Modal
            title={isEditMode ? '编辑API密钥' : '生成API密钥'}
            open={open}
            onOk={handleOk}
            onCancel={onCancel}
            okText={isEditMode ? '保存' : '生成'}
            cancelText="取消"
            destroyOnHidden
        >
            <Form form={form} layout="vertical" autoComplete="off">
                <Form.Item
                    label="密钥名称"
                    name="name"
                    rules={[
                        {required: true, message: '请输入密钥名称'},
                        {min: 2, message: '密钥名称至少2个字符'},
                    ]}
                >
                    <Input placeholder="例如: 生产环境、测试环境等"/>
                </Form.Item>
            </Form>
        </Modal>
    );
};

export default ManageApiKeyModal;
