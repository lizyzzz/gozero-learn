package biz

import "github.com/golang-jwt/jwt/v4"

// secretKey: JWT加密密钥
// timeStamp: 时间戳
// seconds: 过期时间
// payload: 数据载体
func GetJwtToken(secretKey string, timeStamp, seconds int64, payload any) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = timeStamp + seconds
	claims["timeStamp"] = timeStamp
	claims["userId"] = payload
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey)) // HS256算法的密钥必须是 []byte
}
