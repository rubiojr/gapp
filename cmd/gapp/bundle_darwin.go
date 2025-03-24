//go:build darwin
// +build darwin

// Originally from https://gist.github.com/mholt/11008646c95d787c30806d3f24b2c844
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Options struct {
	AssetsDir        string `toml:"assets_dir"`
	BinaryName       string `toml:"binary_name"`
	IconFile         string `toml:"icon_file"`
	AppName          string `toml:"app_name"`
	OutputDir        string `toml:"output_dir"`
	BundleIdentifier string `toml:"bundle_identifier"`
}

func bundle(options *Options) error {
	if options.AssetsDir == "" || options.IconFile == "" || options.BinaryName == "" || options.AppName == "" {
		return fmt.Errorf("assets directory, binary name, icon file, and application name are required.")
	}

	// make and fill out the .app bundle
	appName := strings.TrimSuffix(options.AppName, ".app")
	appFilename := appName + ".app"
	appBundleName := filepath.Join(options.OutputDir, appFilename)
	return makeAppBundle(appBundleName, options)
}

func makeAppBundle(appFilename string, options *Options) error {
	// make the basic directory structure
	for _, dirName := range []string{
		filepath.Join(appFilename, "Contents", "MacOS"),
		filepath.Join(appFilename, "Contents", "Resources"),
	} {
		err := os.MkdirAll(dirName, 0755)
		if err != nil {
			return fmt.Errorf("making app folder structure: %v", err)
		}
	}

	binaryName := options.BinaryName
	bundleIdentifier := options.BundleIdentifier
	assetsDir := options.AssetsDir
	// write the Info.plist file into the bundle
	infoPlist := strings.Replace(infoPlistTpl, "{{.AppName}}", binaryName, -1)
	infoPlist = strings.Replace(infoPlist, "{{.BundleIdentifier}}", bundleIdentifier, -1)
	infoPlistPath := filepath.Join(appFilename, "Contents", "Info.plist")
	err := os.WriteFile(infoPlistPath, []byte(infoPlist), 0644)
	if err != nil {
		return fmt.Errorf("writing plist file: %v", err)
	}

	// set the icons
	err = makeAppIcons(appFilename, options)
	if err != nil {
		return fmt.Errorf("making icons: %v", err)
	}

	// copy the binary into the bundle
	binarySrc := filepath.Join(assetsDir, binaryName)
	binaryDest := filepath.Join(appFilename, "Contents", "MacOS", binaryName)
	err = copyFile(binarySrc, binaryDest, nil)
	if err != nil {
		return fmt.Errorf("copying the binary into the bundle: %v", err)
	}

	// get the list of assets to copy
	assetsDirFile, err := os.Open(assetsDir)
	if err != nil {
		return fmt.Errorf("opening assets directory: %v", err)
	}
	dirEntries, err := assetsDirFile.Readdirnames(100000)
	if err != nil {
		return fmt.Errorf("reading list of assets directory contents: %v", err)
	}

	// copy the assets into the bundle
	for _, entry := range dirEntries {
		if entry == binaryName {
			continue // we already copied the binary, and it went into a different folder
		}

		src := filepath.Join(assetsDir, entry)
		dest := filepath.Join(appFilename, "Contents", "Resources")

		err = deepCopy(src, dest)
		if err != nil {
			return fmt.Errorf("copying assets '%s': %v", entry, err)
		}
	}

	return nil
}

