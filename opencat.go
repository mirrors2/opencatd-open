package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"opencatd-open/router"
	"opencatd-open/store"
	"os"

	"github.com/duke-git/lancet/v2/fileutil"
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
		type user struct {
			ID    uint
			Name  string
			Token string
		}
		var us []user
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
			log.Println("[success]new root token:", ntoken)
			return
		case "root_token":
			log.Println("query root token...")
			if user, err := store.GetUserByID(uint(1)); err != nil {
				log.Fatalln(err)
				return
			} else {
				log.Println("[success]root token:", user.Token)
				return
			}
		case "save":
			log.Println("backup user info -> user.json")
			if users, err := store.GetAllUsers(); err != nil {
				log.Fatalln(err)
				return
			} else {
				for _, u := range users {
					us = append(us, user{ID: u.ID, Name: u.Name, Token: u.Token})
				}
			}
			if !fileutil.IsExist("./db/user.json") {
				file, err := os.Create("./db/user.json")
				if err != nil {
					log.Fatalln(err)
					return
				}
				defer file.Close()
			} else {
				// 文件存在，打开文件
				file, _ := os.OpenFile("./db/user.json", os.O_RDWR|os.O_TRUNC, 0666)
				defer file.Close()

				buff := bytes.NewBuffer(nil)
				json.NewEncoder(buff).Encode(us)

				file.WriteString(buff.String())
				fmt.Println("------- END -------")
				return
			}
		case "load":
			fmt.Println("\nimport user.json -> db")
			if !fileutil.IsExist("./db/user.json") {
				log.Fatalln("404! user.json is not found.")
				return
			}
			users, err := store.GetAllUsers()
			if err != nil {
				log.Println(err)
				return
			}
			if len(users) != 0 {
				log.Println("user db 存在数据，取消导入")
				return
			}
			file, err := os.Open("./db/user.json")
			if err != nil {
				fmt.Println("Error opening file:", err)
				return
			}
			defer file.Close()

			decoder := json.NewDecoder(file)
			err = decoder.Decode(&us)
			if err != nil {
				fmt.Println("Error decoding JSON:", err)
				return
			}
			for _, u := range us {
				log.Println(u.ID, u.Name, u.Token)
				err := store.CreateUser(&store.User{ID: u.ID, Name: u.Name, Token: u.Token})
				if err != nil {
					log.Println(err)
				}
			}
			fmt.Println("------- END -------")
			return

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
