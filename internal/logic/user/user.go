package user

import (
	"context"
	"time"
	"zjsj/internal/pkg/jwt"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type sUser struct{}

func init() {
	service.RegisterUser(New())
}

func New() service.IUser {
	return &sUser{}
}

func buildToken(ctx context.Context, id uint64, nickname string) (Token string, ExpiredAt time.Time, err error) {
	payloadData := g.Map{
		"id":       id,
		"nickname": nickname,
	}

	Token, ExpiredAt, err = jwt.NewJwt(ctx).TokenGenerator(payloadData)

	ExpiredAt = gtime.New(ExpiredAt).Local().Time

	if err != nil {
		return
	}
	return
}
