package websocket

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"zjsj/internal/pkg/curl"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/google/uuid"
)

// 分配状态
const (
	AllocStatusCreating  = "creating"  // 创建中，还没有 ip
	AllocStatusAllocated = "allocated" // 创建成功，已分配 ip
	AllocStatusFailed    = "failed"    // 创建失败
)

// 轮询参数：每 2 秒查一次，最长等 5 分钟
const (
	allocPollInterval = 2 * time.Second
	allocPollTimeout  = 5 * time.Minute
)

type GameServerPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
}

type Allocation struct {
	Uuid            string            `json:"uuid"`
	GameId          string            `json:"gameId"`
	RoomId          string            `json:"roomId"` // 归属房间，一个房间对应一个云服务器
	Ip              string            `json:"ip"`
	GameServerPorts []*GameServerPort `json:"gameServerPorts"`
	CreatedAt       string            `json:"createdAt"`
	Status          string            `json:"status"`
	Msg             string            `json:"msg"`
}

// IsReady 有 ip 且状态为 allocated 才算创建成功
func (a *Allocation) IsReady() bool {
	return a != nil && a.Ip != "" && a.Status == AllocStatusAllocated
}

type UnityCloudConfig struct {
	AppId           string `json:"appId"`
	AppSecret       string `json:"appSecret"`
	AppServerSecret string `json:"appServerSecret"`
	ProfileId       string `json:"profileId"`
	RegionId        string `json:"regionId"`
	Uuid            string `json:"uuid"`
	AllocationTTL   string `json:"allocationTTL"`
}

// UnityService 管理多个云服务器分配（uuid -> Allocation），一个房间一个服务器
type UnityService struct {
	mu          sync.RWMutex
	allocations map[string]*Allocation
	onReady     func(*Allocation) // 创建成功钩子（轮询/回调触发）
	onFailed    func(*Allocation) // 创建失败钩子
}

var unityCloudConfig = &UnityCloudConfig{}

// 分配信息落盘路径：进程重启后用于清理上次遗留的云服务器
var allocationCachePath = gfile.Join(gfile.Pwd(), "storage/config", "allocations.json")

func NewUnityService() *UnityService {
	err := g.Cfg().MustGet(context.TODO(), "unityCloud").Scan(unityCloudConfig)
	if err != nil {
		g.Log("test").Async().Errorf(context.TODO(), "unityCloud 配置错误: %v", err)
		return nil
	}
	if unityCloudConfig.AllocationTTL == "" {
		unityCloudConfig.AllocationTTL = "20m"
	}

	s := &UnityService{allocations: make(map[string]*Allocation)}

	// 房间都在内存里，重启后房间已丢失，上次遗留的云服务器直接释放，避免泄漏计费
	go s.cleanupPersisted()
	return s
}

// SetHooks 注册创建成功/失败回调（由 Manager 注册，用于绑定房间）
func (s *UnityService) SetHooks(onReady, onFailed func(*Allocation)) {
	s.onReady = onReady
	s.onFailed = onFailed
}

// CreateAllocation 异步创建云服务器：立即返回分配信息（ip 为空表示创建中），
// 后台轮询直到 allocated/failed，结果通过钩子通知。每次创建生成新的 uuid。
func (s *UnityService) CreateAllocation(ctx context.Context, roomId string, envs map[string]string) (*Allocation, error) {
	id := uuid.New().String()

	if envs == nil {
		envs = make(map[string]string)
	}
	// 传给云服务器进程，便于其回连时识别归属房间
	envs["roomId"] = roomId
	envs["uuid"] = id

	url := fmt.Sprintf("https://multiverse.scaling.unity.cn/v1/allocation/games/%s/allocations", unityCloudConfig.AppId)
	data := map[string]any{
		"uuid":          id,
		"profileId":     unityCloudConfig.ProfileId,
		"regionId":      unityCloudConfig.RegionId,
		"allocationTTL": unityCloudConfig.AllocationTTL,
		"envs":          envs,
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": getBasicAuthorization(unityCloudConfig.AppId, unityCloudConfig.AppServerSecret),
	}

	v, err := curl.NewHttpClient().Post(ctx, url, data, headers)
	if err != nil {
		return nil, gerror.Wrapf(err, "创建 Allocation 失败")
	}

	type Result struct {
		Allocation *Allocation `json:"allocation"`
	}
	var result Result
	if err = gjson.Unmarshal([]byte(v), &result); err != nil {
		return nil, gerror.Wrapf(err, "解析 Allocation 失败")
	}
	if result.Allocation == nil {
		return nil, gerror.New("创建 Allocation 返回为空")
	}

	alloc := result.Allocation
	alloc.Uuid = id
	alloc.RoomId = roomId
	switch {
	case alloc.Status == AllocStatusFailed:
		// 平台直接返回失败
	case alloc.Ip != "":
		alloc.Status = AllocStatusAllocated
	default:
		// 没有 ip，创建中，交给后台轮询
		alloc.Status = AllocStatusCreating
	}

	s.mu.Lock()
	s.allocations[id] = alloc
	s.mu.Unlock()
	s.persist()

	if alloc.Status == AllocStatusCreating {
		g.Log("test").Async().Infof(ctx, "云服务器创建中, uuid=%s, roomId=%s", id, roomId)
		go s.pollAllocation(id)
	} else {
		g.Log("test").Async().Infof(ctx, "云服务器创建返回, uuid=%s, roomId=%s, status=%s, ip=%s", id, roomId, alloc.Status, alloc.Ip)
	}

	cp := *alloc
	return &cp, nil
}

