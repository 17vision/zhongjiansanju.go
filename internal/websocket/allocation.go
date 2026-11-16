package websocket

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ensureRoomAllocation 确保房间有云服务器：没有就异步创建一个（一个房间一个服务器）
func (manager *Manager) ensureRoomAllocation(ctx context.Context, room *Room) {
	manager.mu.Lock()
	if room.AllocationUuid != "" || room.allocCreating {
		manager.mu.Unlock()
		return
	}
	room.allocCreating = true
	roomId := room.Id
	manager.mu.Unlock()

	alloc, err := unityService.CreateAllocation(ctx, roomId, map[string]string{
		"recordHost": recordHost,
	})
	if err != nil {
		g.Log("test").Async().Errorf(ctx, "房间 %s 创建云服务器失败: %v", roomId, err)
		manager.mu.Lock()
		room.allocCreating = false
		room.ServerStatus = AllocStatusFailed
		manager.mu.Unlock()
		manager.broadcastAsync(ctx, room, WSMessage{
			Type: MsgTypeError,
			Data: UserEventData{RoomId: roomId, Message: "云服务器创建失败:" + err.Error()},
		}, nil)
		return
	}

	manager.mu.Lock()
	room.allocCreating = false
	room.AllocationUuid = alloc.Uuid
	room.ServerStatus = alloc.Status
	if alloc.IsReady() {
		room.Ip = alloc.Ip
	}
	manager.mu.Unlock()

	// 平台直接返回了 ip（极少见），走统一的成功流程
	if alloc.IsReady() {
		manager.onAllocationReady(alloc)
	}
}

// CreateRoomWithServer 供 HTTP 接口使用：创建一个空房间并触发云服务器创建
func (manager *Manager) CreateRoomWithServer(ctx context.Context, mapBase string) *Room {
	manager.mu.Lock()
	rid := nextMapRoomId(mapBase)
	room := &Room{
		Id:       rid,
		MapBase:  mapBase,
		Type:     RoomTypeMap,
		Name:     "地图",
		Capacity: manager.gameConfig.MapCapacity,
		Clients:  make(map[uint64]*Client),
		Status:   RoomStatusWating,
	}
	manager.rooms[RoomTypeMap] = append(manager.rooms[RoomTypeMap], room)
	manager.mu.Unlock()

	manager.ensureRoomAllocation(ctx, room)
	return room
}

// onAllocationReady 云服务器创建成功（轮询/回调触发）：更新房间 ip，尝试绑定已回连的游戏服务器
func (manager *Manager) onAllocationReady(alloc *Allocation) {
	ctx := context.TODO()

	manager.mu.Lock()
	room := manager.findRoomByAllocationUuidLocked(alloc.Uuid)
	if room == nil {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "云服务器就绪但找不到归属房间, uuid=%s, roomId=%s", alloc.Uuid, alloc.RoomId)
		return
	}
	room.Ip = alloc.Ip
	room.ServerStatus = AllocStatusAllocated

	// 游戏服务器可能先于分配完成回连，这里补绑定
	if room.GameServerClient == nil {
		for _, c := range manager.gameServerClients {
			if c.User.Allocation != nil && c.User.Allocation.Uuid == alloc.Uuid {
				room.GameServerClient = c
				c.User.Extend.IsServer = true
				c.RoomId = room.Id
				break
			}
		}
	}
	manager.mu.Unlock()

	// 兼容旧逻辑：记录最近一个可用服务器 ip
	manager.updateGameServerIp(ctx, alloc.Ip)

	manager.notifyServerReady(ctx, room)

	// 服务器就绪，通知大厅用户可以进房间了
	manager.checkUserCanEnterRoom(ctx)
}

// onAllocationFailed 云服务器创建失败：通知房间内用户并送回大厅，销毁房间
func (manager *Manager) onAllocationFailed(alloc *Allocation) {
	ctx := context.TODO()

	manager.mu.Lock()
	room := manager.findRoomByAllocationUuidLocked(alloc.Uuid)
	if room == nil {
		manager.mu.Unlock()
		return
	}
	room.ServerStatus = AllocStatusFailed
	clients := make([]*Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		clients = append(clients, c)
	}
	manager.mu.Unlock()

	msg := alloc.Msg
	if msg == "" {
		msg = "云服务器创建失败"
	}
	for _, c := range clients {
		manager.KickRoom(ctx, room, c.User, msg)
	}
}

