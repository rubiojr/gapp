package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
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
						Value:   "gapp.bin",
						Usage:   "Output binary path",
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Configuration file path",
					},
					&cli.StringFlag{
						Name:  "bundle-manifest",
						Usage: "Bundle the binary",
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
	configFile := c.String("config")
	if configFile == "" {
		return fmt.Errorf("config file path is required")
	}

	fileInfo, err := os.Stat(configFile)
	if err != nil {
		return fmt.Errorf("config file does not exist: %w", err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("config path is a directory, not a file")
	}

	outputBin := c.String("output")
	outputBin, err = filepath.Abs(outputBin)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output binary: %w", err)
	}

	outputDir, err := os.MkdirTemp("", "gapp-build-")
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Output directory: %s\n", outputDir)
	logFile := filepath.Join(outputDir, "build.log")

	f, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer f.Close()

	data, err := embedFS.ReadFile("_embed/main.go")
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %w", err)
	}

	outputPath := filepath.Join(outputDir, "main.go")
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	data, err = os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	outputPath = filepath.Join(outputDir, "glance.yml")
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

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOPROXY=direct")
	cmd.Stdout = f
	cmd.Stderr = f

	defer func() {
		if err != nil {
			lf, _ := os.ReadFile(logFile)
			fmt.Println(string(lf))
		}
	}()

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	cmd = exec.Command("go", "build", "-o", outputBin)
	cmd.Stdout = f
	cmd.Stderr = f

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	err = os.Chdir(currentDir)
	if err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	fmt.Printf("Successfully wrote binary to %s\n", outputBin)

	manifest := c.String("bundle-manifest")
	if manifest != "" {
		fmt.Println("Bundling app bundle...")
		if runtime.GOOS == "darwin" {
			opts, err := readManifest(manifest)
			if err != nil {
				return fmt.Errorf("failed to read manifest file: %w", err)
			}
			return bundle(opts)

		}
	}

	return nil
}

func readManifest(path string) (*Options, error) {
	var options Options

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	if err := toml.Unmarshal(data, &options); err != nil {
		return nil, fmt.Errorf("failed to parse manifest file: %w", err)
	}

	return &options, nil
}
