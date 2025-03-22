package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		withSession(session sqlx.Session) UserModel
		FindUserByName(ctx context.Context, name string) (*User, error)
		FindUserByNameAndPwd(ctx context.Context, name, pwd string) (*User, error)
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn),
	}
}

func (m *customUserModel) withSession(session sqlx.Session) UserModel {
	return NewUserModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customUserModel) FindUserByName(ctx context.Context, name string) (*User, error) {
	query := fmt.Sprintf("select %s from %s where `username` = ? limit 1", userRows, m.table)
	var resp User
	// 查询
	err := m.conn.QueryRowCtx(ctx, &resp, query, name)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound, sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (m *customUserModel) FindUserByNameAndPwd(ctx context.Context, name, pwd string) (*User, error) {
	query := fmt.Sprintf("select %s from %s where `username` = ? and `password` = ? limit 1", userRows, m.table)
	var resp User
	// 查询
	err := m.conn.QueryRowCtx(ctx, &resp, query, name, pwd)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound, sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
