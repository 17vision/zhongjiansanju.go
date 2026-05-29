package middleware

import (
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func Response(r *ghttp.Request) {
	if r.Response.BufferLength() > 0 {
		return
	}

	var (
		httpStatus = http.StatusForbidden
		err        = r.GetError()
		res        = r.GetHandlerResponse()
	)

	status := make(map[int]int)
	status[http.StatusUnauthorized] = http.StatusUnauthorized
	status[http.StatusForbidden] = http.StatusForbidden
	status[http.StatusTooManyRequests] = http.StatusTooManyRequests
	status[http.StatusInternalServerError] = http.StatusInternalServerError

	if err != nil {
		res = g.Map{
			"message": err.Error(),
		}
	} else {
		if r.Response.Status == 0 || r.Response.Status == http.StatusOK {
			httpStatus = http.StatusOK
		} else {
			if _, ok := status[r.Response.Status]; ok {
				httpStatus = status[r.Response.Status]
			}
		}
	}

	// 主要处理 200 401 403 429 500 这几个状态
	r.Response.WriteHeader(httpStatus)
	r.Response.WriteJson(res)
}
