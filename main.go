package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/not-not-P/iputils"
)

func main() {
	host := os.Getenv("HTTP_HOST")
	if len(host) == 0 {
		host = ":8080"
	}
	fmt.Println("HTTP server listening to", host)
	http.HandleFunc("/", LookupHandler)
	http.ListenAndServe(host, nil)
}

func LookupHandler(w http.ResponseWriter, req *http.Request) {
	// req.URL is always at least one-char long (minimal URL: "/")
	ipStr := req.URL.String()[1:]
	ip := net.ParseIP(ipStr)
	if ip == nil {
		w.WriteHeader(400)
		return
	}
	if iputils.IsBogon(ip) {
		w.WriteHeader(400)
		return
	}

	var ipVer string
	if strings.Contains(ipStr, ":") {
		ipVer = iputils.IPv6
	} else {
		ipVer = iputils.IPv4
	}

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer rdb.Close()

	var max string
	if ipVer == iputils.IPv6 {
		max = strconv.FormatUint(iputils.IPv6PrefixToUint(ip), 10)
	} else {
		max = strconv.FormatUint(uint64(iputils.IPv4ToUint(ip)), 10)
	}
	record, _ := rdb.ZRevRangeByScoreWithScores(ctx, "ip"+ipVer, &redis.ZRangeBy{
		Min:    "0",
		Max:    max,
		Offset: 0,
		Count:  1,
	}).Result()

	if len(record) == 0 {
		w.WriteHeader(404)
		return
	}
	parts := strings.Split(record[0].Member.(string), "|")

	w.Header().Set("Content-Type", "application/json; utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(struct {
		IP      string `json:"ip"`
		Country string `json:"country"`
		Route   string `json:"route"`
	}{
		IP:      ip.String(),
		Country: parts[0],
		Route:   parts[1],
	})
}
