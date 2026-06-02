package user

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/pkg/hash"
)

func (s *sUser) AccountExist(ctx context.Context, account string) (res bool, err error) {
	res, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Account, account).Exist()
	return
}

func (s *sUser) Create(ctx context.Context, req *model.UserCreateReq) (res *model.UserCreateRes, err error) {
	password, err := hash.BcryptHash(req.Password)
	if err != nil {
		return nil, err
	}

	id, err := dao.Users.Ctx(ctx).Data(do.Users{
		Account:  req.Account,
		Password: password,
		Name:     req.Name,
		Gender:   req.Gender,
		Role:     req.Role,
	}).InsertAndGetId()

	if err != nil {
		return nil, err
	}

	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, id).Scan(&res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
