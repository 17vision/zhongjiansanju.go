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
	mu sync.RWMutex
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

func (s *UnityService) GetAllocation(ctx context.Context) (*Allocation, error) {
	v, err := gcache.Get(ctx, allocationKey)
	if err != nil {
		return nil, gerror.New("读取缓存错误:" + err.Error())
	}

	var alloc *Allocation
	if v.IsNil() {
		alloc = buildAllocation(ctx)
		return alloc, nil
	}
	err = v.Scan(&alloc)
	if err != nil {
		return nil, gerror.Newf("缓存数据转 Allocation 失败:%s", err.Error())
	}
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
		g.Log("test").Async().Errorf(ctx, "创建 Allocation 失败:%s", err.Error())
		return nil
	}

	type Result struct {
		Allocation *Allocation `json:"allocation"`
	}
	var result = &Result{}
	err = gjson.Unmarshal([]byte(v), result)
	if err != nil {
		g.Log("test").Async().Errorf(ctx, "Create Allocation 数据解析失败:%s", err.Error())
		return nil
	}

	if result.Allocation.Ip == "" {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		pollCount := 0
		for {
			pollCount++
			alloc, err := getAllocation(ctx)
			if err != nil {
				g.Log("test").Async().Error(ctx, "Create Allocation 失败")
				break
			}

			if alloc != nil && alloc.Ip != "" {
				result.Allocation = alloc
				break
			}

			if pollCount > 20 {
				g.Log("test").Async().Error(ctx, "尝试获取 Allocation 超时， 用时 %d 秒", 20*2)
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
		g.Log("test").Async().Errorf(ctx, "获取 Allocation 失败:%s", err.Error())
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
	return result.Allocation, nil
}

func getBasicAuthorization(appID, appServiceSecret string) string {
	credentials := appID + ":" + appServiceSecret
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encodedCredentials
}
