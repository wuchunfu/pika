package collector

import (
	"strings"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/pkg/agent/config"
	"github.com/shirou/gopsutil/v4/disk"
)

// 不以 /dev/ 开头但仍是真实存储的文件系统类型
var realFSTypes = map[string]bool{
	"nfs":    true,
	"nfs4":   true,
	"cifs":   true,
	"smbfs":  true,
	"zfs":    true,
	"glusterfs": true,
	"cephfs": true,
}

// DiskCollector 磁盘监控采集器
type DiskCollector struct {
	config *config.Config
}

// NewDiskCollector 创建磁盘采集器
func NewDiskCollector(cfg *config.Config) *DiskCollector {
	return &DiskCollector{
		config: cfg,
	}
}

// isRealDisk 判断是否为真实存储设备
func isRealDisk(device, fstype string) bool {
	// Windows 盘符
	if len(device) == 2 && device[1] == ':' {
		return true
	}
	// Unix 块设备
	if strings.HasPrefix(device, "/dev/") {
		return true
	}
	// 网络/集群存储
	return realFSTypes[fstype]
}

// macOS APFS 容器内与数据卷共享物理空间的系统辅助卷
var macosSystemVolumes = map[string]bool{
	"Preboot":   true,
	"VM":        true,
	"Update":    true,
	"xarts":     true,
	"iSCPreboot": true,
	"Hardware":  true,
}

// isMacOSRedundant 判断是否为 macOS APFS 冗余系统卷
func isMacOSRedundant(mountPoint, fstype string) bool {
	if fstype != "apfs" {
		return false
	}
	const prefix = "/System/Volumes/"
	if !strings.HasPrefix(mountPoint, prefix) {
		return false
	}
	volumeName := strings.TrimPrefix(mountPoint, prefix)
	return macosSystemVolumes[volumeName]
}

// Collect 采集磁盘数据
// 只采集配置的 DiskInclude 白名单中的挂载点
func (d *DiskCollector) Collect() ([]protocol.DiskData, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var diskDataList []protocol.DiskData
	for _, partition := range partitions {
		// 跳过虚拟文件系统
		if !isRealDisk(partition.Device, partition.Fstype) {
			continue
		}

		// 跳过 macOS APFS 冗余系统卷（与 Data 共享物理空间）
		if isMacOSRedundant(partition.Mountpoint, partition.Fstype) {
			continue
		}

		// 检查是否在白名单中
		if !d.config.ShouldIncludeDiskMountPoint(partition.Mountpoint) {
			continue
		}

		// 获取动态使用情况
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue // 跳过无法访问的分区
		}

		// 跳过容量为 0 的分区
		if usage.Total == 0 {
			continue
		}

		diskDataList = append(diskDataList, protocol.DiskData{
			MountPoint:   partition.Mountpoint,
			Device:       partition.Device,
			Fstype:       partition.Fstype,
			Total:        usage.Total,
			Used:         usage.Used,
			Free:         usage.Free,
			UsagePercent: usage.UsedPercent,
		})
	}

	// 所有白名单分区均采集失败时，返回空列表表示异常
	if len(diskDataList) == 0 {
		return nil, nil
	}

	return diskDataList, nil
}
