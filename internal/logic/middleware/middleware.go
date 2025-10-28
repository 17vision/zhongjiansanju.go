package middleware

import (
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
)

type sMiiddleware struct{}

func init() {
	service.RegisterMiiddleware(New())
}

func New() service.IMiiddleware {
	return &sMiiddleware{}
}

func (s *sMiiddleware) CORS(r *ghttp.Request) {
	CORS(r)

	r.Middleware.Next()
}
