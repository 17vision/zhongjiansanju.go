package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

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

// PercentageToFloat 将百分比字符串转换为小数
// 例如："85%" -> 0.85，"20%" -> 0.2
func PercentageToFloat(percentage string) (float64, error) {
	// 移除百分号
	cleanStr := strings.TrimSuffix(percentage, "%")
	if cleanStr == percentage {
		return 0, errors.New("invalid percentage format: missing % sign")
	}

	// 转换为浮点数
	number, err := strconv.ParseFloat(cleanStr, 64)
	if err != nil {
		return 0, errors.New("invalid percentage format: not a number")
	}

	return number / 100, nil
}
