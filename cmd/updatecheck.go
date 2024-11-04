package cmd

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Feed represents the structure of the Atom feed
type Feed struct {
	Entries []Entry `xml:"entry"`
}

// Entry represents a single commit entry in the Atom feed
type Entry struct {
	Updated string `xml:"updated"`
	Link    string `xml:"link,attr"`
}

// updatecheckCmd represents the updatecheck command
var updatecheckCmd = &cobra.Command{
	Use:   "updatecheck",
	Short: "Check for today's commits in specified GitHub repositories",
	Long: `This command checks for today's commits in the specified GitHub repositories.
You can provide one or more URLs that point to the .zip files of the branches.
It will transform the URLs to check for commits and return the original URLs if any commits match today's date.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Process each provided URL
		for _, url := range args {
			checkURL(url)
		}

		// Optionally, also check for URLs from standard input
		if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeNamedPipe) != 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				checkURL(scanner.Text())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(updatecheckCmd)
}

func checkURL(url string) {
	// Check if the URL ends with /main.zip or /master.zip
	if strings.HasSuffix(url, "/main.zip") || strings.HasSuffix(url, "/master.zip") {
		// Transform the URL
		transformedURL := transformURL(url)
		if transformedURL != "" {
			// Fetch the Atom feed and check for today's commits
			if matchCommits(transformedURL) {
				fmt.Println("Updated today:", url) // Print the original URL if there's a match
			}
		}
	}
}

// transformURL transforms the input URL as specified
func transformURL(url string) string {
	// Replace 'archive/refs/heads' with 'commits' and replace 'main.zip' or 'master.zip' with '.atom'
	url = strings.Replace(url, "archive/refs/heads", "commits", 1)
	if strings.HasSuffix(url, "/main.zip") {
		return strings.TrimSuffix(url, "/main.zip") + ".atom"
	} else if strings.HasSuffix(url, "/master.zip") {
		return strings.TrimSuffix(url, "/master.zip") + ".atom"
	}
	return ""
}

// matchCommits checks if there are any commits matching today's date
func matchCommits(url string) bool {
	today := time.Now().Format("2006-01-02") // Get today's date

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch URL %s: %v\n", url, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: received status code %d for URL %s\n", resp.StatusCode, url)
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response body: %v\n", err)
		return false
	}

	var feed Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse XML: %v\n", err)
		return false
	}

	for _, entry := range feed.Entries {
		if strings.HasPrefix(entry.Updated, today) {
			return true
		}
	}

	return false
}
