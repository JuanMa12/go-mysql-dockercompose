package main

import (
  "fmt"
  "strconv"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/jinzhu/gorm"
  _ "github.com/go-sql-driver/mysql"

  "net/http"
  "github.com/dgrijalva/jwt-go"
  "os"
  "strings"
)

type User struct {
  gorm.Model
  Name string
  Email string
  Password string
}

type Body struct{
  ID uint64 `json:"id"`
  Email string `json:"email"`
  Password string `json:"password"`
}

type AccessDetails struct {
  AccessUuid string
  UserId   uint64
}

func main() {
  db := sqlConnect()
  db.AutoMigrate(&User{})
  defer db.Close()

  router := gin.Default()

  router.LoadHTMLGlob("templates/*.html")

  router.GET("/", listUsers)
  router.POST("/new", createUser)
  router.POST("/delete/:id", deleteUser)

  api := router.Group("/api")
	{
		api.GET("/", TokenAuthMiddleware(), apiListUsers)
		api.POST("/new", apiCreateUser)
		api.POST("/delete/:id", apiDeleteUser)

    api.POST("/login", Login)
	}

  router.Run(":3003")
}

func Login(c *gin.Context) {
  db := sqlConnect()
  var findUser User
  body := Body{}
  if err := c.ShouldBindJSON(&body); err != nil {
     c.JSON(402, "Invalid json provided")
     return
  } 
  //Where email in user 
  db.Where("Email = ?", body.Email).First(&findUser)
  if findUser.Name == "" && findUser.Email == "" {
    c.JSON(500, gin.H{
      "message":"user dont exists",
    })
    panic("User dont exists")
  }
  defer db.Close()
  c.JSON(401, gin.H{"body":body, "findUser":findUser})
  //compare the user from the request, with the one we defined:
  if body.Email != findUser.Email || body.Password != findUser.Password {
     c.JSON(401, "Please provide valid login details")
     return
  }
  //c.JSON(200, "Information success")
  token, err := CreateToken(findUser.Email)
  if err != nil {
     c.JSON(402, err.Error())
     return
  }
  c.JSON(200, gin.H{"token":token})
}

func apiListUsers(ctx *gin.Context){
  db := sqlConnect()
  var users []User
  db.Order("created_at asc").Find(&users)
  defer db.Close()

  ctx.JSON(200, gin.H{
    "users": users,
  })
  
}

func apiCreateUser(ctx *gin.Context) {
  db := sqlConnect()
  post_name := ctx.PostForm("name")
  post_email := ctx.PostForm("email")
  post_password := ctx.PostForm("password")
  newUser := db.Create(&User{Name: post_name, Email: post_email, Password: post_password})
  defer db.Close()

  ctx.JSON(201, gin.H{
    "message":"user created " + post_name + "with email:" + post_email,
    "model:": newUser, 
  })
}

func apiDeleteUser(ctx *gin.Context) {
  db := sqlConnect()
  n := ctx.Param("id")
  id, err := strconv.Atoi(n)
  if err != nil {
    ctx.JSON(500, gin.H{
      "message":"id is not a number",
    })
    panic("id is not a number")
  }
  var user User
  db.First(&user, id)
  if user.Name == "" && user.Email == "" {
    ctx.JSON(500, gin.H{
      "message":"user dont exists",
    })
    panic("User dont exists")
  }
  deleteUser:= db.Delete(&user)
  defer db.Close()

  ctx.JSON(202, gin.H{
    "message":"user deleted " + user.Name + "with email:" + user.Email,
    "model:": deleteUser, 
  })
}

func listUsers(ctx *gin.Context){
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
  password := ctx.PostForm("password")
  fmt.Println("create user " + name + " with email " + email)
  db.Create(&User{Name: name, Email: email, Password: password })
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


// ### methods JWT

func CreateToken(useremail string) (string, error) {
  var err error
  //Creating Access Token
  os.Setenv("ACCESS_SECRET", "MYSECRECT_KEY") //this should be in an env file
  atClaims := jwt.MapClaims{}
  atClaims["authorized"] = true
  atClaims["user_email"] = useremail
  atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
  at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
  token, err := at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
  if err != nil {
     return "", err
  }
  return token, nil
}

func TokenAuthMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
     err := TokenValid(c.Request)
     if err != nil {
        c.JSON(http.StatusUnauthorized, err.Error())
        c.Abort()
        return
     }
     c.Next()
  }
}

func TokenValid(r *http.Request) error {
  token, err := VerifyToken(r)
  if err != nil {
     return err
  }
  if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
     return err
  }
  return nil
}

func VerifyToken(r *http.Request) (*jwt.Token, error) {
  tokenString := ExtractToken(r)
  token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
     //Make sure that the token method conform to "SigningMethodHMAC"
     if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
     }
     return []byte(os.Getenv("ACCESS_SECRET")), nil
  })
  if err != nil {
     return nil, err
  }
  return token, nil
}

func ExtractToken(r *http.Request) string {
  bearToken := r.Header.Get("Authorization")
  //panic(bearToken)
  //normally Authorization the_token_xxx
  strArr := strings.Split(bearToken, " ")
  if len(strArr) == 2 {
     return strArr[1]
  }
  return ""
}