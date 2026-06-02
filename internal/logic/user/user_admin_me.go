package user

import (
	"context"
	"fmt"
	"zjsj/internal/dao"
	"zjsj/internal/model"
)

func (s *sUser) Me(ctx context.Context, id int64) (res *model.User, err error) {
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, id).Scan(&res)

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
