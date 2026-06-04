package user

import (
	"context"
	"fmt"
	"zjsj/internal/dao"
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *sUser) Me(ctx context.Context, id int64) (res *model.User, err error) {
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, id).Scan(&res)

	if res == nil {
		return nil, gerror.New("用户不存在")
	}

	roleArr := []string{"", "站长", "管理员", "普通人员"}
	genderArr := []string{"未知", "男", "女"}

	res.Password = ""
	res.RoleStr = roleArr[res.Role]
	res.GenderStr = genderArr[res.Gender]

	if res.Avatar == "" {
		res.Avatar = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", res.Name)
	}
	return
}