// bindGameServer 游戏服务器回连：按分配 uuid 绑定到归属房间
func (manager *Manager) bindGameServer(ctx context.Context, client *Client) {
	var allocUuid string
	if client.User.Allocation != nil {
		allocUuid = client.User.Allocation.Uuid
	}

	manager.mu.Lock()
	var room *Room
	if allocUuid != "" {
		room = manager.findRoomByAllocationUuidLocked(allocUuid)
	}
	// 兜底：没带 uuid 时，绑定第一个已分配 ip 但还没有游戏服务器的房间
	if room == nil {
		for _, r := range manager.rooms[RoomTypeMap] {
			if r.Ip != "" && r.GameServerClient == nil {
				room = r
				break
			}
		}
	}
	if room == nil {
		manager.mu.Unlock()
		g.Log("test").Async().Warningf(ctx, "游戏服务器 %d 回连但找不到归属房间, uuid=%s", client.User.Id, allocUuid)
		return
	}

	room.GameServerClient = client
	client.User.Extend.IsServer = true
	client.RoomId = room.Id
	if room.Ip == "" && client.User.Allocation != nil && client.User.Allocation.Ip != "" {
		room.Ip = client.User.Allocation.Ip
	}
	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "游戏服务器 %d 绑定房间 %s, ip=%s", client.User.Id, room.Id, room.Ip)

	manager.updateGameServerIp(ctx, room.Ip)

	// 告诉游戏服务器它服务的房间
	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeJoined,
		Data: UserEventData{RoomId: room.Id, Ip: room.Ip, UserId: client.User.Id},
	})

	manager.notifyServerReady(ctx, room)
}

// notifyServerReady 房间 ip 和游戏服务器都就绪后，广播 serverReady（只发一次）
func (manager *Manager) notifyServerReady(ctx context.Context, room *Room) {
	manager.mu.Lock()
	if room.serverNotified || room.Ip == "" || room.GameServerClient == nil {
		manager.mu.Unlock()
		return
	}
	room.serverNotified = true
	port := room.GameServerClient.User.Port
	ip := room.Ip
	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "房间 %s 服务器就绪, ip=%s, port=%d", room.Id, ip, port)

	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeServerReady,
		Data: UserEventData{RoomId: room.Id, Ip: ip, Port: port},
	}, nil)
}

// pushServerStatus 给单个用户推送房间服务器状态（创建中/就绪/失败）
func (manager *Manager) pushServerStatus(ctx context.Context, room *Room, client *Client) {
	if room.Type != RoomTypeMap {
		return
	}

	manager.mu.RLock()
	status := room.ServerStatus
	ip := room.Ip
	var port int
	if room.GameServerClient != nil {
		port = room.GameServerClient.User.Port
	}
	manager.mu.RUnlock()

	switch {
	case status == AllocStatusAllocated && ip != "" && port != 0:
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeServerReady,
			Data: UserEventData{RoomId: room.Id, Ip: ip, Port: port},
		})
	case status == AllocStatusFailed:
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeError,
			Data: UserEventData{RoomId: room.Id, Message: "云服务器创建失败，请重新进入房间"},
		})
	default:
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeServerCreating,
			Data: UserEventData{RoomId: room.Id, Message: "云服务器创建中，请等待"},
		})
	}
}

// releaseRoomServer 释放房间绑定的云服务器（HTTP 删除接口用）
func (manager *Manager) releaseRoomServer(ctx context.Context, uuid string) {
	manager.mu.Lock()
	room := manager.findRoomByAllocationUuidLocked(uuid)
	if room != nil {
		room.AllocationUuid = ""
		room.ServerStatus = ""
		room.Ip = ""
		room.serverNotified = false
		room.allocCreating = false
		if room.GameServerClient != nil {
			room.GameServerClient.User.Extend.IsServer = false
			room.GameServerClient.RoomId = ""
			room.GameServerClient = nil
		}
	}
	manager.mu.Unlock()

	delCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	unityService.DeleteAllocation(delCtx, uuid)

	if room != nil {
		manager.broadcastAsync(ctx, room, WSMessage{
			Type: MsgTypeError,
			Data: UserEventData{RoomId: room.Id, Message: "云服务器已释放"},
		}, nil)
	}
}

// findRoomByAllocationUuidLocked 按分配 uuid 找房间（调用方必须持有 manager.mu）
func (manager *Manager) findRoomByAllocationUuidLocked(uuid string) *Room {
	for _, r := range manager.rooms[RoomTypeMap] {
		if r.AllocationUuid == uuid {
			return r
		}
	}
	return nil
}