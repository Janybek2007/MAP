package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var osArgs = func() []string {
	return os.Args[1:]
}

func shutdownOurServer(port int) error {
	client := &http.Client{Timeout: 600 * time.Millisecond}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/shutdown", port), nil)
	if err != nil {
		return err
	}
	request.Header.Set(serverMarkerHeader, "1")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	_ = response.Body.Close()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = listener.Close()
			return nil
		}
		time.Sleep(120 * time.Millisecond)
	}

	return nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("неподдерживаемая платформа: %s", runtime.GOOS)
	}
}

func isOurServer(port int) bool {
	client := &http.Client{Timeout: 350 * time.Millisecond}
	response, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return false
	}

	if response.Header.Get(serverMarkerHeader) != "1" {
		return false
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return false
	}

	return payload["status"] == "ok"
}

func waitForServerReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 350 * time.Millisecond}
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(120 * time.Millisecond)
	}
	return false
}

func waitForHTTP200(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(120 * time.Millisecond)
	}
	return false
}
