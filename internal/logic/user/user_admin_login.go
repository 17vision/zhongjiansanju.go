package user

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/pkg/hash"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

func (s *sUser) Login(ctx context.Context, account string, password string) (res *model.UserLoginRes, err error) {
	var user *model.User
	err = dao.Users.Ctx(ctx).Where(do.Users{
		Account: account,
	}).Scan(&user)

	if err != nil {
		return
	}

	if user == nil {
		err = gerror.New("账号不存在")
		return
	}

	check := hash.BcryptCheck(password, user.Password)
	if !check {
		err = gerror.New("账号或密码错误")
		return
	}

	token, expiredAt, err := buildToken(ctx, user.Id, user.Name)
	if err != nil {
		return nil, err
	}

	res = &model.UserLoginRes{}
	res.Token = token
	res.ExpiredAt = gtime.New(expiredAt)
	return
}
