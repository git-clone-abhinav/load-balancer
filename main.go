package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	twoqueue "github.com/floatdrop/2q"
	godotenv "github.com/joho/godotenv"
)

var (
	RPCs        []string
	FallbackRPC []string

	cache    *twoqueue.TwoQueue[string, bool]
	cacheTTL = time.Minute
	mu       sync.Mutex
	port     string
	slackURL string
)

func init() {
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalln("Error loading .env file: %v", err)
		}
	}

	RPCs = strings.Split(os.Getenv("RPCs"), ",")
	FallbackRPC = strings.Split(os.Getenv("FALLBACK_RPCs"), ",")

	port = os.Getenv("PORT")

	duration, err := strconv.Atoi(os.Getenv("ERROR_TIME_TO_LIVE_MINUTES"))
	if err != nil {
		log.Fatalf("Invalid duration for ERROR_TIME_TO_LIVE_MINUTES: %v", err)
	}
	cacheTTL = time.Duration(duration) * time.Minute
	fmt.Println("Cache TTL:", cacheTTL)

	if len(RPCs) == 0 {
		log.Fatal("No RPCs provided")
	}
	if len(FallbackRPC) == 0 {
		log.Fatal("No fallback RPC provided")
	}

	slackURL = os.Getenv("SLACK_WEBHOOK_URL")
	if slackURL == "" {
		log.Fatal("Slack webhook URL is not set")
	}

	fmt.Println("RPCs:", RPCs)
	fmt.Println("Fallback RPC:", FallbackRPC)
	fmt.Println("Slack URL:", slackURL)
}

// main initializes the cache and starts the HTTP server with the load balancer.
func main() {
	cache = twoqueue.New[string, bool](10)

	http.HandleFunc("/", loadBalancer)

	if port == "" {
		port = "8080"
	}
	log.Printf("Load balancer started on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// loadBalancer handles incoming HTTP requests and attempts to forward them to primary or fallback RPCs.
func loadBalancer(w http.ResponseWriter, r *http.Request) {
	if !tryForwardRequests(w, r, RPCs) {
		sendNotificationToSlack("WARNING : All RPCs are reaching their ratelimits.")
		if !tryForwardRequests(w, r, FallbackRPC) {
			sendNotificationToSlack("FATAL : Even fallback RPCs are reaching their ratelimits.")
			http.Error(w, "All RPCs are reaching their ratelimits.", http.StatusInternalServerError)
		}
	}
}

// tryForwardRequests attempts to forward the request to a list of URLs and returns true if successful.
func tryForwardRequests(w http.ResponseWriter, r *http.Request, urls []string) bool {
	urls = shuffleURLs(urls)

	for _, url := range urls {
		if found := cache.Get(url); found != nil {
			continue
		}

		resp, err := forwardRequest(url, r)
		if err != nil {
			log.Printf("Request to %s failed: %v", url, err)
			addToCacheWithTTL(url, cacheTTL)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read response body", http.StatusInternalServerError)
				return true
			}
			w.WriteHeader(resp.StatusCode)
			w.Write(body)
			fmt.Println(resp.StatusCode, url)
			return true
		} else {
			addToCacheWithTTL(url, cacheTTL)
		}
	}

	return false
}

// sendNotificationToSlack sends a notification message to Slack.
func sendNotificationToSlack(message string) {
	payload := `{
		"attachments":[
			{
				"fallback":"LoadBalancer > ` + message + `",
				"pretext":"LoadBalancer > ` + message + `",
				"color":"#FF0000",
				"fields":[
					{
						"title":"Load Balancer Error",
						"value":"All RPCs are reaching their ratelimits, consider increasing the number of RPCs or the rate limit for each one.",
						"short":false
					}
				]
			}
		]
	}`
	fmt.Println("Slack url : ", slackURL)
	req, err := http.NewRequest("POST", slackURL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request to Slack: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Slack notification failed with status %d: %s", resp.StatusCode, string(body))
	} else {
		log.Println("Slack notification sent successfully")
	}
}

// forwardRequest forwards the HTTP request to the specified URL and returns the response.
func forwardRequest(url string, r *http.Request) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, url+r.URL.Path, r.Body)
	if err != nil {
		return nil, err
	}

	req.Header = r.Header
	client := &http.Client{}
	return client.Do(req)
}

// shuffleURLs shuffles the order of the URLs in the provided slice.
func shuffleURLs(urls []string) []string {
	shuffled := make([]string, len(urls))
	copy(shuffled, urls)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

// addToCacheWithTTL adds a URL to the cache with a specified TTL (time-to-live).
func addToCacheWithTTL(url string, ttl time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	cache.Set(url, true)
	time.AfterFunc(ttl, func() {
		mu.Lock()
		defer mu.Unlock()
		cache.Remove(url)
	})
}
