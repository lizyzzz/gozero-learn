package middlewares

import (
	"fmt"
	"net/http"
)

func PrintLog(next http.HandlerFunc) http.HandlerFunc {
	count := 0
	return func(w http.ResponseWriter, r *http.Request) {
		count++
		fmt.Printf("this is %d log.\n", count)
		next(w, r)
	}
}
