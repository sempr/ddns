package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-redis/redis/v8"
	flags "github.com/jessevdk/go-flags"
	"github.com/sempr/ddns"
	"github.com/sempr/ddns/api"
	"github.com/sempr/ddns/query"
)

func main() {

	var opts struct {
		RedisURL string `short:"r" long:"redis" default:"redis://127.0.0.1:6379/0"`
		HttpBind string `short:"b" long:"bind" default:"127.0.0.1:8080"`
		DnsPort  int    `short:"u" long:"udp" default:"5353"`
		Domain   string `short:"d" long:"domain" default:".ddns.bigking.tk"`
	}

	flags.ParseArgs(&opts, os.Args)
	fmt.Println(opts)

	REDIS_URL := opts.RedisURL

	opt, _ := redis.ParseURL(REDIS_URL)
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	var s = ddns.NewRedisDNSStorage(rdb)

	domain := opts.Domain
	api := api.NewAPIServer(s, "/", domain)
	query := query.NewDNSQuery(s, domain)

	go func() {
		api.Run(opts.HttpBind)
	}()

	go func() {
		query.Run(opts.DnsPort)
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
