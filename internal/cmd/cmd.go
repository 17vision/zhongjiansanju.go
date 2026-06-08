package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"zjsj/internal/controller/game_record"
	"zjsj/internal/controller/glasses"
	"zjsj/internal/controller/glasses_use"
	"zjsj/internal/controller/user"
	"zjsj/internal/service"
	"zjsj/internal/websocket"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(service.Middleware().CORS)
				group.Middleware(service.Middleware().GateKeeper)
				group.Middleware(service.Middleware().Response)

				group.Group("/admin", func(group *ghttp.RouterGroup) {
					group.POST("/login", user.NewAdmin().Login)

					group.Group("/", func(group *ghttp.RouterGroup) {
						group.Middleware(service.Middleware().Auth)

						// 自己
						group.GET("/me", user.NewAdmin().Me)

						// 用户
						group.GET("/users", user.NewAdmin().UserList)

						group.POST("/users", user.NewAdmin().Create)

						group.PUT("/users", user.NewAdmin().Update)

						// 设备
						group.GET("/glasses/info", glasses.NewAdmin().Info)

						group.GET("/glasses", glasses.NewAdmin().List)

						group.POST("/glasses", glasses.NewAdmin().Create)

						group.DELETE("/glasses", glasses.NewAdmin().Delete)

						// 统计
						group.GET("/game_records/info", game_record.NewAdmin().Info)

						group.GET("/game_records/timeperiod_count", game_record.NewAdmin().TimeperiodCount)

						group.GET("/game_records", game_record.NewAdmin().List)
					})
				})
			})

			// 插入设备
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(service.Middleware().GateKeeper)
				group.Middleware(service.Middleware().Response)

				// 设备
				group.GET("/glasses", glasses.NewApi().List)

				group.POST("/glasses", glasses.NewApi().Create)

				// 设备使用
				group.POST("/glasses_use", glasses_use.NewApi().Create)

				group.GET("/glasses_use/one", glasses_use.NewApi().One)
			})

			websocket.BindRouters(s)

			s.Run()
			return nil
		},
	}
)
