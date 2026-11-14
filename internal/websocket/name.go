package websocket

import (
	"math/rand"
	"time"
)

var (
	r        = rand.New(rand.NewSource(time.Now().UnixNano()))
	surnames = []string{
		"赵", "钱", "孙", "李", "周", "吴", "郑", "王",
		"冯", "陈", "褚", "卫", "蒋", "沈", "韩", "杨",
		"朱", "秦", "尤", "许", "何", "吕", "施", "张",
		"孔", "曹", "严", "华", "金", "魏", "陶", "姜",
		"戚", "谢", "邹", "喻", "柏", "水", "窦", "章",
		"云", "苏", "潘", "葛", "奚", "范", "彭", "郎",
		"鲁", "韦", "昌", "马", "苗", "凤", "花", "方",
		"俞", "任", "袁", "柳", "酆", "鲍", "史", "唐",
	}
	boys = []string{
		"伟", "强", "磊", "洋", "勇", "军", "杰", "涛",
		"明", "超", "浩", "鹏", "博", "宇", "晨", "轩",
		"泽", "睿", "诚", "俊", "帆", "彬", "恒", "凯",
	}
	girls = []string{
		"芳", "娜", "敏", "静", "丽", "艳", "玲", "娟",
		"婷", "雪", "欣", "雅", "雯", "萱", "瑶", "菲",
		"萍", "琳", "妍", "雨", "怡", "玥", "瑾", "茹",
	}
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GetName() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	gender := r.Int() % 3

	return RandomName(gender)
}

func RandomName(gender int) string {
	sur := surnames[r.Intn(len(surnames))]
	var pool []string
	switch gender {
	case 1:
		pool = boys
	case 2:
		pool = girls
	default:
		if r.Intn(2) == 0 {
			pool = boys
		} else {
			pool = girls
		}
	}
	if r.Intn(3) == 0 {
		return sur + pool[r.Intn(len(pool))]
	}
	return sur + pool[r.Intn(len(pool))] + pool[r.Intn(len(pool))]
}
