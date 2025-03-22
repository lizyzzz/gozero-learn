package account

import (
	"context"
	"time"

	"user-api/internal/biz"
	"user-api/internal/model"
	"user-api/internal/svc"
	"user-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterRsp, err error) {
	// 注册逻辑

	// 1. 根据用户名查询是否已经注册
	userModel := model.NewUserModel(l.svcCtx.Conn)
	user, err := userModel.FindUserByName(l.ctx, req.Username)

	if err != nil {
		return nil, biz.DBError
	}

	if user != nil {
		return nil, biz.AlreadyRegister
	}

	// 2. 用户没注册则插入
	_, err = userModel.Insert(l.ctx, &model.User{
		Username:      req.Username,
		Password:      req.Password,
		RegisterTime:  time.Now(),
		LastLoginTime: time.Now(),
	})

	if err != nil {
		return nil, biz.DBError
	}

	return
}
