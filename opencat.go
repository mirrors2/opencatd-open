package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"opencatd-open/router"
	"opencatd-open/store"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:embed dist/*
var web embed.FS

func getFileSystem(path string) http.FileSystem {
	fs, err := fs.Sub(web, path)
	if err != nil {
		panic(err)
	}

	return http.FS(fs)
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "reset_root":
			log.Println("reset root token...")
			if _, err := store.GetUserByID(uint(1)); err != nil {
				if err == gorm.ErrRecordNotFound {
					log.Println("请在opencat(或其他APP)客户端完成team初始化")
					return
				} else {
					log.Fatalln(err)
					return
				}
			}
			ntoken := uuid.NewString()
			if err := store.UpdateUser(uint(1), ntoken); err != nil {
				log.Fatalln(err)
				return
			}
			log.Println("new root token:", ntoken)
			return
		case "root_token":
			log.Println("reset root token...")
			if user, err := store.GetUserByID(uint(1)); err != nil {
				log.Fatalln(err)
				return
			} else {
				log.Println("root token:", user.Token)
				return
			}
		default:
			return
		}

	}
	port := os.Getenv("PORT")
	r := gin.Default()
	group := r.Group("/1")
	{
		group.Use(router.AuthMiddleware())

		// 获取当前用户信息
		group.GET("/me", router.HandleMe)

		group.GET("/me/usages", router.HandleMeUsage)

		// 获取所有Key
		group.GET("/keys", router.HandleKeys)

		// 获取所有用户信息
		group.GET("/users", router.HandleUsers)

		group.GET("/usages", router.HandleUsage)

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

	r.Any("/v1/*proxypath", router.HandleProy)

	// r.POST("/v1/chat/completions", router.HandleProy)
	// r.GET("/v1/models", router.HandleProy)
	// r.GET("/v1/dashboard/billing/subscription", router.HandleProy)

	// r.Use(static.Serve("/", static.LocalFile("dist", false)))
	idxFS, err := fs.Sub(web, "dist")
	if err != nil {
		panic(err)
	}
	r.GET("/", gin.WrapH(http.FileServer(http.FS(idxFS))))
	assetsFS, err := fs.Sub(web, "dist/assets")
	if err != nil {
		panic(err)
	}
	r.GET("/assets/*filepath", gin.WrapH(http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS)))))
	if port == "" {
		port = "80"
	}
	r.Run(":" + port)
}
