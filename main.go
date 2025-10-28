package main

import (
	_ "zjsj/internal/packed"

	_ "zjsj/internal/logic"

	"github.com/gogf/gf/v2/os/gctx"

	"zjsj/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
