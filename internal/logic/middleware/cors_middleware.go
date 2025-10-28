package middleware

import (
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type CORSConfig struct {
	AllowDomain      []string `json:"allow_domain"`
	AllowOrigin      string   `json:"allow_origin"`
	AllowCredentials string   `json:"allow_credentials"`
	AllowMethods     string   `json:"allow_methods"`
	AllowHeaders     string   `json:"allow_headers"`
}

func CORS(r *ghttp.Request) {
	corsConfig := &CORSConfig{}
	err := g.Cfg("cors").MustGet(r.GetCtx(), "cors.admin").Scan(corsConfig)
	if err != nil {
		gerror.Wrap(err, "cors config 配置错误")
		return
	}

	corsOptions := r.Response.DefaultCORSOptions()
	corsOptions.AllowDomain = corsConfig.AllowDomain
	corsOptions.AllowOrigin = corsConfig.AllowOrigin
	corsOptions.AllowCredentials = corsConfig.AllowCredentials
	corsOptions.AllowMethods = corsConfig.AllowMethods
	corsOptions.AllowHeaders = corsConfig.AllowHeaders

	r.Response.CORS(corsOptions)
}
