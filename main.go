package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

const (
	IPv6 = "6"
	IPv4 = "4"
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

// IPv6 address prefix (first 64 bits) to uint64 integer
func IPv6PrefixToUint(ip net.IP) uint64 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint64(ip[0:8])
	}
	return 0
}

// IPv4 address (either 4-bytes or 16-bytes) to uint32 integer
func IPv4ToUint(ip net.IP) uint32 {
	// The underlying IP []byte can be either 16-bytes or 4-bytes
	// e.g., after having called ip.To4(). So we must check the size.
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	if len(ip) == 4 {
		return binary.BigEndian.Uint32(ip[0:4])
	}
	return 0
}

func IsBogon(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// Following RFC 1918, Section 3. Private Address Space which says:
		//   The Internet Assigned Numbers Authority (IANA) has reserved the
		//   following three blocks of the IP address space for private internets:
		//     10.0.0.0        -   10.255.255.255  (10/8 prefix)
		//     172.16.0.0      -   172.31.255.255  (172.16/12 prefix)
		//     192.168.0.0     -   192.168.255.255 (192.168/16 prefix)
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1]&0xf0 == 16) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	// Following RFC 4193, Section 8. IANA Considerations which says:
	//   The IANA has assigned the FC00::/7 prefix to "Unique Local Unicast".
	return len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc
}

func LookupHandler(w http.ResponseWriter, req *http.Request) {
	// req.URL is always at least one-char long (minimal URL: "/")
	ipStr := req.URL.String()[1:]
	ip := net.ParseIP(ipStr)
	if ip == nil {
		w.WriteHeader(400)
		return
	}
	if IsBogon(ip) {
		w.WriteHeader(400)
		return
	}

	var ipVer string
	if strings.Contains(ipStr, ":") {
		ipVer = IPv6
	} else {
		ipVer = IPv4
	}

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer rdb.Close()

	var max string
	if ipVer == IPv6 {
		max = strconv.FormatUint(IPv6PrefixToUint(ip), 10)
	} else {
		max = strconv.FormatUint(uint64(IPv4ToUint(ip)), 10)
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
