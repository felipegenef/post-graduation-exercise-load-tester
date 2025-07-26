package cmd

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	url         string
	totalReqs   int
	concurrency int
)

var rootCmd = &cobra.Command{
	Use:   "loadtester",
	Short: "Go Expert Exam load tester. Use --url --requests and --concurrency to execute the command.",
	Run: func(cmd *cobra.Command, args []string) {
		runLoadTest()
	},
}

// Prints a bold, centered ASCII banner in cyan (Go) color
func printBanner() {
	banner := `

 ██████╗  ██████╗     ███████╗██╗  ██╗██████╗ ███████╗██████╗ ████████╗
██╔════╝ ██╔═══██╗    ██╔════╝╚██╗██╔╝██╔══██╗██╔════╝██╔══██╗╚══██╔══╝
██║  ███╗██║   ██║    █████╗   ╚███╔╝ ██████╔╝█████╗  ██████╔╝   ██║   
██║   ██║██║   ██║    ██╔══╝   ██╔██╗ ██╔═══╝ ██╔══╝  ██╔══██╗   ██║   
╚██████╔╝╚██████╔╝    ███████╗██╔╝ ██╗██║     ███████╗██║  ██║   ██║   
 ╚═════╝  ╚═════╝     ╚══════╝╚═╝  ╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝   ╚═╝   
`
	cyan := color.New(color.FgCyan).Add(color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	cyan.Println(banner)
	fmt.Println("                      Go Expert Load Tester 🚀")
	fmt.Printf("🪛  %s\n", green.Sprint("Execution parameters:"))
	fmt.Printf("🔗 URL: %s\n", yellow.Sprint(url))
	fmt.Printf("📦 Total requests: %s\n", yellow.Sprintf("%d", totalReqs))
	fmt.Printf("🔀 Concurrency: %s\n", yellow.Sprintf("%d", concurrency))
}

func Execute() {
	rootCmd.Flags().StringVar(&url, "url", "", "URL of the service to be tested")
	rootCmd.Flags().IntVar(&totalReqs, "requests", 100, "Total number of requests")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 10, "Number of simultaneous calls")

	rootCmd.MarkFlagRequired("url")
	rootCmd.MarkFlagRequired("requests")
	rootCmd.MarkFlagRequired("concurrency")

	rootCmd.Execute()
}

func runLoadTest() {
	printBanner()
	start := time.Now()

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	statusCodes := make(map[int]int)
	responseTimes := []float64{}
	mu := sync.Mutex{}

	// Create progress bar
	bar := progressbar.NewOptions(totalReqs,
		progressbar.OptionSetDescription("🔄 Executing requests..."),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	for i := 0; i < totalReqs; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			reqStart := time.Now()
			resp, err := http.Get(url)
			duration := time.Since(reqStart).Seconds() * 1000

			mu.Lock()
			responseTimes = append(responseTimes, duration)
			if err != nil {
				statusCodes[0]++
			} else {
				defer resp.Body.Close()
				statusCodes[resp.StatusCode]++
			}
			bar.Add(1)
			mu.Unlock()
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate statistics
	avg, min, max, p90, p95, p99 := calcStats(responseTimes)

	// FINAL REPORT
	fmt.Println("\n📊 Test Report:")
	fmt.Printf("🔗 URL: %s\n", url)
	fmt.Printf("📈 Total requests: %d\n", totalReqs)
	fmt.Printf("⏱️  Total time: %v\n", totalDuration)
	fmt.Printf("🧮 Average time: %.2f ms\n", avg)
	fmt.Printf("🔻 Min time: %.2f ms\n", min)
	fmt.Printf("🔺 Max time: %.2f ms\n", max)
	fmt.Printf("🎯 P90: %.2f ms | P95: %.2f ms | P99: %.2f ms\n", p90, p95, p99)

	// Show status codes
	fmt.Println("\n📦 Status Codes:")
	onlySuccess := true
	for code := range statusCodes {
		if code < 200 || code >= 300 {
			onlySuccess = false
			break
		}
	}

	if onlySuccess {
		fmt.Println(color.New(color.FgGreen, color.Bold).Sprint("✅ All requests were successful (2xx)"))
	} else {
		for code, count := range statusCodes {
			colored := statusColor(code)
			fmt.Println(colored(fmt.Sprintf("  %d: %d", code, count)))
		}
	}
	// Novo log para status 0
	if countZero, exists := statusCodes[0]; exists && countZero > 0 {
		fmt.Println(color.New(color.FgRed, color.Bold).Sprint(
			fmt.Sprintf("❌ %d requests could not finish (no HTTP response)", countZero),
		))
	}
}

func calcStats(times []float64) (avg, min, max, p90, p95, p99 float64) {
	if len(times) == 0 {
		return
	}
	sort.Float64s(times)

	sum := 0.0
	min = times[0]
	max = times[len(times)-1]

	for _, t := range times {
		sum += t
	}
	avg = sum / float64(len(times))

	getPercentile := func(p float64) float64 {
		index := int(float64(len(times))*p) - 1
		if index < 0 {
			index = 0
		}
		if index >= len(times) {
			index = len(times) - 1
		}
		return times[index]
	}

	p90 = getPercentile(0.90)
	p95 = getPercentile(0.95)
	p99 = getPercentile(0.99)

	return
}

func statusColor(code int) func(a ...interface{}) string {
	switch {
	case code >= 200 && code < 300:
		return color.New(color.FgGreen).SprintFunc()
	case code >= 300 && code < 400:
		return color.New(color.FgYellow).SprintFunc()
	case code >= 400 && code < 500:
		return color.New(color.FgHiYellow).SprintFunc()
	case code >= 500:
		return color.New(color.FgRed).SprintFunc()
	default:
		return color.New(color.FgHiBlack).SprintFunc()
	}
}
