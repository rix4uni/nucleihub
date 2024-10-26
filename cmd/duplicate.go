package cmd

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/spf13/cobra"
)

// duplicateCmd represents the duplicate command
var duplicateCmd = &cobra.Command{
    Use:   "duplicate",
    Short: "Find and save unique or large content templates from downloaded files",
    Long:  `This command processes templates in the specified input directory and saves unique or largest files to the output directory.

Examples:
  nucleihub duplicate --input-directory ~/nucleihub-downloaded-repos --output-directory ~/nucleihub-templates
  nucleihub duplicate --input-directory ~/nucleihub-downloaded-repos --output-directory ~/nucleihub-templates --large-content
`,
    Run: func(cmd *cobra.Command, args []string) {
        inputDir, _ := cmd.Flags().GetString("input-directory")
        outputDir, _ := cmd.Flags().GetString("output-directory")
        largeContentFlag, _ := cmd.Flags().GetBool("large-content")

        inputDir = expandHomePath(inputDir)
        outputDir = expandHomePath(outputDir)

        processTemplates(inputDir, outputDir, largeContentFlag)
    },
}

func expandHomePath(path string) string {
    if strings.HasPrefix(path, "~/") {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            fmt.Println("Error retrieving home directory:", err)
            return path
        }
        path = filepath.Join(homeDir, path[2:])
    } else if strings.HasPrefix(path, "$HOME") {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            fmt.Println("Error retrieving home directory:", err)
            return path
        }
        path = filepath.Join(homeDir, path[5:])
    }
    return path
}

func processTemplates(inputDir, outputDir string, largeContentFlag bool) {
    os.MkdirAll(outputDir, os.ModePerm)

    files := make(map[string][]string) // map of base filename to file paths

    // Regular expression to match and strip the 32-character suffix
    re := regexp.MustCompile(`^(.*?)-[a-f0-9]{32}\.yaml$`)

    err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            filename := info.Name()
            baseFilename := filename

            if matches := re.FindStringSubmatch(filename); matches != nil {
                baseFilename = matches[1] + ".yaml" // Strip suffix for comparison
            }

            files[baseFilename] = append(files[baseFilename], path)
        }
        return nil
    })

    if err != nil {
        fmt.Println("Error walking the path:", err)
        return
    }

    for baseFilename, paths := range files {
        if len(paths) > 1 {
            // Handle duplicates based on flag
            if largeContentFlag {
                handleLargeContent(paths, baseFilename, outputDir)
            } else {
                handleDefault(paths, baseFilename, outputDir)
            }
        } else {
            // Copy unique file directly
            saveFile(paths[0], filepath.Join(outputDir, baseFilename))
        }
    }

    fmt.Println("Templates processed successfully.")
}

func handleDefault(paths []string, baseFilename, outputDir string) {
    var savedFile bool

    for i, path := range paths {
        if !savedFile {
            // Save the first file with the base filename
            saveFile(path, filepath.Join(outputDir, baseFilename))
            savedFile = true
        } else {
            // Save duplicates with a suffix
            outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_%d.yaml", strings.TrimSuffix(baseFilename, ".yaml"), i))
            saveFile(path, outputFile)
        }
    }
}

func handleLargeContent(paths []string, baseFilename, outputDir string) {
    var largestFile string
    var largestSize int64 = -1

    for _, path := range paths {
        info, err := os.Stat(path)
        if err != nil {
            fmt.Println("Error getting file info:", err)
            continue
        }

        if info.Size() > largestSize {
            largestSize = info.Size()
            largestFile = path
        }
    }

    if largestFile != "" {
        saveFile(largestFile, filepath.Join(outputDir, baseFilename))
    }
}

func saveFile(src, dst string) {
    content, err := ioutil.ReadFile(src)
    if err != nil {
        fmt.Println("Error reading file:", err)
        return
    }
    err = ioutil.WriteFile(dst, content, 0644)
    if err != nil {
        fmt.Println("Error writing file:", err)
        return
    }
    fmt.Printf("Saved template to %s\n", dst)
}

func init() {
    rootCmd.AddCommand(duplicateCmd)

    duplicateCmd.Flags().String("input-directory", "~/nucleihub-downloaded-repos", "Directory to scan for templates")
    duplicateCmd.Flags().String("output-directory", "~/nucleihub-templates", "Directory to save processed templates")
    duplicateCmd.Flags().Bool("large-content", false, "Only save the largest file content if duplicates are found")
}