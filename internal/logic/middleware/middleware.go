package middleware

import (
	"zjsj/internal/pkg/jwt"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
)

type sMiddleware struct{}

func init() {
	service.RegisterMiddleware(New())
}

func New() service.IMiddleware {
	return &sMiddleware{}
}

func (s *sMiddleware) GateKeeper(r *ghttp.Request) {
	if !GateKeeper(r) {
		r.Exit()
		return
	}

	r.Middleware.Next()
}

func (s *sMiddleware) CORS(r *ghttp.Request) {
	CORS(r)

	r.Middleware.Next()
}

func (s *sMiddleware) Response(r *ghttp.Request) {
	r.Middleware.Next()

	Response(r)
}

func (s *sMiddleware) Auth(r *ghttp.Request) {
	jwt.NewJwt(r.GetCtx()).MiddlewareFunc()(r)

	r.Middleware.Next()
}
