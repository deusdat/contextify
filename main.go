package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	inputPath   string
	outputPath  string
	excludeDirs []string
	includeExts []string
	excludeMap  map[string]bool
	includeMap  map[string]bool
	logger      *slog.Logger
}

func main() {
	var (
		inputPath   = flag.String("input", ".", "Input directory path (relative or absolute)")
		outputPath  = flag.String("output", "context.txt", "Output file path")
		excludeDirs = flag.String("exclude", "", "Comma-separated list of directories to exclude (e.g., node_modules,dist,.git)")
		includeExts = flag.String("extensions", "", "Comma-separated list of file extensions to include (e.g., .ts,.js,.go)")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Configure logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Always exclude .git directory
	excludeList := parseCommaSeparated(*excludeDirs)
	excludeList = ensureGitExcluded(excludeList)

	config := &Config{
		inputPath:   *inputPath,
		outputPath:  *outputPath,
		excludeDirs: excludeList,
		includeExts: parseCommaSeparated(*includeExts),
		logger:      logger,
	}

	// Create lookup maps for faster checking
	config.excludeMap = createLookupMap(config.excludeDirs)
	config.includeMap = createLookupMap(config.includeExts)

	logger.Info("Starting contextify",
		"input", config.inputPath,
		"output", config.outputPath,
		"excludeDirs", config.excludeDirs,
		"includeExts", config.includeExts,
	)

	if err := processDirectory(config); err != nil {
		logger.Error("Failed to process directory", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully created context file", "output", config.outputPath)
}

func parseCommaSeparated(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func createLookupMap(items []string) map[string]bool {
	lookup := make(map[string]bool)
	for _, item := range items {
		lookup[item] = true
	}
	return lookup
}

func processDirectory(config *Config) error {
	logger := config.logger

	// Convert to absolute path for consistent handling
	absPath, err := filepath.Abs(config.inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	logger.Debug("Processing directory", "absolutePath", absPath)

	// Create output file
	outputFile, err := os.Create(config.outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			logger.Error("Failed to close output file", "error", closeErr)
		}
	}()

	writer := bufio.NewWriter(outputFile)
	defer func() {
		if flushErr := writer.Flush(); flushErr != nil {
			logger.Error("Failed to flush writer", "error", flushErr)
		}
	}()

	// Write header
	if err := writeHeader(writer, absPath, config); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	fileCount := 0
	// Walk the directory tree
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Warn("Error accessing path", "path", path, "error", err)
			return err
		}

		// Get relative path from the input directory
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return err
		}

		// Check if we should exclude this directory
		if d.IsDir() {
			if shouldExcludeDir(relPath, config.excludeMap) {
				logger.Debug("Excluding directory", "path", relPath)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we should include this file
		if !shouldIncludeFile(path, config.includeMap) {
			logger.Debug("Skipping file (extension not included)", "path", relPath)
			return nil
		}

		// Process the file
		logger.Debug("Processing file", "path", relPath)
		if err := processFile(path, relPath, writer, logger); err != nil {
			logger.Error("Failed to process file", "path", relPath, "error", err)
			return err
		}

		fileCount++
		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("Processing completed", "filesProcessed", fileCount)
	return nil
}

func writeHeader(writer *bufio.Writer, absPath string, config *Config) error {
	headers := []string{
		"# Contextify Output\n",
		fmt.Sprintf("# Generated from: %s\n", absPath),
		fmt.Sprintf("# Excluded directories: %s\n", strings.Join(config.excludeDirs, ", ")),
	}

	if len(config.includeExts) > 0 {
		headers = append(headers, fmt.Sprintf("# Included extensions: %s\n", strings.Join(config.includeExts, ", ")))
	}
	headers = append(headers, "\n")

	for _, header := range headers {
		if _, err := fmt.Fprint(writer, header); err != nil {
			return err
		}
	}

	return nil
}

func shouldExcludeDir(relPath string, excludeMap map[string]bool) bool {
	if len(excludeMap) == 0 {
		return false
	}

	// Check each part of the path
	parts := strings.Split(relPath, string(filepath.Separator))
	for _, part := range parts {
		if excludeMap[part] {
			return true
		}
	}

	// Also check the full relative path
	return excludeMap[relPath]
}

func shouldIncludeFile(filePath string, includeMap map[string]bool) bool {
	// If no extensions specified, include all files
	if len(includeMap) == 0 {
		return true
	}

	ext := filepath.Ext(filePath)
	return includeMap[ext]
}

func processFile(fullPath, relPath string, writer *bufio.Writer, logger *slog.Logger) error {
	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Warn("Failed to close input file", "path", relPath, "error", closeErr)
		}
	}()

	// Get file info for logging
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Warn("Could not get file stats", "path", relPath, "error", err)
	} else {
		logger.Debug("File info", "path", relPath, "size", fileInfo.Size())
	}

	// Write file header with path information
	if _, err := fmt.Fprintf(writer, "## File: %s\n", relPath); err != nil {
		return fmt.Errorf("failed to write file header: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "```\n"); err != nil {
		return fmt.Errorf("failed to write code block start: %w", err)
	}

	// Copy file contents
	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		if _, err := fmt.Fprintf(writer, "%s\n", scanner.Text()); err != nil {
			return fmt.Errorf("failed to write file content: %w", err)
		}
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", fullPath, err)
	}

	logger.Debug("File processed", "path", relPath, "lines", lineCount)

	if _, err := fmt.Fprintf(writer, "```\n\n"); err != nil {
		return fmt.Errorf("failed to write code block end: %w", err)
	}

	return nil
}

// ensureGitExcluded adds .git to the exclude list if it's not already present
func ensureGitExcluded(excludeDirs []string) []string {
	for _, dir := range excludeDirs {
		if dir == ".git" {
			return excludeDirs // .git already in the list
		}
	}
	// Add .git to the list
	return append(excludeDirs, ".git")
}