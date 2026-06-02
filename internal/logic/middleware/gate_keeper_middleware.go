package middleware

import (
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func GateKeeper(r *ghttp.Request) bool {
	sign := r.Header.Get("Sign")
	time := r.Header.Get("Time")
	if sign == "" || time == "" {
		r.Response.WriteHeader(403)
		r.Response.WriteJson(g.Map{
			"message": "请 5 分钟后再试",
		})
		return false
	}

	gatekeeperMap := g.Cfg().MustGet(r.Context(), "api.gatekeeper").Map()
	secret := ""
	if gatekeeperMap != nil {
		secret = gatekeeperMap["secret"].(string)
	}

	newSign := gmd5.MustEncryptString(secret + time)

	if newSign != sign {
		r.Response.WriteHeader(403)
		r.Response.WriteJson(g.Map{
			"message": "请 10 分钟后再试",
		})
		return false
	}
	return true
}
