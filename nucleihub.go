package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/rix4uni/nucleihub/banner"
	"github.com/spf13/pflag"
)

// globalYAMLFiles tracks YAML files in the global output directory (filename -> size)
var globalYAMLFiles = make(map[string]int64)
var globalYAMLMutex sync.Mutex

// Colorized output helpers
var (
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
)

// hashSuffixRegex matches YAML files with 32-char hex hash suffixes (e.g., file-d41d8cd98f00b204e9800998ecf8427e.yaml)
var hashSuffixRegex = regexp.MustCompile(`-[a-f0-9]{32}\.yaml$`)

// numericSuffixRegex matches YAML files with numeric ID suffixes (e.g., file-23.yaml, file-123.yaml)
var numericSuffixRegex = regexp.MustCompile(`-[0-9]+\.yaml$`)

func main() {
	// Define flags with pflag
	outputDir := pflag.StringP("output-directory", "o", filepath.Join(os.Getenv("HOME"), "nucleihub-templates"), "Directory to download into")
	parallel := pflag.IntP("parallel", "p", 10, "Number of operations to perform in parallel")
	noValidate := pflag.Bool("no-validate", false, "Skip post-download nuclei validation")
	silent := pflag.Bool("silent", false, "Silent mode.")
	version := pflag.Bool("version", false, "Print the version of the tool and exit.")
	pflag.Parse()

	if *version {
        banner.PrintBanner()
        banner.PrintVersion()
        return
    }

    if !*silent {
        banner.PrintBanner()
    }

	// Read URLs from stdin
	urls := readURLsFromStdin()
	if len(urls) == 0 {
		fmt.Println("No URLs provided. Pipe URLs via stdin.")
		fmt.Println("Example: cat reponames.txt | nucleihub")
		return
	}

	// Run download logic
	downloadAll(urls, *outputDir, *parallel)

	// Post-download validation
	if !*noValidate {
		validateAndCleanTemplates(*outputDir)
	}
}

// readURLsFromStdin reads all URLs from stdin
func readURLsFromStdin() []string {
	scanner := bufio.NewScanner(os.Stdin)
	var urls []string
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
	return urls
}

// downloadAll processes all URLs concurrently
func downloadAll(urls []string, directory string, parallel int) {
	// Create a semaphore channel to limit concurrent operations
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore when done

			processURL(url, directory)
		}(url)
	}

	// Wait for all operations to complete
	wg.Wait()
}

// processURL handles a single URL (git clone, yaml download, or zip download)
func processURL(url, directory string) {
	// Create a directory name based on the URL
	var dirName string
	if strings.HasPrefix(url, "https://github.com/") {
		// Handle GitHub repository URLs
		parts := strings.Split(url[len("https://github.com/"):], "/")
		if len(parts) >= 2 {
			username := parts[0]
			repoName := strings.TrimSuffix(parts[1], ".git")
			dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
		}
	} else if strings.HasPrefix(url, "https://raw.githubusercontent.com/") {
		// Handle raw GitHub file URLs
		parts := strings.Split(url[len("https://raw.githubusercontent.com/"):], "/")
		if len(parts) >= 3 {
			username := parts[0]
			repoName := parts[1]
			dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
		}
	} else if strings.HasPrefix(url, "https://gist.githubusercontent.com/") {
		// Handle gist URLs
		parts := strings.Split(url[len("https://gist.githubusercontent.com/"):], "/")
		if len(parts) >= 3 {
			username := parts[0]
			repoName := parts[1]
			dirName = filepath.Join(directory, fmt.Sprintf("%s-%s", username, repoName))
		}
	}

	// Check if the directory already exists and remove it if so
	if _, err := os.Stat(dirName); !os.IsNotExist(err) {
		os.RemoveAll(dirName)
	}

	// Determine action based on file extension
	switch {
	case strings.HasSuffix(url, ".git"):
		// Clone Git repository (disable interactive prompts for private repos)
		cloneArgs := []string{"clone", url, dirName, "--depth", "1", "--single-branch", "--no-tags", "--no-recurse-submodules"}
		cloneCmd := exec.Command("git", cloneArgs...)
		cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		if err := cloneCmd.Run(); err != nil {
			fmt.Printf("%s %s\n", red("[FAILED]"), url)
		} else {
			fmt.Printf("%s %s\n", cyan("[CLONED]"), url)
			flattenAndMoveYAMLFiles(dirName, directory)
		}

	case strings.HasSuffix(url, ".yaml"), strings.HasSuffix(url, ".zip"):
		// Download file directly with Go's HTTP client
		if err := downloadFile(url, dirName); err != nil {
			fmt.Printf("%s %s\n", red("[FAILED]"), url)
		} else {
			fmt.Printf("%s %s\n", green("[DOWNLOADED]"), url)
		}

		// Unzip and flatten if it's a zip
		if strings.HasSuffix(url, ".zip") {
			unzip(filepath.Join(dirName, filepath.Base(url)), dirName)
			os.Remove(filepath.Join(dirName, filepath.Base(url)))
		}

		// Flatten and move YAML files to global directory (for both .yaml and .zip)
		if strings.HasSuffix(url, ".yaml") || strings.HasSuffix(url, ".zip") {
			flattenAndMoveYAMLFiles(dirName, directory)
		}

	default:
		fmt.Printf("%s %s\n", red("[FAILED]"), url)
	}
}

