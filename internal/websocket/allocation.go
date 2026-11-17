package websocket

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ensureRoomAllocation 确保房间有云服务器：优先复用空闲服务器，没有才异步创建（一个房间一个服务器）
func (manager *Manager) ensureRoomAllocation(ctx context.Context, room *Room) {
	manager.mu.Lock()
	if room.AllocationUuid != "" || room.allocCreating {
		manager.mu.Unlock()
		return
	}
	manager.mu.Unlock()

	// 有已回连的空闲云服务器就直接拿来用，避免重复创建
	if manager.tryReuseIdleServer(ctx, room) {
		return
	}

	manager.mu.Lock()
	// 并发下状态可能已变（别的 goroutine 绑定/创建过），复查
	if room.AllocationUuid != "" || room.allocCreating || room.GameServerClient != nil {
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
	// 创建期间房间被销毁（用户秒退等）：释放刚创建的分配，避免孤儿计费
	if room.destroyed {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "房间 %s 已销毁，释放创建中的云服务器 uuid=%s", roomId, alloc.Uuid)
		go func(id string) {
			delCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			unityService.DeleteAllocation(delCtx, id)
		}(alloc.Uuid)
		return
	}
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

// tryReuseIdleServer 把已回连但没绑定房间的空闲云服务器复用给房间，成功返回 true
func (manager *Manager) tryReuseIdleServer(ctx context.Context, room *Room) bool {
	manager.mu.Lock()
	if room.AllocationUuid != "" || room.GameServerClient != nil {
		manager.mu.Unlock()
		return false
	}
	var server *Client
	for _, c := range manager.gameServerClients {
		if c.RoomId != "" || c.IsClosed() || c.User.Extend.IsServer {
			continue
		}
		// 原房间还在等它回连（分配仍挂在别的现存房间下）：不抢
		if c.User.Allocation != nil && c.User.Allocation.Uuid != "" {
			if r := manager.findRoomByAllocationUuidLocked(c.User.Allocation.Uuid); r != nil && r != room {
				continue
			}
		} else if time.Now().Unix()-c.CreatedAt < 15 {
			// 没带 uuid 无法确认归属：刚回连且别的房间还在创建中，可能是那台的服务器，不抢
			awaiting := false
			for _, r := range manager.rooms[RoomTypeMap] {
				if r != room && r.GameServerClient == nil && r.ServerStatus == AllocStatusCreating {
					awaiting = true
					break
				}
			}
			if awaiting {
				continue
			}
		}
		server = c
		break
	}
	if server == nil {
		manager.mu.Unlock()
		return false
	}

	uuid := ""
	ip := ""
	if server.User.Allocation != nil {
		uuid = server.User.Allocation.Uuid
		ip = server.User.Allocation.Ip
	}
	if ip == "" {
		ip = server.User.Extend.ConnectIp
	}
	server.RoomId = room.Id
	server.User.Extend.IsServer = true
	room.GameServerClient = server
	room.AllocationUuid = uuid
	room.ServerStatus = AllocStatusAllocated
	room.serverNotified = false
	room.Ip = ip
	manager.mu.Unlock()

	// 服务器自报过分配 uuid：把分配记录改挂到新房间，房间销毁时随带释放
	if uuid != "" {
		unityService.RebindRoom(uuid, room.Id)
		if a := unityService.GetAllocation(uuid); a != nil && a.Ip != "" {
			ip = a.Ip
			manager.mu.Lock()
			room.Ip = ip
			manager.mu.Unlock()
		}
	}

	g.Log("test").Async().Infof(ctx, "房间 %s 复用空闲云服务器: id=%d, ip=%s, uuid=%s", room.Id, server.User.Id, ip, uuid)

	manager.updateGameServerIp(ctx, ip)

	// 告诉游戏服务器它服务的新房间
	manager.unicastAsync(ctx, server, WSMessage{
		Type: MsgTypeJoined,
		Data: UserEventData{RoomId: room.Id, Ip: ip, UserId: server.User.Id},
	})

	manager.notifyServerReady(ctx, room)
	manager.checkUserCanEnterRoom(ctx)
	return true
}

// CreateRoomWithServer 供 HTTP 接口使用：创建一个空房间并触发云服务器创建
func (manager *Manager) CreateRoomWithServer(ctx context.Context, mapBase string) *Room {
	manager.mu.Lock()
	rid := nextMapRoomId(mapBase)
	room := &Room{
		Id:        rid,
		MapBase:   mapBase,
		Type:      RoomTypeMap,
		Name:      "地图",
		Capacity:  manager.gameConfig.MapCapacity,
		Clients:   make(map[uint64]*Client),
		Status:    RoomStatusWating,
		CreatedAt: time.Now().Unix(),
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
		// 房间已不存在：释放该分配，避免平台孤儿计费
		go func(id string) {
			delCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			unityService.DeleteAllocation(delCtx, id)
		}(alloc.Uuid)
		return
	}
	room.Ip = alloc.Ip
	room.ServerStatus = AllocStatusAllocated

	// 游戏服务器可能先于分配完成回连，这里补绑定：按 uuid 或连接 IP（游戏服务器 WS 来源 IP 就是云服务器 IP）
	if room.GameServerClient == nil {
		for _, c := range manager.gameServerClients {
			if c.User.Extend.IsServer {
				continue
			}
			if (c.User.Allocation != nil && c.User.Allocation.Uuid == alloc.Uuid) || c.User.Extend.ConnectIp == alloc.Ip {
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

	// 空房间（HTTP 预建后没人进就失败）直接销毁，避免滞留
	manager.mu.RLock()
	empty := len(room.Clients) == 0
	manager.mu.RUnlock()
	if empty {
		manager.destroyRoom(room)
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
	// 兜底 1：按连接 IP 匹配（游戏服务器 WS 来源 IP 就是云服务器 IP）
	if room == nil && client.User.Extend != nil && client.User.Extend.ConnectIp != "" {
		for _, r := range manager.rooms[RoomTypeMap] {
			if r.Ip == client.User.Extend.ConnectIp && r.GameServerClient == nil {
				room = r
				break
			}
		}
	}
	// 兜底 2：没带 uuid 也对不上 IP 时，绑定第一个已分配 ip 但还没有游戏服务器的房间
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
		g.Log("test").Async().Warningf(ctx, "游戏服务器 %d 回连但找不到归属房间, uuid=%s, ip=%s", client.User.Id, allocUuid, client.User.Extend.ConnectIp)
		return
	}

	// 同一房间重复回连（服务器重启等）：顶替旧绑定
	if room.GameServerClient != nil && room.GameServerClient != client {
		room.GameServerClient.User.Extend.IsServer = false
		room.GameServerClient.RoomId = ""
		room.serverNotified = false
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
