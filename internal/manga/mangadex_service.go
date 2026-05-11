package manga

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MangaDexResponse parses the MangaDex API /manga response including relationships
type MangaDexResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string `json:"title"`
			Description map[string]string `json:"description"`
			Status      string            `json:"status"`
			Year        int               `json:"year"`
			Tags        []struct {
				Attributes struct {
					Name  map[string]string `json:"name"`
					Group string            `json:"group"`
				} `json:"attributes"`
			} `json:"tags"`
		} `json:"attributes"`
		Relationships []struct {
			Type       string `json:"type"`
			Attributes *struct {
				Name string `json:"name"`
			} `json:"attributes,omitempty"`
		} `json:"relationships"`
	} `json:"data"`
}

func (h *Handler) FetchFromMangaDex() {
	client := &http.Client{Timeout: 60 * time.Second}

	fmt.Println(">>> Connecting to MangaDex API...")

	// Request with includes[]=author to get real author names in relationships
	url := "https://api.mangadex.org/manga?limit=100&includes[]=author&includes[]=artist&availableTranslatedLanguage[]=en"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("  MangaDex: request creation failed: %v\n", err)
		return
	}

	req.Header.Set("User-Agent", "MangaHub/1.0 (academic project)")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  MangaDex: connection failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("  MangaDex: API returned status %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  MangaDex: failed to read body: %v\n", err)
		return
	}

	var result MangaDexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("  MangaDex: JSON decode error: %v\n", err)
		return
	}

	query := `INSERT OR IGNORE INTO manga (id, title, author, genres, status, total_chapters, description, year, rating, popularity)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	count := 0
	for _, m := range result.Data {
		// --- Title: prefer English, fallback to first available ---
		title := m.Attributes.Title["en"]
		if title == "" {
			for _, v := range m.Attributes.Title {
				title = v
				break
			}
		}
		if title == "" {
			continue
		}

		// --- Author: extracted from relationships ---
		author := "Unknown Author"
		for _, rel := range m.Relationships {
			if (rel.Type == "author" || rel.Type == "artist") && rel.Attributes != nil && rel.Attributes.Name != "" {
				author = rel.Attributes.Name
				break
			}
		}

		// --- Genres: extracted from tags (group = "genre") ---
		var genreList []string
		for _, tag := range m.Attributes.Tags {
			if tag.Attributes.Group == "genre" {
				if name, ok := tag.Attributes.Name["en"]; ok && name != "" {
					genreList = append(genreList, name)
				}
			}
		}
		if len(genreList) == 0 {
			genreList = []string{"Action"}
		}
		genresBytes, _ := json.Marshal(genreList)

		// --- Status ---
		status := m.Attributes.Status
		if status == "" {
			status = "ongoing"
		}

		// --- Description ---
		description := m.Attributes.Description["en"]
		if description == "" {
			description = "No description available."
		}
		// Trim very long descriptions
		if len(description) > 500 {
			description = description[:500] + "..."
		}
		// Strip markdown-style links like [link](url)
		description = strings.ReplaceAll(description, "\r\n", " ")
		description = strings.ReplaceAll(description, "\n", " ")

		// --- Year ---
		year := m.Attributes.Year
		if year == 0 {
			year = 2000
		}

		_, err := h.DB.Exec(query, m.ID, title, author, string(genresBytes), status, 0, description, year, 0.0, 0)
		if err != nil {
			fmt.Printf("  MangaDex: error saving '%s': %v\n", title, err)
		} else {
			count++
		}
	}

	fmt.Printf("  MangaDex: successfully added %d manga to database.\n", count)
}