// GetAllocation 按 uuid 查询内存中的分配信息
func (s *UnityService) GetAllocation(id string) *Allocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if a, ok := s.allocations[id]; ok {
		cp := *a
		return &cp
	}
	return nil
}

// GetRoomAllocation 按房间查询分配信息
func (s *UnityService) GetRoomAllocation(roomId string) *Allocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.allocations {
		if a.RoomId == roomId {
			cp := *a
			return &cp
		}
	}
	return nil
}

// HasCreatingAllocation 是否还有创建中的云服务器（孤儿清理据此允许未绑定服务器等待补绑定）
func (s *UnityService) HasCreatingAllocation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.allocations {
		if a.Status == AllocStatusCreating {
			return true
		}
	}
	return false
}

// ListAllocations 返回全部分配信息
func (s *UnityService) ListAllocations() []*Allocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Allocation, 0, len(s.allocations))
	for _, a := range s.allocations {
		cp := *a
		list = append(list, &cp)
	}
	return list
}

// HandleCallback 平台回调入口（可选，轮询兜底），与轮询共用同一套幂等完成逻辑
func (s *UnityService) HandleCallback(id, ip string, ports []string, status, msg string) {
	switch status {
	case AllocStatusAllocated:
		s.finishAllocation(id, ip, parseCallbackPorts(ports))
	case AllocStatusFailed:
		s.failAllocation(id, msg)
	default:
		g.Log("test").Async().Warningf(context.TODO(), "未知的分配回调状态: %s, uuid=%s", status, id)
	}
}

// DeleteAllocation 删除分配（释放云服务器），并移除本地记录
func (s *UnityService) DeleteAllocation(ctx context.Context, id string) {
	s.mu.Lock()
	_, ok := s.allocations[id]
	delete(s.allocations, id)
	s.mu.Unlock()
	if !ok {
		return
	}
	s.persist()

	if err := s.deleteRemote(ctx, id); err != nil {
		g.Log("test").Async().Errorf(ctx, "删除 Allocation 失败, uuid=%s: %v", id, err)
	} else {
		g.Log("test").Async().Infof(ctx, "已删除 Allocation, uuid=%s", id)
	}
}

// pollAllocation 后台轮询创建结果，使用独立 context，不受请求断开影响
func (s *UnityService) pollAllocation(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), allocPollTimeout)
	defer cancel()

	ticker := time.NewTicker(allocPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.Log("test").Async().Errorf(ctx, "轮询 Allocation 超时, uuid=%s", id)
			s.failAllocation(id, "轮询超时，云服务器创建失败")
			// 超时后主动释放，避免平台继续计费
			delCtx, delCancel := context.WithTimeout(context.Background(), 10*time.Second)
			s.DeleteAllocation(delCtx, id)
			delCancel()
			return
		case <-ticker.C:
			alloc, err := s.fetchAllocation(ctx, id)
			if err != nil {
				g.Log("test").Async().Infof(ctx, "查询 Allocation 失败, uuid=%s: %v", id, err)
				continue
			}
			if alloc == nil {
				continue
			}
			if alloc.Status == AllocStatusFailed {
				s.failAllocation(id, alloc.Msg)
				return
			}
			if alloc.Ip != "" {
				s.finishAllocation(id, alloc.Ip, alloc.GameServerPorts)
				return
			}
			// ip 还是空，继续等
		}
	}
}

