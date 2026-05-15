package pkg

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Data: data})
}

func OKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: msg})
}

func Fail(c *gin.Context, httpCode int, code int, msg string) {
	c.JSON(httpCode, Response{Code: code, Message: msg})
}

func PageData(total int64, list interface{}) gin.H {
	return gin.H{
		"total": total,
		"list":  list,
	}
}
