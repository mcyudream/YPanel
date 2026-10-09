// Docker 资源用量统计（system df）。
package dockerx

import (
	"context"
	"time"

	"github.com/moby/moby/client"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Usage 采集 Docker 资源用量：容器/镜像/卷/构建缓存的大小与数量、明细、网络数、主机端口数。
// system df verbose 由 daemon 实算 size，镜像多时耗时明显，调用方应懒加载、勿进高频轮询。
func (m *Manager) Usage(ctx context.Context) (*dto.DockerUsage, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	du, err := cli.DiskUsage(ctx, client.DiskUsageOptions{
		Containers: true,
		Images:     true,
		Volumes:    true,
		BuildCache: true,
		Verbose:    true,
	})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "docker df: "+err.Error())
	}
	out := &dto.DockerUsage{
		ImagesTotalSize:  du.Images.TotalSize,
		ImagesCount:      int(du.Images.TotalCount),
		ContainersRWSize: du.Containers.TotalSize,
		ContainersCount:  int(du.Containers.TotalCount),
		VolumesTotalSize: du.Volumes.TotalSize,
		VolumesCount:     int(du.Volumes.TotalCount),
		BuildCacheSize:   du.BuildCache.TotalSize,
		BuildCacheCount:  int(du.BuildCache.TotalCount),
		ContainerItems:   make([]dto.DockerUsageItem, 0, len(du.Containers.Items)),
		ImageItems:       make([]dto.DockerUsageItem, 0, len(du.Images.Items)),
		VolumeItems:      make([]dto.DockerUsageItem, 0, len(du.Volumes.Items)),
		CollectedAt:      time.Now(),
	}
	for _, c := range du.Containers.Items {
		out.ContainerItems = append(out.ContainerItems, dto.DockerUsageItem{
			Name: containerName(c.Names),
			Size: c.SizeRw,
			Sub:  string(c.State),
		})
	}
	for _, img := range du.Images.Items {
		tag := "<none>:<none>"
		if len(img.RepoTags) > 0 {
			tag = img.RepoTags[0]
		}
		out.ImageItems = append(out.ImageItems, dto.DockerUsageItem{
			Name: tag,
			Size: img.Size,
			Sub:  shortID(img.ID),
		})
	}
	for _, v := range du.Volumes.Items {
		size := int64(0)
		if v.UsageData != nil {
			size = v.UsageData.Size
		}
		out.VolumeItems = append(out.VolumeItems, dto.DockerUsageItem{
			Name: v.Name,
			Size: size,
		})
	}
	// 网络数与主机端口数（去重，tcp/udp 分开计）；失败不阻塞主数据
	if nets, nerr := m.NetworkList(ctx); nerr == nil {
		out.NetworksCount = len(nets)
	}
	if containers, lerr := m.List(ctx); lerr == nil {
		seen := map[string]bool{}
		for _, c := range containers {
			for _, p := range c.Ports {
				if p.HostPort == "" {
					continue
				}
				key := p.HostPort + "/" + p.Proto
				if !seen[key] {
					seen[key] = true
					out.HostPortsCount++
				}
			}
		}
	}
	return out, nil
}

// BuildCachePrune 清空全部未使用的构建缓存，返回释放的字节数。
func (m *Manager) BuildCachePrune(ctx context.Context) (int64, error) {
	cli, err := m.getClient()
	if err != nil {
		return 0, err
	}
	res, err := cli.BuildCachePrune(ctx, client.BuildCachePruneOptions{All: true})
	if err != nil {
		return 0, errs.Wrapc(errs.CodeFileOpFailed, "buildcache prune: "+err.Error())
	}
	return int64(res.Report.SpaceReclaimed), nil
}
