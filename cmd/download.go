package cmd

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// Define variables for directory, parallel, and depth flags
var directory string
var parallel int
var depth int
var keepZip bool

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Clone or download repositories and files with various options",
	Long: `Clone Git repositories, download YAML and ZIP files, with customization options.

Examples:
  echo "https://github.com/rix4uni/nucleihub.git" | nucleihub download
  cat reponames.txt | nucleihub download
  cat reponames.txt | nucleihub download -o ~/nucleihub-downloaded-repos
  cat reponames.txt | nucleihub download -p 10
  cat reponames.txt | nucleihub download -d 0`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create a scanner to read from stdin
		scanner := bufio.NewScanner(os.Stdin)

		// Collect all repository URLs from stdin
		var repoURLs []string
		for scanner.Scan() {
			repoURLs = append(repoURLs, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading input:", err)
			return
		}

		// Create a semaphore channel to limit concurrent operations
		sem := make(chan struct{}, parallel)
		var wg sync.WaitGroup

		for _, url := range repoURLs {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()

				// Acquire semaphore
				sem <- struct{}{}
				defer func() { <-sem }() // Release semaphore when done

				// Create a directory name based on the URL
				var dirName string
				if strings.HasPrefix(url, "https://github.com/") {
					// Handle GitHub repository URLs
					parts := strings.Split(url[len("https://github.com/"):], "/")
					if len(parts) >= 2 {
						username := parts[0]
						repoName := strings.TrimSuffix(parts[1], ".git") // Get the repository name
						dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
					}
				} else if strings.HasPrefix(url, "https://raw.githubusercontent.com/") {
					// Handle raw GitHub file URLs
					parts := strings.Split(url[len("https://raw.githubusercontent.com/"):], "/")
					if len(parts) >= 3 {
						username := parts[0] // Username is the first part
						repoName := parts[1] // Repository name is the second part
						dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
					}
				} else if strings.HasPrefix(url, "https://gist.githubusercontent.com/") {
					// Handle raw GitHub file URLs
					parts := strings.Split(url[len("https://gist.githubusercontent.com/"):], "/")
					if len(parts) >= 3 {
						username := parts[0] // Username is the first part
						repoName := parts[1] // Repository name is the second part
						dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
					}
				}

				// Check if the directory already exists and remove it if so
				if _, err := os.Stat(dirName); !os.IsNotExist(err) {
					fmt.Printf("Directory %s already exists, removing it...\n", dirName)
					err := os.RemoveAll(dirName)
					if err != nil {
						fmt.Printf("Failed to remove existing directory: %s\n", err)
						return
					}
				}

				// Determine action based on file extension
				switch {
				case strings.HasSuffix(url, ".git"):
					// Clone Git repository
					cloneArgs := []string{"clone", url, dirName}
					if depth > 0 {
						cloneArgs = append(cloneArgs, "--depth", fmt.Sprintf("%d", depth))
					}
					cloneCmd := exec.Command("git", cloneArgs...)
					var stderr bytes.Buffer
					cloneCmd.Stderr = &stderr

					if err := cloneCmd.Run(); err != nil {
						if strings.Contains(stderr.String(), "Repository not found") {
							fmt.Printf("[NOT FOUND] %s\n", url)
						} else {
							fmt.Printf("Error cloning repository %s: %s\n", url, stderr.String())
						}
					} else {
						fmt.Printf("[CLONED] %s into %s\n", url, dirName)
					}

				case strings.HasSuffix(url, ".yaml"), strings.HasSuffix(url, ".zip"):
					// Download file directly with Go's HTTP client
					if err := downloadFile(url, dirName); err != nil {
						fmt.Printf("Error downloading file %s: %v\n", url, err)
					} else {
						fmt.Printf("[DOWNLOADED] %s into %s\n", url, dirName)
					}

					// Unzip the downloaded file if it's a zip
					if strings.HasSuffix(url, ".zip") {
						if err := unzip(filepath.Join(dirName, filepath.Base(url)), dirName); err != nil {
							fmt.Printf("Error unzipping file %s: %v\n", url, err)
						} else {
							fmt.Printf("[UNZIPPED] %s into %s\n", url, dirName)
						}

						// Check the keepZip flag before deleting the .zip file
						if !keepZip {
							zipFilePath := filepath.Join(dirName, filepath.Base(url))
							if err := os.Remove(zipFilePath); err != nil {
								fmt.Printf("Failed to delete .zip file: %v\n", err)
							} else {
								fmt.Printf("[DELETED] %s\n", zipFilePath) // Updated to show the full path
							}
						}
					}

				default:
					fmt.Printf("Unsupported URL format: %s\n", url)
				}
			}(url)
		}

		// Wait for all operations to complete
		wg.Wait()
	},
}

// downloadFile downloads a file from the given URL and saves it to the specified directory
func downloadFile(url, dirName string) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Extract file name from URL
	fileName := filepath.Join(dirName, filepath.Base(url))

	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	// Get the file from URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %s", resp.Status)
	}

	// Write the response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}

// unzip extracts a zip file to a destination folder concurrently
func unzip(zipPath, destFolder string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure destination folder exists
	if err := os.MkdirAll(destFolder, 0755); err != nil {
		return err
	}

	// Wait group for concurrent extraction
	var wg sync.WaitGroup

	for _, f := range r.File {
		wg.Add(1)

		go func(f *zip.File) {
			defer wg.Done()
			if err := extractFile(f, destFolder); err != nil {
				fmt.Printf("Failed to extract %s: %v\n", f.Name, err)
			}
		}(f)
	}

	wg.Wait()
	return nil
}

// extractFile extracts a single file or directory from the zip archive
func extractFile(f *zip.File, destFolder string) error {
	// Create the full file path
	destPath := filepath.Join(destFolder, f.Name)

	// Handle directories
	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, f.Mode())
	}

	// Create all parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Open the file inside the zip archive
	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents from the zip archive to the destination file
	_, err = io.Copy(destFile, srcFile)
	return err
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&directory, "output-directory", "o", filepath.Join(os.Getenv("HOME"), "nucleihub-downloaded-repos"), "Directory to clone or download into")
	downloadCmd.Flags().IntVarP(&parallel, "parallel", "p", 10, "Number of operations to perform in parallel")
	downloadCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Create a shallow clone with a history truncated to the specified number of commits, use -d 0 if you want all commits")
	downloadCmd.Flags().BoolVar(&keepZip, "keepzip", false, "Keep the downloaded .zip file (default: delete it)")
}
