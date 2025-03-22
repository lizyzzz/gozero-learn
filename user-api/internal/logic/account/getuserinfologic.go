package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"user-api/internal/biz"
	"user-api/internal/model"
	"user-api/internal/svc"
	"user-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserInfoLogic) GetUserInfo() (resp *types.UserInfoResp, err error) {
	// 获取用户信息逻辑

	// 1. 从 jwt 中获取 userId
	userId, err := l.ctx.Value("userId").(json.Number).Int64()

	if err != nil {
		return nil, biz.TokenError
	}

	// 2. 查找用户信息
	userModel := model.NewUserModel(l.svcCtx.Conn)
	user, err := userModel.FindOne(l.ctx, userId)

	if err != nil && (errors.Is(err, model.ErrNotFound) || errors.Is(err, sql.ErrNoRows)) {
		return nil, biz.TokenError
	} else if err != nil {
		return nil, biz.DBError
	}

	resp = &types.UserInfoResp{
		Username: user.Username,
		Id:       user.Id,
	}

	return resp, nil
}
