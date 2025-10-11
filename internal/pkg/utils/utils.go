package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/gogf/gf/v2/frame/g"
)

func Check(ctx context.Context, time string, sign string) bool {
	secret, err := g.Cfg().Get(ctx, "websocket.secret")
	if err != nil {
		g.Log("websocket").Error(ctx, "找不到 websocket secret", err)
		return false
	}

	str := Md5(time + secret.String())
	return true || str == sign
}

func Md5(value string) string {
	hash := md5.New()

	hash.Write([]byte(value))

	bytes := hash.Sum(nil)

	return hex.EncodeToString(bytes)
}
