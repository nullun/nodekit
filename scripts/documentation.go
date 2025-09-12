package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/algorandfoundation/nodekit/cmd"
	"github.com/algorandfoundation/nodekit/ui/style"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra/doc"
)

const (

	// DEVPORTAL_PATH is the root directory path for storing developer portal-related documentation and assets.
	DEVPORTAL_PATH = "./.devportal/"

	// MANPAGE_URL defines the base URL path where the manual pages for the CLI commands are hosted.
	MANPAGE_URL = "/man/"
)

// copyFile copies the content of a source file to a destination file; optionally removes the source file if move is true.
// src is the path to the source file.
// dst is the path to the destination file.
// move is a boolean indicating whether the source file should be removed after copying.
// Returns an error if any file operation fails.
func copyFile(src, dst string, move bool) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Optionally, sync to ensure writes are flushed to disk
	err = destinationFile.Sync()
	if err != nil {
		return err
	}
	if !move {
		return nil
	}

	return os.Remove(src)
}

// appendString appends the provided content to the file at the given filePath or creates the file if it doesn't exist.
// filePath is the path to the file to append to or create.
// content is the string data to be appended to the file.
// Returns an error if the file cannot be opened, written to, or closed properly.
func appendString(filePath, content string) error {
	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the content to the file
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// generateMarkdown generates Markdown documentation for commands in the CLI tool and saves it in the specified directory.
// It uses custom functions for file preprocessing and link handling to customize the output.
// Returns an error if the documentation generation fails.
func generateMarkdown() error {
	// No need to prefix the file
	filePrepender := func(filename string) string {
		return ""
	}
	// Ensure the links are valid for the README
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		if base == cmd.Name {
			return "/README.md"
		}
		return MANPAGE_URL + strings.ToLower(base) + ".md"
	}
	cmd.RootCmd.DisableAutoGenTag = true
	// Generate the documentation
	return doc.GenMarkdownTreeCustom(cmd.RootCmd, fmt.Sprintf(".%s", MANPAGE_URL), filePrepender, linkHandler)
}

// replaceBetweenStrings replaces everything between startString and endString with replacementText in the content of the file
func replaceBetweenStrings(filePath, startString, endString, replacementText string) error {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Convert content to string
	text := string(content)

	// Find the start and end boundaries
	startIndex := strings.Index(text, startString)
	endIndex := strings.Index(text, endString)
	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return fmt.Errorf("could not find valid boundaries between '%s' and '%s'", startString, endString)
	}

	// Build the new content
	// Preserve everything before and after the boundaries, and insert the replacement text in-between
	newContent := text[:startIndex] + replacementText + text[endIndex+len(endString):]

	// Write the modified content back to the file
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

// updateBanner replaces a section of text in the given file with an image tag for displaying a terminal render banner.
func updateBanner(filePath string) error {
	textBanner := ansi.Strip(style.BANNER)
	textSplit := strings.Split(textBanner, "\n")
	return replaceBetweenStrings(filePath, textSplit[1], textSplit[len(textSplit)-2], "<img alt=\"Terminal Render\" src=\"/assets/nodekit.png\" width=\"65%\">")
}

