package main

import (
	_ "zjsj/internal/packed"

	_ "zjsj/internal/logic"

	_ "time/tzdata"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"

	"zjsj/internal/cmd"
)

func init() {
	// 关键：设置全局时区
	if err := gtime.SetTimeZone("Asia/Shanghai"); err != nil {
		panic(err)
	}
}

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
