package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

//go:embed _embed
var embedFS embed.FS

const version = "1.0.0"

func main() {
	app := &cli.App{
		Name:  "gapp",
		Usage: "A CLI application with a build command",
		Commands: []*cli.Command{
			{
				Name:  "build",
				Usage: "Build the application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output directory path",
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Configuration file path",
					},
				},
				Action: func(c *cli.Context) error {
					return runBuild(c)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func runBuild(c *cli.Context) error {
	outputDir := c.String("output")
	if outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	configFile := c.String("config")
	if configFile != "" {
		fmt.Printf("Using config file: %s\n", configFile)
	}

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	data, err := embedFS.ReadFile("_embed/main.go")
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %w", err)
	}

	outputPath := filepath.Join(outputDir, "main.go")
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	gomod := filepath.Join(outputDir, "go.mod")
	err = os.WriteFile(gomod, []byte("module gapp\ngo 1.23"), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	err = os.Chdir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to change to output directory: %w", err)
	}

	cmd := exec.Command("go", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	err = os.Chdir(currentDir)
	if err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	fmt.Printf("Successfully wrote main.txt to %s\n", outputPath)
	return nil
}
