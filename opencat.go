package main

import (
	"log"
	"opencatd-open/router"
	"opencatd-open/store"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "reset_root" {
		log.Println("reset root token...")
		ntoken := uuid.NewString()
		if err := store.UpdateUser(uint(1), ntoken); err != nil {
			log.Fatalln(err)
			return
		}
		log.Println("new root token:", ntoken)
		return
	}

	r := gin.Default()
	group := r.Group("/1")
	{
		group.Use(router.AuthMiddleware())

		// 获取当前用户信息
		group.GET("/me", router.HandleMe)

		// 获取所有Key
		group.GET("/keys", router.HandleKeys)

		// 获取所有用户信息
		group.GET("/users", router.HandleUsers)

		// 添加Key
		group.POST("/keys", router.HandleAddKey)

		// 删除Key
		group.DELETE("/keys/:id", router.HandleDelKey)

		// 添加用户
		group.POST("/users", router.HandleAddUser)

		// 删除用户
		group.DELETE("/users/:id", router.HandleDelUser)

		// 重置用户Token
		group.POST("/users/:id/reset", router.HandleResetUserToken)
	}

	// 初始化用户
	r.POST("/1/users/init", router.Handleinit)

	r.POST("/v1/chat/completions", router.HandleProy)
	r.GET("/v1/models", router.HandleProy)
	r.GET("/v1/dashboard/billing/credit_grants", router.HandleProy)

	r.Run(":80")
}
