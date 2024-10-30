package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/projectdiscovery/mapcidr"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Extract Shodan IPInfo from internetdb.shodan.io
// cat /tmp/list_of_IP | cinfo -c 100
var (
	csvOutput      bool
	jsonOutput     bool
	servicesOutput bool
	onlyHost       bool
	concurrency    int
	omitEmpty      bool

	Services []Service
)

type ShodanIPInfo struct {
	Cpes      []string `json:"cpes"`
	Hostnames []string `json:"hostnames"`
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
	Tags      []string `json:"tags"`
	Vulns     []string `json:"vulns"`
}

type Service struct {
	IP       string `json:"ip"`
	Protocol string `json:"protocol"`
	Port     string `json:"port"`
	Service  string `json:"service"`
}

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 2, "Set the concurrency level")
	flag.BoolVar(&jsonOutput, "json", false, "Show Output as Json format")
	flag.BoolVar(&csvOutput, "csv", true, "Show Output as CSV format")
	flag.BoolVar(&servicesOutput, "services", false, "Create also a services.json output")
	flag.BoolVar(&onlyHost, "open", false, "Show Output as format 'IP:Port' only")
	flag.BoolVar(&omitEmpty, "omitEmpty", true, "Only provide output if an entry exists. Default true")
	flag.Parse()

	log.SetOutput(os.Stderr)

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Strings(args)
		ip := args[len(args)-1]
		StartJob(ip)
		os.Exit(0)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {

				StartJob(job)
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			url := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && url != "" {
				jobs <- url
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func StartJob(raw string) {
	_, _, err := net.ParseCIDR(raw)
	if err != nil {
		GetIPInfo(raw)
		return
	}

	if ips, err := mapcidr.IPAddresses(raw); err == nil {
		for _, ip := range ips {
			GetIPInfo(ip)
		}
	}
}

func GetIPInfo(IP string) {
	data := sendGET(IP)
	if data == "" {
		return
	}

	if omitEmpty && strings.Contains(data, "No information available") {
		log.Printf("No info available for %s", IP)
		return
	}

	if jsonOutput {
		fmt.Println(data)
		if !servicesOutput {
			return
		}
	}

	var shodanIPInfo ShodanIPInfo
	if ok := jsoniter.Unmarshal([]byte(data), &shodanIPInfo); ok != nil {
		return
	}

	if servicesOutput {
		for _, port := range shodanIPInfo.Ports {
			Services = append(Services, Service{
				IP:       shodanIPInfo.IP,
				Protocol: "tcp",
				Port:     string(rune(port)),
				Service:  " ",
			})
		}
	}

	if csvOutput {
		for _, port := range shodanIPInfo.Ports {
			line := fmt.Sprintf("%s:%d", IP, port)
			if onlyHost {
				fmt.Println(line)
				continue
			}

			line = fmt.Sprintf("%s,%s,%s,%s,%s", line, strings.Join(shodanIPInfo.Hostnames, ";"), strings.Join(shodanIPInfo.Tags, ";"), strings.Join(shodanIPInfo.Cpes, ";"), strings.Join(shodanIPInfo.Vulns, ";"))
			fmt.Println(line)
		}
	}
}

func sendGET(IP string) string {
	ipURL := fmt.Sprintf("https://internetdb.shodan.io/%s", IP)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Get(ipURL)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "%v", err)
		log.Printf("%v\n", err)
		return ""
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("Too many requests, sleeping: %s\n", IP)
		time.Sleep(10 * time.Second)
		resp, err = client.Get(ipURL)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return ""
	}
	return string(body)
}
