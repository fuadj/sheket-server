package main

import (
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/gorilla/securecookie"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	c "sheket/server/controller"
	"sheket/server/controller/auth"
	"sheket/server/models"
)

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/signup", c.UserSignupHandler)
	router.HandleFunc("/signin", c.UserLoginHandler)

	router.HandleFunc("/v1/company/create", auth.RequireLogin(c.CompanyCreateHandler))
	// lists companies a user belongs in
	router.HandleFunc("/v1/company/list", auth.RequireLogin(c.UserCompanyListHandler))

	router.HandleFunc("/v1/member/add", auth.RequireLogin(c.AddCompanyMember))

	router.HandleFunc("/v1/sync/entity", auth.RequireLogin(c.EntitySyncHandler))
	router.HandleFunc("/v1/sync/transaction", auth.RequireLogin(c.TransactionSyncHandler))

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Error, %s request couldn't be matched\n", r.URL.Path)
	})

	fmt.Println("Running!!!")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func init() {
	db_store, err := models.ConnectDbStore()
	if err != nil {
		fmt.Printf("%s", err.Error())
		panic(err)
	}
	store := models.NewShStore(db_store)
	c.Store = store
	auth.Store = store
}
