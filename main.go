package main

import (
  "fmt"
  "strconv"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/jinzhu/gorm"
  _ "github.com/go-sql-driver/mysql"
)

type User struct {
  gorm.Model
  Name string
  Email string
}

func main() {
  db := sqlConnect()
  db.AutoMigrate(&User{})
  defer db.Close()

  router := gin.Default()

  router.LoadHTMLGlob("templates/*.html")

  router.GET("/", listUsers)
  router.POST("/", createUser)
  router.POST("/delete/:id", deleteUser)

  api := router.Group("api")
	{
    api.GET("/", apiListUsers)
    //api.POST("/new", apiCreateUser)
    //api.POST("/read", apiDeleteUser)
	}

  router.Run(":3000")
}

func apiListUsers(ctx *gin.Context) {
  db := sqlConnect()
  var users []User
  db.Order("created_at asc").Find(&users)
  defer db.Close()

  ctx.JSON(200, gin.H{
    "users": users,
  })
}

func listUsers(ctx *gin.Context) {
  db := sqlConnect()
  var users []User
  db.Order("created_at asc").Find(&users)
  defer db.Close()

  ctx.HTML(200, "index.html", gin.H{
    "users": users,
  })
}

func createUser(ctx *gin.Context) {
  db := sqlConnect()
  name := ctx.PostForm("name")
  email := ctx.PostForm("email")
  fmt.Println("create user " + name + " with email " + email)
  db.Create(&User{Name: name, Email: email})
  defer db.Close()

  ctx.Redirect(302, "/")
}

func deleteUser(ctx *gin.Context) {
  db := sqlConnect()
  n := ctx.Param("id")
  id, err := strconv.Atoi(n)
  if err != nil {
    panic("id is not a number")
  }
  var user User
  db.First(&user, id)
  db.Delete(&user)
  defer db.Close()

  ctx.Redirect(302, "/")
}

func sqlConnect() (database *gorm.DB) {
  DBMS := "mysql"
  USER := "go_test"
  PASS := "password"
  PROTOCOL := "tcp(db:3306)"
  DBNAME := "go_database"

  CONNECT := USER + ":" + PASS + "@" + PROTOCOL + "/" + DBNAME + "?charset=utf8&parseTime=true&loc=Asia%2FTokyo"
  
  count := 0
  db, err := gorm.Open(DBMS, CONNECT)
  if err != nil {
    for {
      if err == nil {
        fmt.Println("")
        break
      }
      fmt.Print(".")
      time.Sleep(time.Second)
      count++
      if count > 180 {
        fmt.Println("")
        panic(err)
      }
      db, err = gorm.Open(DBMS, CONNECT)
    }
  }

  return db
}