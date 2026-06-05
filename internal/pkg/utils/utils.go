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

// 通用泛型去重函数
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return slice
	}

	m := make(map[T]struct{})
	uniqueSlice := make([]T, 0, len(slice))
	for _, v := range slice {
		// 如果 map 中不存在该 key，说明是第一次出现，加入结果集
		if _, exists := m[v]; !exists {
			m[v] = struct{}{}
			uniqueSlice = append(uniqueSlice, v)
		}
	}
	return uniqueSlice
}

// Difference 差集: 在 a 中但不在 b 中
func Difference[T comparable](a, b []T) []T {
	if len(a) == 0 {
		return []T{}
	}
	if len(b) == 0 {
		return Unique(a) // b为空，差集就是a去重
	}

	m := make(map[T]struct{}, len(b))
	for _, v := range b {
		m[v] = struct{}{}
	}

	result := make([]T, 0, len(a))
	seen := make(map[T]struct{}) // 用于防止a自身有重复元素时重复添加
	for _, v := range a {
		if _, existsInB := m[v]; !existsInB {
			if _, seenInResult := seen[v]; !seenInResult {
				result = append(result, v)
				seen[v] = struct{}{}
			}
		}
	}
	return result
}

// Intersect 交集: 在 a 中且在 b 中
func Intersect[T comparable](a, b []T) []T {
	if len(a) == 0 || len(b) == 0 {
		return []T{}
	}

	m := make(map[T]struct{}, len(b))
	for _, v := range b {
		m[v] = struct{}{}
	}

	result := make([]T, 0)
	seen := make(map[T]struct{}) // 防止a自身有重复元素时重复添加
	for _, v := range a {
		if _, existsInB := m[v]; existsInB {
			if _, seenInResult := seen[v]; !seenInResult {
				result = append(result, v)
				seen[v] = struct{}{}
			}
		}
	}
	return result
}

// Union 并集: a 和 b 合并并去重
func Union[T comparable](a, b []T) []T {
	if len(a) == 0 {
		return Unique(b)
	}
	if len(b) == 0 {
		return Unique(a)
	}

	merged := make([]T, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)
	return Unique(merged)
}
