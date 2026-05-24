package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	fmt.Println("🛠️  NosBoost Resource Compilation Utility")
	fmt.Println("==================================================")

	manifestFile := "nosboost.manifest"
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		fmt.Printf("❌ ERROR: %s not found in the root directory.\n", manifestFile)
		os.Exit(1)
	}

	// 1. Check if rsrc is available in path
	_, err := exec.LookPath("rsrc")
	if err != nil {
		fmt.Println("⚠️  'rsrc' tool not found in PATH. Attempting to install...")
		err = runCommand("go", "install", "github.com/akavel/rsrc@latest")
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to install github.com/akavel/rsrc: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 'rsrc' installed successfully.")
	}

	// 2. Discover GOPATH/bin to locate the installed rsrc binary if it's not on system PATH
	rsrcPath := "rsrc"
	if _, err := exec.LookPath("rsrc"); err != nil {
		// Try to find it in go env GOPATH
		out, err := exec.Command("go", "env", "GOPATH").Output()
		if err == nil {
			gopath := string(out)
			// Clean whitespace
			gopath = filepath.Clean(filepath.VolumeName(gopath) + filepath.Clean(gopath[len(filepath.VolumeName(gopath)):]))
			// Standard paths
			home, _ := os.UserHomeDir()
			candidates := []string{
				filepath.Join(filepath.Clean(string(out)), "bin", "rsrc.exe"),
				filepath.Join(filepath.Clean(string(out)), "bin", "rsrc"),
				filepath.Join(home, "go", "bin", "rsrc.exe"),
				filepath.Join(home, "go", "bin", "rsrc"),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					rsrcPath = c
					break
				}
			}
		}
	}

	// 3. Compile the manifest into COFF resource syso file
	outputSyso := filepath.Join("cmd", "nosboost", "rsrc.syso")
	fmt.Printf("📦 Compiling %s to %s...\n", manifestFile, outputSyso)
	
	args := []string{"-manifest", manifestFile, "-o", outputSyso}
	
	// Optional: If an icon exists (e.g., assets/icon.ico), you can append: -ico assets/icon.ico
	iconPath := filepath.Join("assets", "icon.ico")
	if _, err := os.Stat(iconPath); err == nil {
		fmt.Printf("🎨 Found app icon at %s, linking to resource file...\n", iconPath)
		args = append(args, "-ico", iconPath)
	}

	err = runCommand(rsrcPath, args...)
	if err != nil {
		fmt.Printf("❌ ERROR: Resource compilation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("==================================================")
	fmt.Println("🎉 SUCCESS: native Windows COFF resource file generated!")
	fmt.Printf("   Path: %s\n", outputSyso)
	fmt.Println("   The next 'go build' run will automatically embed this manifest.")
	
	if runtime.GOOS != "windows" {
		fmt.Println("⚠️  NOTE: You are running on a non-Windows OS. This compiled resource")
		fmt.Println("   will be embedded once compiled with GOOS=windows.")
	}
}
