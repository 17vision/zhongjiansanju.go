package user

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/pkg/hash"

	"github.com/gogf/gf/v2/frame/g"
)

func (s *sUser) Update(ctx context.Context, id int64, req *model.UserUpdateReq) (res *model.UserUpdateRes, err error) {
	data := g.Map{}

	if req.Password != "" {
		data["password"], err = hash.BcryptHash(req.Password)
		if err != nil {
			return nil, err
		}
	}

	if req.Name != "" {
		data["name"] = req.Name
	}

	if req.Gender != 0 {
		data["gender"] = req.Gender
	}

	if req.Role != 0 {
		data["role"] = req.Role
	}

	_, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, id).Data(data).Update()
	if err != nil {
		return nil, err
	}

	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, id).Scan(&res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
