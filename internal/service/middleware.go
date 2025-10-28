// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"github.com/gogf/gf/v2/net/ghttp"
)

type (
	IMiiddleware interface {
		CORS(r *ghttp.Request)
	}
)

var (
	localMiiddleware IMiiddleware
)

func Miiddleware() IMiiddleware {
	if localMiiddleware == nil {
		panic("implement not found for interface IMiiddleware, forgot register?")
	}
	return localMiiddleware
}

func RegisterMiiddleware(i IMiiddleware) {
	localMiiddleware = i
}
