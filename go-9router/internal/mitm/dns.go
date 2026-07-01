package mitm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	hostsMutex sync.Mutex
	toolHosts  = []string{
		"daily-cloudcode-pa.googleapis.com",
		"cloudcode-pa.googleapis.com",
	}
)

func getHostsPath() string {
	if runtime.GOOS == "windows" {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		return filepath.Join(systemRoot, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// Check if DNS entry already exists for a host
func HasDNSEntry(host string) bool {
	hostsPath := getHostsPath()
	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), host)
}

// Add DNS redirection entries to hosts file
func AddDNSEntries(sudoPassword string) error {
	hostsMutex.Lock()
	defer hostsMutex.Unlock()

	hostsPath := getHostsPath()
	contentBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	content := string(contentBytes)
	trimmed := strings.TrimRight(content, "\r\n\t ")
	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}

	var toAppend []string
	for _, host := range toolHosts {
		if !strings.Contains(content, host) {
			toAppend = append(toAppend, fmt.Sprintf("127.0.0.1 %s", host))
		}
	}

	if len(toAppend) == 0 {
		return nil
	}

	newContent := trimmed + eol + strings.Join(toAppend, eol) + eol

	if err := writeHostsFile(hostsPath, []byte(newContent), sudoPassword); err != nil {
		return err
	}

	_ = FlushDNS(sudoPassword)
	return nil
}

// Remove DNS redirection entries from hosts file
func RemoveDNSEntries(sudoPassword string) error {
	hostsMutex.Lock()
	defer hostsMutex.Unlock()

	hostsPath := getHostsPath()
	contentBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	lines := strings.Split(string(contentBytes), "\n")
	var newLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		shouldRemove := false
		for _, host := range toolHosts {
			if strings.Contains(trimmedLine, host) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			newLines = append(newLines, line)
		}
	}

	eol := "\n"
	newContent := strings.Join(newLines, eol)
	newContent = strings.TrimRight(newContent, "\r\n\t ") + eol

	if err := writeHostsFile(hostsPath, []byte(newContent), sudoPassword); err != nil {
		return err
	}

	_ = FlushDNS(sudoPassword)
	return nil
}

// Helper to write to hosts file using sudo if on Unix
func writeHostsFile(path string, data []byte, sudoPassword string) error {
	if runtime.GOOS == "windows" {
		// Attempt direct write on Windows (will succeed if already elevated)
		// We use atomic write on Windows: write tmp -> rename to bak -> rename tmp to path
		tmpNew := path + ".9router.new"
		tmpBak := path + ".9router.bak"

		if err := os.WriteFile(tmpNew, data, 0644); err != nil {
			return fmt.Errorf("windows direct write failed: %w", err)
		}

		_ = os.Remove(tmpBak)
		if err := os.Rename(path, tmpBak); err != nil {
			_ = os.Remove(tmpNew)
			return fmt.Errorf("failed backing up hosts: %w", err)
		}

		if err := os.Rename(tmpNew, path); err != nil {
			_ = os.Rename(tmpBak, path) // rollback
			return fmt.Errorf("failed renaming hosts: %w", err)
		}
		_ = os.Remove(tmpBak)
		return nil
	}

	// Unix-like system
	// If password is provided, use sudo -S, otherwise try tee directly
	var cmd *exec.Cmd
	if sudoPassword != "" {
		cmd = exec.Command("sudo", "-S", "tee", path)
		cmd.Stdin = bytes.NewBuffer(append([]byte(sudoPassword+"\n"), data...))
	} else {
		cmd = exec.Command("tee", path)
		cmd.Stdin = bytes.NewReader(data)
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	cmd.Stdout = nil // discard stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write hosts file: %s (err: %w)", errBuf.String(), err)
	}

	return nil
}

// Flush local DNS caches
func FlushDNS(sudoPassword string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("ipconfig", "/flushdns")
		return cmd.Run()
	} else if runtime.GOOS == "darwin" {
		if sudoPassword != "" {
			cmd = exec.Command("sudo", "-S", "sh", "-c", "dscacheutil -flushcache && killall -HUP mDNSResponder")
			cmd.Stdin = strings.NewReader(sudoPassword + "\n")
		} else {
			cmd = exec.Command("sh", "-c", "dscacheutil -flushcache && killall -HUP mDNSResponder")
		}
		return cmd.Run()
	} else {
		// Linux
		if sudoPassword != "" {
			cmd = exec.Command("sudo", "-S", "resolvectl", "flush-caches")
			cmd.Stdin = strings.NewReader(sudoPassword + "\n")
		} else {
			cmd = exec.Command("resolvectl", "flush-caches")
		}
		_ = cmd.Run() // best effort
	}
	return nil
}

// Synchronous DNS cleanup during shutdown
func RemoveAllDNSEntriesSync() {
	hostsPath := getHostsPath()
	contentBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(contentBytes), "\n")
	var newLines []string
	modified := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		shouldRemove := false
		for _, host := range toolHosts {
			if strings.Contains(trimmedLine, host) {
				shouldRemove = true
				modified = true
				break
			}
		}
		if !shouldRemove {
			newLines = append(newLines, line)
		}
	}

	if !modified {
		return
	}

	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}
	newContent := strings.Join(newLines, eol)
	newContent = strings.TrimRight(newContent, "\r\n\t ") + eol

	_ = os.WriteFile(hostsPath, []byte(newContent), 0644)

	// Flush DNS (best-effort sync)
	if runtime.GOOS == "windows" {
		_ = exec.Command("ipconfig", "/flushdns").Run()
	} else if runtime.GOOS == "darwin" {
		_ = exec.Command("sh", "-c", "dscacheutil -flushcache && killall -HUP mDNSResponder").Run()
	} else {
		_ = exec.Command("resolvectl", "flush-caches").Run()
	}
}
