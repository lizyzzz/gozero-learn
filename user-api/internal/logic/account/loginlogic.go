package account

import (
	"context"
	"strconv"
	"time"

	"user-api/internal/biz"
	"user-api/internal/model"
	"user-api/internal/svc"
	"user-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginRsp, err error) {
	// 1. 校验用户名和密码
	userModel := model.NewUserModel(l.svcCtx.Conn)
	user, err := userModel.FindUserByNameAndPwd(l.ctx, req.Username, req.Password)

	if err != nil {
		return nil, biz.DBError
	}
	if user == nil {
		return nil, biz.UserNameAndPwdError
	}

	rdsKey := "userid:" + strconv.FormatInt(user.Id, 10)
	// 2. 从 redis 中获取 token
	rdsToken, err := l.svcCtx.RedisConn.GetCtx(l.ctx, rdsKey)
	if err != nil {
		return nil, biz.RedisError
	}

	if rdsToken != "" {
		resp = &types.LoginRsp{
			Token: rdsToken,
		}
		return
	}

	// 3. 生成 token
	secret := l.svcCtx.Config.Auth.AccessSecret
	expire := l.svcCtx.Config.Auth.Expire

	token, err := biz.GetJwtToken(secret, time.Now().Unix(), expire, user.Id)
	if err != nil {
		return nil, biz.TokenError
	}

	// 4. 把 token 存入 redis 中, 操作过期踢掉线
	err = l.svcCtx.RedisConn.SetexCtx(l.ctx, rdsKey, token, int(expire))
	if err != nil {
		return nil, biz.RedisError
	}

	resp = &types.LoginRsp{
		Token: token,
	}
	return
}
