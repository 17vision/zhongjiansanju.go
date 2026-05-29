package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

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
				group.Middleware(service.Middleware().GateKeeper)
				group.Middleware(service.Middleware().Response)

				group.Group("/admin", func(group *ghttp.RouterGroup) {
					group.POST("/login", user.NewAdmin().Login)

					group.Group("/", func(group *ghttp.RouterGroup) {
						group.Middleware(service.Middleware().Auth)
					})
				})
			})

			websocket.BindRouters(s)

			s.Run()
			return nil
		},
	}
)
