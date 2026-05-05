import {Tabs} from 'antd';
import {Bell, MessageSquare, Settings2, Wifi} from 'lucide-react';
import AlertSettings from './AlertSettings';
import NotificationChannels from './NotificationChannels';
import SystemConfig from './SystemConfig';
import PublicIPConfig from './PublicIPConfig';
import {PageHeader} from "@admin/components";
import {useSearchParams} from "react-router-dom";

// 默认 IPv4 API 列表
export const defaultIPv4APIs = [
    'https://myip.ipip.net',
    'https://ddns.oray.com/checkip',
    'https://ip.3322.net',
    'https://4.ipw.cn',
    'https://v4.yinghualuo.cn/bejson',
];

// 默认 IPv6 API 列表
export const defaultIPv6APIs = [
    'https://speed.neu6.edu.cn/getIP.php',
    'https://v6.ident.me',
    'https://6.ipw.cn',
    'https://v6.yinghualuo.cn/bejson',
];

const Settings = () => {

    const [searchParams, setSearchParams] = useSearchParams({tab: 'system'});

    const items = [
        {
            key: 'system',
            label: (
                <span className="flex items-center gap-2">
                    <Settings2 size={16}/>
                    系统配置
                </span>
            ),
            children: <SystemConfig/>,
        },
        {
            key: 'channels',
            label: (
                <span className="flex items-center gap-2">
                    <MessageSquare size={16}/>
                    通知渠道
                </span>
            ),
            children: <NotificationChannels/>,
        },
        {
            key: 'public-ip',
            label: (
                <span className="flex items-center gap-2">
                    <Wifi size={16}/>
                    公网 IP 采集
                </span>
            ),
            children: (
                <PublicIPConfig
                    defaultIPv4APIs={defaultIPv4APIs}
                    defaultIPv6APIs={defaultIPv6APIs}
                />
            ),
        },
        {
            key: 'alert',
            label: (
                <span className="flex items-center gap-2">
                    <Bell size={16}/>
                    告警规则
                </span>
            ),
            children: <AlertSettings/>,
        },
    ];

    return (
        <div className={'space-y-6'}>
            <PageHeader
                title="系统设置"
                description="CONFIGURATION"
            />
            <Tabs tabPlacement={'start'}
                  items={items}
                  activeKey={searchParams.get('tab')}
                  onChange={(key) => {
                      setSearchParams({tab: key});
                  }}
            />
        </div>
    );
};

export default Settings;
