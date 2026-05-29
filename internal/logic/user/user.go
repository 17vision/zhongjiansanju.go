package user

import (
	"context"
	"time"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/pkg/hash"
	"zjsj/internal/pkg/jwt"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
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

	if err != nil {
		return
	}
	return
}

func (s *sUser) Login(ctx context.Context, account string, password string) (res *model.UserLoginRes, err error) {
	err = dao.Users.Ctx(ctx).Where(do.Users{
		Account: account,
	}).Scan(&res)

	if err != nil {
		return
	}

	if res == nil {
		err = gerror.New("账号不存在")
		return
	}

	check := hash.BcryptCheck(password, res.Password)
	if !check {
		err = gerror.New("账号或密码错误")
		return
	}

	token, expiredAt, err := buildToken(ctx, res.Id, res.Name)
	if err != nil {
		return nil, err
	}

	res.Password = ""
	res.Token = token
	res.ExpireIn = gtime.New(expiredAt)

	roleArr := []string{"", "站长", "管理员", "普通人员"}
	res.RoleStr = roleArr[res.Role]
	return
}
