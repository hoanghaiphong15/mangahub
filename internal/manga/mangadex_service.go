package manga

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type MangaDexResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string `json:"title"`
			Description map[string]string `json:"description"`
			Status      string            `json:"status"`
			Year        int               `json:"year"`
		} `json:"attributes"`
	} `json:"data"`
}

func (h *Handler) FetchFromMangaDex() {
	// 1. HTTP client with custom transport to mimic browser behavior and handle SSL properly
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Helpful for debugging SSL issues, but in production, you should verify certificates properly
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         "api.mangadex.org",
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	fmt.Println(">>> [WARP MODE] Connecting to MangaDex API with Browser Headers...")

	// 2. Create the API request with headers that mimic a real browser
	req, err := http.NewRequest("GET", "https://api.mangadex.org/manga?limit=100", nil)
	if err != nil {
		fmt.Printf(" Request creation failed: %v\n", err)
		return
	}

	// Bypass Cloudflare and appear as a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://mangadex.org/")

	// 3. Execute the request and handle the response
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(" Connection Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf(" API returned error status: %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf(" Failed to read body: %v\n", err)
		return
	}

	var result MangaDexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println(" JSON decoding error:", err)
		return
	}

	// 4. Save the fetched manga data into the database
	query := `INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, year, rating, popularity) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	count := 0
	for _, m := range result.Data {
		title := m.Attributes.Title["en"]
		if title == "" {
			for _, v := range m.Attributes.Title {
				title = v
				break
			}
		}

		author := "Unknown Author"
		genres := `["Action", "API fetched"]`
		status := m.Attributes.Status
		if status == "" {
			status = "ongoing"
		}

		description := m.Attributes.Description["en"]
		if description == "" {
			description = "No description available."
		}

		year := m.Attributes.Year
		if year == 0 {
			year = 2026
		}

		_, err := h.DB.Exec(query, m.ID, title, author, genres, status, 0, description, year, 0.0, 0)
		if err != nil {
			fmt.Printf("Error saving %s: %v\n", title, err)
		} else {
			count++
		}
	}

	fmt.Printf("\n --- SUCCESS --- \nSuccessfully added %d manga to database via WARP!\n", count)
}