func makeAppIcons(appFolder string, options *Options) error {
	iconFile := options.IconFile
	// start by copying the icon into the bundle
	iconFilename := filepath.Base(iconFile)
	resFolder := filepath.Join(appFolder, "Contents", "Resources")
	copyTo := filepath.Join(resFolder, iconFilename)
	err := copyFile(iconFile, copyTo, nil)
	if err != nil {
		return err
	}

	useIcon := iconFile // usable icon files are of type .png, .jpg, .gif, or .tiff - and we handle .svg
	tmpFolder := filepath.Join(resFolder, "tmp")
	err = os.MkdirAll(tmpFolder, 0755)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpFolder)

	// lazy way to convert SVG files to PNG, by using QuickLook
	// -z displays generation performance info (instead of showing thumbnail)
	// -t Computes the thumbnail
	// -s sets the size of the thumbnail
	// -o sets the output directory (NOT the actual output file)
	if filepath.Ext(iconFile) == ".svg" {
		cmd := exec.Command("qlmanage", "-z", "-t", "-s", "1024", "-o", tmpFolder, iconFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("running qlmanage: %v", err)
		}
		useIcon = filepath.Join(tmpFolder, iconFile+".png")
	}

	// make the various icon sizes
	// see https://developer.apple.com/library/content/documentation/GraphicsAnimation/Conceptual/HighResolutionOSX/Optimizing/Optimizing.html
	iconset := filepath.Join(tmpFolder, "icon.iconset")
	err = os.Mkdir(iconset, 0755)
	if err != nil {
		return err
	}
	sizes := []int{16, 32, 64, 128, 256, 512, 1024}
	for i, size := range sizes {
		nameSize := size
		var suffix string
		if i > 0 {
			nameSize = sizes[i-1]
			suffix = "@2x"
		}

		iconName := fmt.Sprintf("icon_%dx%d%s.png", nameSize, nameSize, suffix)
		outIconFile := filepath.Join(iconset, iconName)

		sizeStr := fmt.Sprintf("%d", size)
		cmd := exec.Command("sips", "-z", sizeStr, sizeStr, useIcon, "--out", outIconFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("running sips: %v", err)
		}

		// make standard-DPI version if we didn't already
		if i > 0 && i < len(sizes)-1 {
			stdName := fmt.Sprintf("icon_%dx%d.png", size, size)
			err := copyFile(outIconFile, filepath.Join(iconset, stdName), nil)
			if err != nil {
				return fmt.Errorf("copying icon file: %v", err)
			}
		}
	}

	// create the final .icns file
	icnsFile := filepath.Join(resFolder, "icon.icns")
	cmd := exec.Command("iconutil", "-c", "icns", "-o", icnsFile, iconset)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("running iconutil: %v", err)
	}

	return nil
}

func copyFile(from, to string, fromInfo os.FileInfo) error {
	log.Printf("[INFO] Copying %s to %s", from, to)

	if fromInfo == nil {
		var err error
		fromInfo, err = os.Stat(from)
		if err != nil {
			return err
		}
	}

	// open source file
	fsrc, err := os.Open(from)
	if err != nil {
		return err
	}

	// create destination file, with identical permissions
	fdest, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fromInfo.Mode()&os.ModePerm)
	if err != nil {
		fsrc.Close()
		if _, err2 := os.Stat(to); err2 == nil {
			return fmt.Errorf("opening destination (which already exists): %v", err)
		}
		return err
	}

	// copy the file and ensure it gets flushed to disk
	if _, err = io.Copy(fdest, fsrc); err != nil {
		fsrc.Close()
		fdest.Close()
		return err
	}
	if err = fdest.Sync(); err != nil {
		fsrc.Close()
		fdest.Close()
		return err
	}

	// close both files
	if err = fsrc.Close(); err != nil {
		fdest.Close()
		return err
	}
	if err = fdest.Close(); err != nil {
		return err
	}

	return nil
}

// deepCopy makes a deep copy of from into to.
func deepCopy(from, to string) error {
	if from == "" || to == "" {
		return fmt.Errorf("no source or no destination; both required")
	}

	// traverse the source directory and copy each file
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		// error accessing current file
		if err != nil {
			return err
		}

		// skip files/folders without a name
		if info.Name() == "" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// if directory, create destination directory (if not
		// already created by our pre-walk)
		if info.IsDir() {
			subdir := strings.TrimPrefix(path, filepath.Dir(from))
			destDir := filepath.Join(to, subdir)
			if _, err := os.Stat(destDir); os.IsNotExist(err) {
				err := os.Mkdir(destDir, info.Mode()&os.ModePerm)
				if err != nil {
					return err
				}
			}
			return nil
		}

		destPath := filepath.Join(to, strings.TrimPrefix(path, filepath.Dir(from)))
		err = copyFile(path, destPath, info)
		if err != nil {
			return fmt.Errorf("copying file %s: %v", path, err)
		}
		return nil
	})
}

// See https://developer.apple.com/library/content/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html#//apple_ref/doc/uid/10000123i-CH101-SW19
// for information about the Info.plist and bundling an application.
//
// If code signing / notarizing, you also need:
//
//	<key>CFBundlePackageType</key>
//	<string>APPL</string>
const infoPlistTpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>{{.AppName}}</string>
	<key>CFBundleIconFile</key>
	<string>icon.icns</string>
	<key>CFBundleIdentifier</key>
	<string>{{.BundleIdentifier}}</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSUIElement</key>
	<true/>
</dict>
</plist>
`
