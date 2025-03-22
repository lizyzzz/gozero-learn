package biz

// 自定义 Error 类型
type Error struct {
	Code int    `json:"code"` // http 状态码
	Msg  string `json:"msg"`
}

func NewError(code int, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}

// 实现 error 接口
func (e *Error) Error() string {
	return e.Msg
}
