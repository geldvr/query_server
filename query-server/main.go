package main

import (
	"context"
	"itv/query-server/dispatcher"
	"itv/query-server/routes"
	"itv/shared/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, signals...)

	httpPort := ":" + config.GetString("query-server.port")
	router := routes.NewRouter()
	httpServer := http.Server{Addr: httpPort, Handler: router}

	dispatcher := dispatcher.NewDispatcher()
	dispatcher.Run()

	go func() {
		log.Printf("HTTP server started[pid %d/port %s]", syscall.Getpid(), httpPort)

		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("HTTP server server err: %s", err.Error())
		}
	}()

	<-stop

	log.Println("Shutting down...")
	dispatcher.Stop()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	httpServer.Shutdown(ctx)
	log.Println("Server stopped")
}
