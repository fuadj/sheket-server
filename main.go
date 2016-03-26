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

	router.HandleFunc("/createcompany", auth.RequireLogin(c.CompanyCreateHandler))
	router.HandleFunc("/syncentity", auth.RequireLogin(c.EntitySyncHandler))
	router.HandleFunc("/synctrans", auth.RequireLogin(c.TransactionSyncHandler))

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Error, %s request couldn't be matched\n", r.URL.Path)
	})

	fmt.Println("Running!!!")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func init() {
	db_store, err := models.ConnectDbStore()
	if err != nil {
		//panic(err)
		fmt.Printf("%s", err.Error())
		return
	}
	store := models.NewShDataStore(db_store)
	c.Store = store
	auth.Store = store
}
