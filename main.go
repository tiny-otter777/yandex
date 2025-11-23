package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Parse()
	consecErrors := 0
	for {
		err := pollOnce(*urlFlag)
		if err != nil {
			consecErrors++
			if consecErrors >= 3 {
				fmt.Println("Unable to fetch server statistic.")
			}
		} else {
			consecErrors = 0
		}
		time.Sleep(*period)
	}
}

var (
	urlFlag = flag.String("url", "http://srv.msk01.gigacorp.local/_stats", "URL to fetch stats from (GET)")
	period  = flag.Duration("interval", time.Second, "poll interval (e.g. 1s, 500ms)")
	client  = &http.Client{Timeout: 5 * time.Second}
)

func pollOnce(url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	body, err := readAllAsString(resp.Body)
	if err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	parts := strings.Split(body, ",")
	if len(parts) != 7 {
		return fmt.Errorf("bad format: expected 7 fields, got %d", len(parts))
	}

	loadVal, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return fmt.Errorf("parse load: %w", err)
	}
	totalMem, err := parseInt64(parts[1])
	if err != nil {
		return fmt.Errorf("parse totalMem: %w", err)
	}
	usedMem, err := parseInt64(parts[2])
	if err != nil {
		return fmt.Errorf("parse usedMem: %w", err)
	}
	totalDisk, err := parseInt64(parts[3])
	if err != nil {
		return fmt.Errorf("parse totalDisk: %w", err)
	}
	usedDisk, err := parseInt64(parts[4])
	if err != nil {
		return fmt.Errorf("parse usedDisk: %w", err)
	}
	totalNet, err := parseInt64(parts[5])
	if err != nil {
		return fmt.Errorf("parse totalNet: %w", err)
	}
	usedNet, err := parseInt64(parts[6])
	if err != nil {
		return fmt.Errorf("parse usedNet: %w", err)
	}

	if loadVal > 30 {
		if math.Mod(loadVal, 1) == 0 {
			fmt.Printf("Load Average is too high: %d\n", int64(loadVal))
		} else {
			fmt.Printf("Load Average is too high: %.2f\n", loadVal)
		}
	}

	if totalMem <= 0 {
		return fmt.Errorf("invalid total memory (0)")
	}
	memPercent := int((usedMem * 100) / totalMem)
	if memPercent > 80 {
		fmt.Printf("Memory usage too high: %d%%\n", memPercent)
	}

	if totalDisk <= 0 {
		return fmt.Errorf("invalid total disk (0)")
	}
	if float64(usedDisk) > float64(totalDisk)*0.9 {
		freeMB := (totalDisk - usedDisk) / (1024 * 1024)
		if freeMB < 0 {
			freeMB = 0
		}
		fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
	}

	if totalNet <= 0 {
		return fmt.Errorf("invalid total network bandwidth (0)")
	}
	if float64(usedNet) > float64(totalNet)*0.9 {
		availBytesPerSec := totalNet - usedNet
		if availBytesPerSec < 0 {
			availBytesPerSec = 0
		}
		availMbit := int(availBytesPerSec / 1_000_000)
		fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availMbit)
	}

	return nil
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func readAllAsString(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, 0, 1024)
	scanner.Buffer(buf, maxCapacity)
	var b strings.Builder
	first := true
	for scanner.Scan() {
		if !first {
			b.WriteByte('\n')
		}
		b.WriteString(scanner.Text())
		first = false
	}
	return b.String(), scanner.Err()
}
