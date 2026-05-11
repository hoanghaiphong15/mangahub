package manga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Struct to match the response from MangaDex API (simplified for our needs)
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

// conversion from handler.go
func (h *Handler) FetchFromMangaDex() {

	// Force the use of IPv4 only (tcp4)
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		// Completely disable Proxy from the environment to prevent connection errors
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Enforce IPv4 by overriding the network type to "tcp4"
			return dialer.DialContext(ctx, "tcp4", addr)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	fmt.Println(">>> HARD FORCING IPv4 for MangaDex API...")

	// 2. using the custom client with IPv4 forced to call MangaDex API to get manga data
	resp, err := client.Get("https://api.mangadex.org/manga?limit=100")
	if err != nil {
		fmt.Printf("Error API call: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check if status code is 200
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("MangaDex API returned non-OK status: %d\n", resp.StatusCode)
		return
	}

	// // call MangaDex API to get manga data
	// resp, err := http.Get("https://api.mangadex.org/manga?limit=100")
	// if err != nil {
	// 	fmt.Println("Error API call:", err)
	// 	return
	// }
	// defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result MangaDexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("JSON decoding error :", err)
		return
	}

	// prepare SQL statement for inserting/updating manga data
	query := `INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, year, rating, popularity) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	count := 0
	for _, m := range result.Data {
		// take the English title if available, otherwise take any available title
		title := m.Attributes.Title["en"]
		if title == "" {
			for _, v := range m.Attributes.Title {
				title = v
				break
			}
		}

		// Since the API doesn't provide author and genres, we will use placeholders for now
		author := "Unknown Author"
		genres := "[\"Action\", \"API fetched\"]" // make it a JSON string to match our DB schema
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
		} // default to current year if not provided

		// 3. Execute into database
		_, err := h.DB.Exec(query,
			m.ID,
			title,
			author,
			genres,
			status,
			0, // total_chapters default 0 because API doesn't provide it, can be updated later
			description,
			year,
			0.0, // rating default
			0,   // popularity default
		)

		if err != nil {
			fmt.Printf("Error save %s: %v\n", title, err)
		} else {
			count++
		}
	}

	fmt.Printf("--- Successful: adding to %d  manga from API MangaDex! ---\n", count)
}
