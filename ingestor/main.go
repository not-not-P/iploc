package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/not-not-P/iputils"
)

const (
	linksFile = "links.txt"
)

var (
	IPv6LineMatcher = regexp.MustCompile(`^[a-z]+\|([A-Z][A-Z])\|(ipv6)\|(([0-9]|[a-f]|\:)+)\|([0-9]+)\|[0-9]+\|allocated\|.+$`)
	IPv4LineMatcher = regexp.MustCompile(`^[a-z]+\|([A-Z][A-Z])\|(ipv4)\|(([0-9]|\.)+)\|([0-9]+)\|[0-9]+\|allocated\|.+$`)
)

func main() {
	content, err := os.ReadFile(linksFile)
	if err != nil {
		log.Fatalln(err)
	}
	links := strings.Split(string(content), "\n")
	log.Println("Updating ranges")
	err = UpdateRanges(links)
	if err != nil {
		log.Fatalln(err)
	}
}

func UpdateRanges(links []string) error {
	var total uint32 = 0
	var success uint32 = 0

	var wg sync.WaitGroup
	for _, link := range links {
		link := link
		if len(link) == 0 {
			continue
		}
		wg.Add(1)
		atomic.AddUint32(&total, 1)
		go func() {
			defer wg.Done()
			err := DownloadFile(link)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Done", link)
				atomic.AddUint32(&success, 1)
			}
		}()
	}
	wg.Wait()

	if total == success { // At this point, no other goroutine can access total or success
		return nil
	}
	return errors.New("One or more list has not been processed")
}

func DownloadFile(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	defer client.CloseIdleConnections()
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		m := IPv4LineMatcher.FindStringSubmatch(line)
		version := 4
		if len(m) == 0 {
			m = IPv6LineMatcher.FindStringSubmatch(line)
			version = 6
			if len(m) == 0 {
				continue
			}
		}
		country := m[1]
		ip := net.ParseIP(m[3])
		size, err := strconv.Atoi(m[5])

		if ip == nil || err != nil {
			log.Println("Skip invalid line:", line)
			continue
		}

		if version == 4 {
			ipInt := iputils.IPv4ToUint(ip)
			mask := SizeToCIDR(size)
			fmt.Printf("ip4:  %d => %s|%s/%d\n", ipInt, country, ip, mask)
		} else {
			prefixInt := iputils.IPv6PrefixToUint(ip)
			fmt.Printf("ip6:  %d => %s|%s/%d\n", prefixInt, country, ip, size)
		}
	}
	return nil
}

// size = 2^(32-m), m being the CIDR mask
func SizeToCIDR(size int) int {
	return int(-math.Log2(float64(size))) + 32
}
