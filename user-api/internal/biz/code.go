package biz

const OK = 200

var (
	DBError             = NewError(10000, "数据库错误")
	AlreadyRegister     = NewError(10100, "用户已注册")
	UserNameAndPwdError = NewError(10101, "用户名或密码错误")
	TokenError          = NewError(10102, "Token错误")
	RedisError          = NewError(10103, "Redis错误")
)
