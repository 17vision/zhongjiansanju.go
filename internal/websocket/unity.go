package websocket

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
	"uuid"
	"zjsj/internal/pkg/curl"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcache"
)

type GameServerPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
}

type Allocation struct {
	GameId          string            `json:"gameId"`
	Ip              string            `json:"ip"`
	GameServerPorts []*GameServerPort `json:"gameServerPorts"`
	CreatedAt       string            `json:"createdAt"`
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

type UnityService struct {
	mu     sync.RWMutex
	getIng bool
}

var unityCloudConfig = &UnityCloudConfig{}

var allocationKey = "unity:service:allocation"

func NewUnityService() *UnityService {
	err := g.Cfg().MustGet(context.TODO(), "unityCloud").Scan(unityCloudConfig)
	if err != nil {
		gerror.Wrap(err, "unityCloud 配置错误")
		return nil
	}
	uuid := uuid.New().String()
	unityCloudConfig.Uuid = uuid
	unityCloudConfig.AllocationTTL = "1h"
	return &UnityService{}
}

func (s *UnityService) GetStatus() bool {
	return s.getIng
}

func (s *UnityService) GetAllocation(ctx context.Context) (*Allocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.getIng = true

	v, err := gcache.Get(ctx, allocationKey)
	if err != nil {
		s.getIng = false
		return nil, gerror.New("读取缓存错误:" + err.Error())
	}

	var alloc *Allocation
	if v.IsNil() {
		alloc, err = getAllocation(ctx)
		if err != nil {
			g.Log("test").Async().Info(ctx, "无缓存，先查询是否存在服务，如果存在就不需要创建，但是报错:"+err.Error())
		}

		// 如果服务器还存在，就直接返回了。就是缓存和服务器的时间不一致引起的
		if alloc != nil && alloc.Ip != "" {
			s.getIng = false
			return alloc, nil
		}

		alloc = buildAllocation(ctx)
		s.getIng = false
		return alloc, nil
	}
	err = v.Scan(&alloc)
	if err != nil {
		s.getIng = false
		return nil, gerror.Newf("缓存数据转 Allocation 失败:%s", err.Error())
	}
	s.getIng = false

	g.Log("text").Async().Info(ctx, "从缓存读取创建的服务器信息")
	return alloc, nil
}

func buildAllocation(ctx context.Context) *Allocation {
	url := fmt.Sprintf("https://multiverse.scaling.unity.cn/v1/allocation/games/%s/allocations", unityCloudConfig.AppId)
	data := map[string]string{
		"uuid":      unityCloudConfig.Uuid,
		"profileId": unityCloudConfig.ProfileId,
		"regionId":  unityCloudConfig.RegionId,
		// "allocationTTL": unityCloudConfig.AllocationTTL,
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": getBasicAuthorization(unityCloudConfig.AppId, unityCloudConfig.AppServerSecret),
	}

	v, err := curl.NewHttpClient().Post(ctx, url, data, headers)
	if err != nil {
		g.Log("test").Async().Errorf(ctx, "创建 Allocation（%s） 失败:%s", unityCloudConfig.Uuid, err.Error())
		return nil
	}

	type Result struct {
		Allocation *Allocation `json:"allocation"`
	}
	var result = &Result{}
	err = gjson.Unmarshal([]byte(v), result)
	if err != nil {
		g.Log("test").Async().Errorf(ctx, "解析 Allocation（%s） 失败:%s", unityCloudConfig.Uuid, err.Error())
		return nil
	}

	g.Log("test").Async().Info(ctx, "服务已创建，等待创建成功\n\n")

	if result.Allocation.Ip == "" {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		pollCount := 0
		for {
			pollCount++
			alloc, err := getAllocation(ctx)
			if err != nil {
				g.Log("test").Async().Info(ctx, "获取 Create Allocation 失败")
				break
			}

			if alloc != nil && alloc.Ip != "" {
				result.Allocation = alloc
				break
			}

			if pollCount > 30 {
				g.Log("test").Async().Errorf(ctx, "尝试获取 Allocation（%s） 超时， 用时 %d 秒", unityCloudConfig.Uuid, 30*2)
				break
			}

			// 没拿到，等待，同时监听ctx取消
			select {
			case <-ticker.C:
			case <-ctx.Done():
				g.Log("test").Async().Errorf(ctx, "轮询获取 Allocation 被上下文取消，ip一直为空")
			}
		}
	}

	if result.Allocation.Ip != "" {
		gcache.Set(ctx, allocationKey, result.Allocation, 20*time.Minute)
	}

	return result.Allocation
}

func getAllocation(ctx context.Context) (*Allocation, error) {
	url := fmt.Sprintf("https://multiverse.scaling.unity.cn/v1/allocation/games/%s/allocations/%s", unityCloudConfig.AppId, unityCloudConfig.Uuid)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": getBasicAuthorization(unityCloudConfig.AppId, unityCloudConfig.AppServerSecret),
	}

	v, err := curl.NewHttpClient().Get(ctx, url, nil, headers)
	if err != nil {
		g.Log("test").Async().Infof(ctx, "请求 url: %s", url)
		g.Log("test").Async().Infof(ctx, "获取 Allocation（%s） 失败:%s", unityCloudConfig.Uuid, err.Error())
		return nil, err
	}

	type Result struct {
		Allocation *Allocation `json:"allocation"`
	}
	var result = &Result{}
	err = gjson.Unmarshal([]byte(v), result)
	if err != nil {
		g.Log("test").Async().Errorf(ctx, "Get Allocation 数据解析失败:%s", err.Error())
		return nil, err
	}

	if result.Allocation.Ip == "" {
		g.Log("test").Async().Info(ctx, "服务创建中，请等待")
		return nil, nil
	}
	g.Log("test").Async().Info(ctx, "服务已创建，请使用")
	return result.Allocation, nil
}

func getBasicAuthorization(appID, appServiceSecret string) string {
	credentials := appID + ":" + appServiceSecret
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encodedCredentials
}
