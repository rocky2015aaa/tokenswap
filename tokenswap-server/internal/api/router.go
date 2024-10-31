package api

import (
	"net/http"

	"github.com/rocky2015aaa/tokenswap-server/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

func NewRouter(handler *handlers.Handler) http.Handler {
	router := gin.Default()
	router.Use(CORSMiddleware())
	router.Use(UserAuthentication())

	v1 := router.Group("/api/v1")

	v1.GET("/health", handler.Ping)
	v1.GET("/auth/ping", handler.Ping)

	// // Admin routes
	// admin := v1.Group("/admin")
	// {
	// 	// All admin management like update fee rate and so on
	// }

	// User routes
	user := v1.Group("/user")
	{
		user.GET("/", handler.GetUserInfo)
		user.POST("/register", handler.Register)
		user.POST("/verfication", handler.Verification)
		user.PATCH("/update-password", handler.UpdatePassword)
	}

	// Token routes
	token := v1.Group("/token")
	{
		token.POST("/refresh", handler.Refresh)
		token.POST("/renew", handler.RenewTokens)
		token.POST("/renew-exp", handler.RenewTokensWithCustomExpiration)
	}

	// Order routes
	order := v1.Group("/order")
	{
		order.POST("/", handler.CreateOrder)
		order.PATCH("/take", handler.TakeOrder)
		order.PATCH("/cancel", handler.CancelOrder)
		order.GET("/list", handler.GetOrderList)         // for order list(public/private, user/orderbook, order_id)?
		order.GET("/common", handler.GetOrderCommonInfo) // for order infor like fee rate or something?
	}

	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"message": "Method Not Allowed",
		})
	})

	return router
}
