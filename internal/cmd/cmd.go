package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

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

						group.GET("/me", user.NewAdmin().Me)

						group.GET("/users", user.NewAdmin().UserList)

						group.POST("/users", user.NewAdmin().Create)

						group.PUT("/users", user.NewAdmin().Update)
					})
				})
			})

			// 插入设备
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(service.Middleware().GateKeeper)
				group.Middleware(service.Middleware().Response)

				group.POST("glasses", glasses.NewApi().Create)

				group.POST("glasses_use", glasses_use.NewApi().Create)
			})

			websocket.BindRouters(s)

			s.Run()
			return nil
		},
	}
)
