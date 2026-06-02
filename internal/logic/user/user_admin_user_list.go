package user

import (
	"context"
	"fmt"
	"zjsj/internal/dao"
	"zjsj/internal/model"
)

func (s *sUser) List(ctx context.Context, req *model.UserListReq) (res *model.UserListRes, err error) {
	columns := dao.Users.Columns()

	query := dao.Users.Ctx(ctx).OrderDesc(columns.Id)

	if req.Phone != "" {
		query = query.Where(columns.Phone, req.Phone)
	}

	if req.Name != "" {
		query = query.WhereLike(columns.Name, "%"+req.Name+"%")
	}

	var data []*model.User
	var total int
	if err = query.Page(req.Page, req.PageSize).ScanAndCount(&data, &total, true); err != nil {
		return nil, err
	}

	roleArr := []string{"", "站长", "管理员", "普通人员"}
	genderArr := []string{"未知", "男", "女"}
	for _, item := range data {
		item.Password = ""
		item.RoleStr = roleArr[item.Role]
		item.GenderStr = genderArr[item.Gender]

		if item.Avatar == "" {
			item.Avatar = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", item.Name)
		}
	}

	res = &model.UserListRes{}
	res.Total = total
	res.Page = req.Page
	res.PageSize = req.PageSize
	res.Data = data
	return
}
