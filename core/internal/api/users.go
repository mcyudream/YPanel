package api

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// create 建用户（统一 bcrypt）。
func (u *UserAPI) create(username, password, nickname, role string) (*model.User, error) {
	hash, err := u.hashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username: username, Password: hash, Nickname: nickname,
		Role: role, Status: 1, TokenVersion: 1,
	}
	if err := u.DB.Create(user).Error; err != nil {
		return nil, errs.Wrapc(errs.CodeConflict, err.Error())
	}
	return user, nil
}

func (u *UserAPI) hashPassword(pwd string) (string, error) {
	if len(pwd) < 6 || len(pwd) > 64 {
		return "", errs.New(errs.CodeBadRequest, "error.passwordLength", "密码长度需在 6-64 位之间")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
