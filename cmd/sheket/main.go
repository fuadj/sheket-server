package main

import (
	_ "github.com/gorilla/securecookie"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"log"
	"net"
	"os"
	c "sheket/server/controller"
	"sheket/server/controller/auth"
	"sheket/server/models"
	sh_service "sheket/server/sheketproto"
	panic_handler "github.com/kazegusuri/grpc-panic-handler"
	"fmt"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	conn, err := net.Listen("tcp", ":"+port)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	uIntOpt := grpc.UnaryInterceptor(panic_handler.UnaryPanicHandler)
	sIntOpt := grpc.StreamInterceptor(panic_handler.StreamPanicHandler)

	panic_handler.InstallPanicHandler(func(r interface{}) {
		fmt.Printf("panic happened: %v", r)
	})

	grpcServer := grpc.NewServer(uIntOpt, sIntOpt)
	sh_service.RegisterSheketServiceServer(grpcServer, new(c.SheketController))
	grpcServer.Serve(&closableListener{conn})
}

type closableListener struct {
	net.Listener
}

func (c *closableListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now() + (time.Second * 10))
	return conn, nil
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