// updateBanners updates markdown files in the given directory by modifying banner sections
// dirPath specifies the directory containing the files to update.
// delete flag to delete the banner instead of injecting it
// Returns an error if processing fails for any file in the directory.
func updateBanners(dirPath string, delete bool) error {
	// Open the directory
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	// Read directory entries
	files, err := dir.Readdir(-1) // `-1` reads all entries in the directory
	if err != nil {
		return err
	}

	// Iterate over all files and directories
	for _, file := range files {
		if file.IsDir() {
			fmt.Printf("Skipping directory: %s\n", file.Name())
			continue
		}
		if strings.HasSuffix(file.Name(), ".md") && strings.HasPrefix(file.Name(), cmd.Name) {
			if delete {
				err = deleteBanner(dirPath + file.Name())
				if err != nil {
					return err
				}
			} else {
				err = updateBanner(dirPath + file.Name())
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// deleteBanner removes the banner section from a file by replacing text between specific delimiters with empty content.
// filePath specifies the path to the file where the banner will be removed.
// Returns an error if the operation fails.
func deleteBanner(filePath string) error {
	textBanner := ansi.Strip(style.BANNER)
	textSplit := strings.Split(textBanner, "\n")
	return replaceBetweenStrings(filePath, "## nodekit", textSplit[len(textSplit)-2], "## Synopsis")
}

// getAllBlocksFromDir reads files from a directory, extracts content between specified strings, and returns as a map.
// dirPath specifies the directory path containing the files.
// startString and endString mark the boundaries of the content to extract from each file.
// Returns a map of file names (modified without `.md`) to their extracted content, or an error if any operation fails.
func getAllBlocksFromDir(dirPath, startString, endString string) (map[string]string, error) {
	// Results map to store file names and their extracted content
	results := make(map[string]string)

	// Open the directory
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	// Read directory entries
	files, err := dir.Readdir(-1) // `-1` reads all entries in the directory
	if err != nil {
		return nil, err
	}

	// Iterate over files in the directory
	for _, file := range files {
		// Skip directories and files without underscores in their names
		if file.IsDir() || !strings.Contains(file.Name(), "_") {
			continue
		}

		// Build the full file path
		filePath := filepath.Join(dirPath, file.Name())

		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		// Convert content to string
		text := string(content)

		// Find the start and end indices of the desired content
		startIndex := strings.Index(text, startString)
		endIndex := strings.Index(text, endString)
		if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
			return nil, fmt.Errorf("could not find valid boundaries in file '%s' between '%s' and '%s'", file.Name(), startString, endString)
		}

		// Extract the content between the start and end strings
		extractedContent := text[startIndex+len(startString) : endIndex]
		results[strings.Replace(file.Name(), ".md", "", -1)] = extractedContent
	}

	return results, nil
}

const fmTemplate = `---
title: "%s"
---
`

// generateStarlightMarkdown generates Markdown documentation for CLI commands in the developer portal format.
// It uses custom file preprocessing and link handling to format filenames and links appropriately.
// Outputs files to the DEVPORTAL_PATH directory and returns an error if generation fails.
func generateStarlightMarkdown() error {
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))

		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/reference/" + strings.Replace(base, "_", "/", -1)
	}
	cmd.RootCmd.DisableAutoGenTag = false
	return doc.GenMarkdownTreeCustom(cmd.RootCmd, DEVPORTAL_PATH, filePrepender, linkHandler)
}

func main() {
	// Generate Man pages for main repo
	err := generateMarkdown()
	if err != nil {
		panic(err)
	}
	// Add the banners to the markdown
	rootCmdDocPath := fmt.Sprintf(".%s/%s.md", MANPAGE_URL, cmd.Name)
	err = updateBanners(fmt.Sprintf(".%s", MANPAGE_URL), false)
	if err != nil {
		panic(err)
	}
	// Add Footer
	footerDocPath := "./assets/footer.md"
	footerBytes, err := os.ReadFile(footerDocPath)
	if err != nil {
		panic(err)
	}
	err = appendString(rootCmdDocPath, "\n"+string(footerBytes))
	if err != nil {
		panic(err)
	}
	// Move the root command to README
	err = copyFile(fmt.Sprintf("./man/%s.md", cmd.Name), "./README.md", true)
	if err != nil {
		panic(err)
	}

	// Generate Devportal Documentation
	err = generateStarlightMarkdown()
	if err != nil {
		panic(err)
	}
	// Delete the banners from the markdown
	err = updateBanners(DEVPORTAL_PATH, true)
	// Fetch all of the blocks from the directory, assumes subcommands have underscores
	blocks, err := getAllBlocksFromDir(DEVPORTAL_PATH, "## Synopsis", "### SEE ALSO")

	// Sort the keys
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create the new merged documentation
	var mergedList string
	mergedList += "# Commands \n"
	for _, k := range keys {
		// Handle the titles
		fileNameSplit := strings.Split(k, "_")
		if len(fileNameSplit) == 3 {
			mergedList += fmt.Sprintf("## %s %s", fileNameSplit[1], fileNameSplit[2])
		} else {
			mergedList += fmt.Sprintf("## %s", fileNameSplit[1])
		}
		mergedList += strings.Replace(blocks[k], "### Options", "#### Options", -1)
	}
	// Add back the auto tag
	mergedList += "###### Auto"

	// Replace the TOC with the full list of commands
	err = replaceBetweenStrings(fmt.Sprintf("%s/commands.md", DEVPORTAL_PATH), "### SEE ALSO", "###### Auto", mergedList)
	if err != nil {
		panic(err)
	}

	// Delete all files with underscores in their names in the reference directory
	referenceDir := DEVPORTAL_PATH
	dirEntries, err := os.ReadDir(referenceDir)
	if err != nil {
		panic(err)
	}
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.Contains(entry.Name(), "_") {
			filePath := filepath.Join(referenceDir, entry.Name())
			err := os.Remove(filePath)
			if err != nil {
				panic(fmt.Errorf("failed to delete file '%s': %w", entry.Name(), err))
			}
			fmt.Printf("Deleted file: %s\n", filePath)
		}
	}
}