// finishAllocation 标记创建成功（幂等：轮询和回调只会生效一次）
func (s *UnityService) finishAllocation(id, ip string, ports []*GameServerPort) {
	s.mu.Lock()
	a, ok := s.allocations[id]
	if !ok || a.Status == AllocStatusAllocated || a.Status == AllocStatusFailed {
		s.mu.Unlock()
		return
	}
	a.Ip = ip
	a.GameServerPorts = ports
	a.Status = AllocStatusAllocated
	cp := *a
	s.mu.Unlock()
	s.persist()

	g.Log("test").Async().Infof(context.TODO(), "云服务器创建成功, uuid=%s, roomId=%s, ip=%s", id, cp.RoomId, ip)
	if s.onReady != nil {
		s.onReady(&cp)
	}
}

// failAllocation 标记创建失败（幂等）
func (s *UnityService) failAllocation(id, msg string) {
	s.mu.Lock()
	a, ok := s.allocations[id]
	if !ok || a.Status == AllocStatusAllocated || a.Status == AllocStatusFailed {
		s.mu.Unlock()
		return
	}
	a.Status = AllocStatusFailed
	a.Msg = msg
	cp := *a
	s.mu.Unlock()
	s.persist()

	g.Log("test").Async().Errorf(context.TODO(), "云服务器创建失败, uuid=%s, roomId=%s, msg=%s", id, cp.RoomId, msg)
	if s.onFailed != nil {
		s.onFailed(&cp)
	}
}

// fetchAllocation 查询平台分配详情（ip 为空也返回，便于识别 failed 状态）
func (s *UnityService) fetchAllocation(ctx context.Context, id string) (*Allocation, error) {
	url := fmt.Sprintf("https://multiverse.scaling.unity.cn/v1/allocation/games/%s/allocations/%s", unityCloudConfig.AppId, id)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": getBasicAuthorization(unityCloudConfig.AppId, unityCloudConfig.AppServerSecret),
	}

	v, err := curl.NewHttpClient().Get(ctx, url, nil, headers)
	if err != nil {
		return nil, err
	}

	type Result struct {
		Allocation *Allocation `json:"allocation"`
	}
	var result Result
	if err = gjson.Unmarshal([]byte(v), &result); err != nil {
		return nil, gerror.Wrapf(err, "Get Allocation 数据解析失败")
	}
	return result.Allocation, nil
}

// deleteRemote 调平台 DELETE 接口释放服务器
func (s *UnityService) deleteRemote(ctx context.Context, id string) error {
	url := fmt.Sprintf("https://multiverse.scaling.unity.cn/v1/allocation/games/%s/allocations/%s", unityCloudConfig.AppId, id)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": getBasicAuthorization(unityCloudConfig.AppId, unityCloudConfig.AppServerSecret),
	}
	_, err := curl.NewHttpClient().Delete(ctx, url, nil, headers)
	return err
}

// persist 分配信息落盘，重启后用于清理遗留服务器
func (s *UnityService) persist() {
	list := s.ListAllocations()
	if err := gfile.PutContents(allocationCachePath, gjson.MustEncodeString(list)); err != nil {
		g.Log("test").Async().Errorf(context.TODO(), "写入分配缓存失败: %v", err)
	}
}

// cleanupPersisted 启动时清理上次进程遗留的云服务器
func (s *UnityService) cleanupPersisted() {
	content := gfile.GetContents(allocationCachePath)
	if content == "" {
		return
	}
	var list []*Allocation
	if err := gjson.Unmarshal([]byte(content), &list); err != nil {
		g.Log("test").Async().Errorf(context.TODO(), "解析分配缓存失败: %v", err)
		return
	}
	for _, a := range list {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.deleteRemote(ctx, a.Uuid); err != nil {
			g.Log("test").Async().Errorf(ctx, "清理遗留云服务器失败, uuid=%s: %v", a.Uuid, err)
		} else {
			g.Log("test").Async().Infof(ctx, "已清理遗留云服务器, uuid=%s", a.Uuid)
		}
		cancel()
	}
	_ = gfile.PutContents(allocationCachePath, "[]")
}

// parseCallbackPorts 解析回调里的端口格式："7275/UDP/http"
func parseCallbackPorts(ports []string) []*GameServerPort {
	result := make([]*GameServerPort, 0, len(ports))
	for _, p := range ports {
		parts := strings.Split(p, "/")
		if len(parts) != 3 {
			continue
		}
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		result = append(result, &GameServerPort{Port: port, Protocol: parts[1], Name: parts[2]})
	}
	return result
}

func getBasicAuthorization(appID, appServiceSecret string) string {
	credentials := appID + ":" + appServiceSecret
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encodedCredentials
}
