package capture

import (
	"bytes"
	"os/exec"
	"strings"
)

type WindowInfo struct {
	App   string
	Title string
}

func GetActiveWindow() (*WindowInfo, error) {
	script := `
		tell application "System Events"
			set frontApp to first application process whose frontmost is true
			set appName to name of frontApp
			try
				set windowTitle to name of first window of frontApp
			on error
				set windowTitle to ""
			end try
			return appName & "|||" & windowTitle
		end tell
	`
	
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	result := string(bytes.TrimSpace(output))
	parts := strings.SplitN(result, "|||", 2)
	
	info := &WindowInfo{}
	if len(parts) >= 1 {
		info.App = parts[0]
	}
	if len(parts) >= 2 {
		info.Title = parts[1]
	}
	
	return info, nil
}

func GetRunningApps() ([]string, error) {
	script := `
		tell application "System Events"
			set appNames to name of every application process whose visible is true
			set AppleScript's text item delimiters to "|||"
			return appNames as string
		end tell
	`
	
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	result := string(bytes.TrimSpace(output))
	if result == "" {
		return []string{}, nil
	}
	return strings.Split(result, "|||"), nil
}
