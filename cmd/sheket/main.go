package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/gorilla/securecookie"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	c "sheket/server/controller"
	"sheket/server/controller/auth"
	"sheket/server/models"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// users will sign-in to this route. If the user doesn't already exist,
	// it will be added to the database. If all is successful, the user will
	// be signed-in with a cookie.
	router.POST("/api/v1/signin/facebook", c.UserSignInHandler)

	router.POST("/api/v1/company/create", auth.RequireLogin(c.CompanyCreateHandler))
	// lists companies a user belongs in
	router.POST("/api/v1/company/list", auth.RequireLogin(c.UserCompanyListHandler))

	router.POST("/api/v1/member/add", auth.RequireLogin(c.AddCompanyMember))

	router.POST("/api/v1/sync/entity", auth.RequireLogin(c.EntitySyncHandler))
	router.POST("/api/v1/sync/transaction", auth.RequireLogin(c.TransactionSyncHandler))

	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	fmt.Println("Running!!!")
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func init() {
	db_store, err := models.ConnectDbStore()
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	store := models.NewShStore(db_store)
	c.Store = store
	auth.Store = store
}
