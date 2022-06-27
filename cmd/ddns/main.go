package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sempr/ddns"
	"github.com/sempr/ddns/api"
	"github.com/sempr/ddns/query"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	defer rdb.Close()

	var s = ddns.NewRedisDNSStorage(rdb)
	domain := ".ddns.bigking.tk"
	api := api.NewAPIServer(s, "/", domain)
	query := query.NewDNSQuery(s, domain)

	go func() {
		api.Run(18099)
	}()

	go func() {
		query.Run(15353)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := api.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	if err := query.Stop(); err != nil {
		log.Fatal(err)
	}
}