// downloadFile downloads a file from the given URL and saves it to the specified directory
func downloadFile(url, dirName string) error {
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	fileName := filepath.Join(dirName, filepath.Base(url))

	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %s", resp.Status)
	}

	// Write the response body to file with 32KB buffer
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(out, resp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}

// unzip extracts a zip file to a destination folder with 2 concurrent workers
func unzip(zipPath, destFolder string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destFolder, 0755); err != nil {
		return err
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 2)

	for _, f := range r.File {
		wg.Add(1)
		go func(f *zip.File) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			extractFile(f, destFolder)
		}(f)
	}

	wg.Wait()
	return nil
}

// extractFile extracts a single file or directory from the zip archive
func extractFile(f *zip.File, destFolder string) error {
	destPath := filepath.Join(destFolder, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(destFile, srcFile, buf)
	return err
}

// flattenAndMoveYAMLFiles flattens YAML files from repoDir to globalDir, keeping larger duplicates
func flattenAndMoveYAMLFiles(repoDir, globalDir string) error {
	os.MkdirAll(globalDir, os.ModePerm)

	flattenYAMLFiles(repoDir)

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			// Skip files with hash suffixes
			if hashSuffixRegex.MatchString(entry.Name()) {
				os.Remove(filepath.Join(repoDir, entry.Name()))
				continue
			}
			// Skip files with numeric ID suffixes
			if numericSuffixRegex.MatchString(entry.Name()) {
				os.Remove(filepath.Join(repoDir, entry.Name()))
				continue
			}

			info, _ := entry.Info()
			srcPath := filepath.Join(repoDir, entry.Name())
			dstPath := filepath.Join(globalDir, entry.Name())

			globalYAMLMutex.Lock()
			existingSize, exists := globalYAMLFiles[entry.Name()]
			globalYAMLMutex.Unlock()

			if exists {
				currentSize := info.Size()
				if currentSize > existingSize {
					os.Remove(dstPath)
					os.Rename(srcPath, dstPath)
					globalYAMLMutex.Lock()
					globalYAMLFiles[entry.Name()] = currentSize
					globalYAMLMutex.Unlock()
				}
				os.Remove(srcPath)
			} else {
				os.Rename(srcPath, dstPath)
				globalYAMLMutex.Lock()
				globalYAMLFiles[entry.Name()] = info.Size()
				globalYAMLMutex.Unlock()
			}
		}
	}

	os.RemoveAll(repoDir)
	return nil
}

// flattenYAMLFiles recursively moves all .yaml files to the root directory
func flattenYAMLFiles(rootDir string) error {
	rootFiles := make(map[string]int64)

	// First pass: record existing files in root
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			if hashSuffixRegex.MatchString(entry.Name()) {
				os.Remove(filepath.Join(rootDir, entry.Name()))
				continue
			}
			if numericSuffixRegex.MatchString(entry.Name()) {
				os.Remove(filepath.Join(rootDir, entry.Name()))
				continue
			}
			info, _ := entry.Info()
			rootFiles[entry.Name()] = info.Size()
		}
	}

	// Second pass: walk subdirectories and move YAML files
	var dirsToCheck []string
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != rootDir {
			dirsToCheck = append(dirsToCheck, path)
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			if hashSuffixRegex.MatchString(info.Name()) {
				os.Remove(path)
				return nil
			}
			if numericSuffixRegex.MatchString(info.Name()) {
				os.Remove(path)
				return nil
			}

			dir := filepath.Dir(path)
			if dir == rootDir {
				return nil
			}

			destPath := filepath.Join(rootDir, info.Name())

			if existingSize, exists := rootFiles[info.Name()]; exists {
				currentSize := info.Size()
				if currentSize > existingSize {
					os.Remove(destPath)
					os.Rename(path, destPath)
					rootFiles[info.Name()] = currentSize
				}
				os.Remove(path)
			} else {
				os.Rename(path, destPath)
				rootFiles[info.Name()] = info.Size()
			}
		}
		return nil
	})

	for i := len(dirsToCheck) - 1; i >= 0; i-- {
		os.Remove(dirsToCheck[i])
	}

	return err
}

// validateAndCleanTemplates runs nuclei validation on all YAML files and removes invalid ones
func validateAndCleanTemplates(globalDir string) {
	fmt.Printf("\n%s Checking templates with nuclei...\n", magenta("[VALIDATION]"))

	err := filepath.Walk(globalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			cmd := exec.Command("nuclei", "-duc", "-silent", "-validate", "-t", path)
			if err := cmd.Run(); err != nil {
				fmt.Printf("%s %s\n", red("[INVALID]"), path)
				os.Remove(path)
			} else {
				fmt.Printf("%s %s\n", green("[VALIDATED]"), path)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("%s Validation walk failed: %v\n", red("[ERROR]"), err)
	}
}
