package sheket_handler

import "github.com/gin-gonic/gin"

const _ERROR_MSG = "error_message"

type SheketError struct {
	Error interface{}
	Code  int
}

type SheketHandler func(c *gin.Context) *SheketError

func HandleError(h SheketHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			c.JSON(err.Code, gin.H{_ERROR_MSG: err.Error})
		}
	}
}
