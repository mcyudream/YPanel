package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// tokenTTL 会话有效期。
const tokenTTL = 24 * time.Hour

// claims 自定义声明。
type claims struct {
	UID          uint   `json:"uid"`
	Username     string `json:"username"`
	TokenVersion int    `json:"ver"`
	jwt.RegisteredClaims
}

// IssueToken 签发会话 token。
func (a *Auth) IssueToken(u *model.User) (string, time.Time, error) {
	exp := time.Now().Add(tokenTTL)
	c := claims{
		UID:          u.ID,
		Username:     u.Username,
		TokenVersion: u.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Username,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ypanel",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(a.secret)
	return token, exp, err
}

// ParseToken 校验并解析 token；同时校验用户当前 TokenVersion（改密后旧 token 失效）。
func (a *Auth) ParseToken(tokenStr string) (*claims, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errs.ErrUnauthorized
		}
		return a.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errs.ErrUnauthorized
	}
	u, err := a.ByID(c.UID)
	if err != nil || u.Status != 1 || u.TokenVersion != c.TokenVersion {
		return nil, errs.ErrUnauthorized
	}
	return &c, nil
}
