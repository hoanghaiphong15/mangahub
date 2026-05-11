package manga

import (
	"fmt"

	"github.com/gocolly/colly/v2"
)

// PracticeScraping performs educational web scraping as required by the project spec
func (h *Handler) PracticeScraping() {
	// Initialize a new collector
	c := colly.NewCollector()

	// Set up logic to extract data when a ".quote" element is found
	c.OnHTML(".quote", func(e *colly.HTMLElement) {
		quote := e.ChildText(".text")
		author := e.ChildText(".author")

		// Print the output to the terminal (this generates the required logs)
		fmt.Printf("--- [Scraping Practice] Quote: %s - By: %s\n", quote, author)
	})

	fmt.Println(">>> Starting Educational Scraping Practice on quotes.toscrape.com...")

	// Start the scraping process
	err := c.Visit("http://quotes.toscrape.com/")
	if err != nil {
		fmt.Printf("Error during scraping: %v\n", err)
	}
}
