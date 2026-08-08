package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const phoneLaunchAgentLabel = "com.nhestrompia.eve.phone"

var phoneVersionPattern = regexp.MustCompile(`(?m)(\d+)\.(\d+)(?:\.(\d+))?`)

type phoneCommandRunner interface {
	Run(name string, args []string, env []string) ([]byte, error)
}

type systemPhoneCommandRunner struct{}

func (systemPhoneCommandRunner) Run(name string, args []string, env []string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Env = append(os.Environ(), env...)
	return command.CombinedOutput()
}

type tailscaleIdentity struct {
	DNSName   string
	LoginName string
}

func runPhone(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printPhoneUsage(stderr)
		return 2
	}
	runner := systemPhoneCommandRunner{}
	switch args[0] {
	case "setup":
		return runPhoneSetup(args[1:], runner, stdout, stderr)
	case "status":
		return runPhoneStatus(args[1:], runner, stdout, stderr)
	case "disable":
		return runPhoneDisable(args[1:], runner, stdout, stderr)
	case "reset":
		return runPhoneReset(args[1:], stdin, runner, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eve phone command %q\n", args[0])
		printPhoneUsage(stderr)
		return 2
	}
}

func printPhoneUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: eve phone <setup|status|disable|reset>")
	fmt.Fprintln(w, "  setup [--serve-port 8443] [--replace]")
	fmt.Fprintln(w, "  status")
	fmt.Fprintln(w, "  disable")
	fmt.Fprintln(w, "  reset [--yes]")
}

func runPhoneSetup(args []string, runner phoneCommandRunner, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("phone setup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	servePort := fs.Int("serve-port", 8443, "private Tailscale HTTPS port")
	replace := fs.Bool("replace", false, "replace a configuration for another Tailscale identity or origin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *servePort < 1 || *servePort > 65535 {
		fmt.Fprintln(stderr, "eve phone setup requires a valid --serve-port and no positional arguments")
		return 2
	}
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(stderr, "eve phone setup currently supports macOS hosts only")
		return 1
	}
	tailscale, env, err := locateTailscaleCLI()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	versionOutput, err := runner.Run(tailscale, []string{"version"}, env)
	if err != nil {
		fmt.Fprintf(stderr, "read Tailscale version: %v\n%s", err, versionOutput)
		return 1
	}
	if !tailscaleVersionAtLeast(string(versionOutput), 1, 52) {
		fmt.Fprintf(stderr, "Tailscale 1.52 or newer is required; found %s\n", strings.TrimSpace(string(versionOutput)))
		return 1
	}
	statusOutput, err := runner.Run(tailscale, []string{"status", "--json"}, env)
	if err != nil {
		fmt.Fprintf(stderr, "read Tailscale status: %v\n%s", err, statusOutput)
		return 1
	}
	identity, err := parseTailscaleIdentity(statusOutput)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	store := defaultPhoneConfigStore()
	existing, existingErr := store.load()
	if existingErr != nil && !errors.Is(existingErr, os.ErrNotExist) {
		fmt.Fprintln(stderr, existingErr)
		return 1
	}
	changedOrigin := existingErr == nil && (existing.ServePort != *servePort || !strings.EqualFold(existing.ServeHost, identity.DNSName) || !strings.EqualFold(existing.AllowedTailscaleLogin, identity.LoginName))
	if changedOrigin && !*replace {
		fmt.Fprintln(stderr, "the active Tailscale identity or phone origin differs from the saved setup; rerun with --replace to clear old device subscriptions")
		return 1
	}
	serveStatus, _ := runner.Run(tailscale, []string{"serve", "status", "--json"}, env)
	target := "http://127.0.0.1:4317"
	if phoneServePortOccupied(serveStatus, *servePort, target) {
		fmt.Fprintf(stderr, "Tailscale Serve port %d is already used by another service; choose another --serve-port\n", *servePort)
		return 1
	}
	serveArgs := []string{"serve", "--bg", "--https=" + strconv.Itoa(*servePort), target}
	serveOutput, err := runner.Run(tailscale, serveArgs, env)
	if err != nil {
		fmt.Fprintf(stderr, "configure Tailscale Serve: %v\n%s", err, serveOutput)
		return 1
	}
	verifiedServe, err := runner.Run(tailscale, []string{"serve", "status", "--json"}, env)
	if err != nil || !phoneServeStatusMatches(verifiedServe, *servePort, target) {
		fmt.Fprintf(stderr, "Tailscale Serve did not report the EVE mapping on port %d\n", *servePort)
		return 1
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "resolve EVE executable: %v\n", err)
		return 1
	}
	executable, _ = filepath.EvalSymlinks(executable)
	if err := installPhoneLaunchAgent(runner, executable); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	config, err := store.update(func(config *phoneConfig) error {
		if changedOrigin && *replace {
			config.Subscriptions = []phoneSubscription{}
			config.Deliveries = []phoneDelivery{}
			config.VAPIDPrivateKey = ""
		}
		config.Enabled = true
		config.RuntimeAddr = "127.0.0.1:4317"
		config.ServeHost = identity.DNSName
		config.ServePort = *servePort
		config.AllowedTailscaleLogin = identity.LoginName
		return ensurePhoneVAPIDKey(config)
	})
	if err != nil {
		fmt.Fprintf(stderr, "save phone configuration: %v\n", err)
		return 1
	}
	if err := waitForPhoneHealth("http://"+config.RuntimeAddr+"/api/health", 8*time.Second); err != nil {
		fmt.Fprintf(stderr, "EVE runtime did not become healthy: %v\n", err)
		return 1
	}
	phoneURL := config.origin() + "/phone"
	if strings.TrimSpace(os.Getenv("EVE_PHONE_SKIP_REMOTE_VERIFY")) == "" {
		if err := verifyPhoneRemote(config.origin()+"/api/phone/status", config.AllowedTailscaleLogin, 8*time.Second); err != nil {
			fmt.Fprintf(stderr, "the private EVE phone URL could not be verified: %v\n", err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "EVE Approvals is ready.")
	fmt.Fprintln(stdout)
	printPhoneQRCode(stdout, phoneURL)
	fmt.Fprintf(stdout, "\n%s\n\n", phoneURL)
	fmt.Fprintln(stdout, "1. Open the URL on an iPhone connected to this tailnet.")
	fmt.Fprintln(stdout, "2. In Safari, choose Share → Add to Home Screen.")
	fmt.Fprintln(stdout, "3. Open EVE from the Home Screen and tap Enable notifications.")
	return 0
}

func runPhoneStatus(args []string, runner phoneCommandRunner, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "eve phone status takes no arguments")
		return 2
	}
	config, err := defaultPhoneConfigStore().load()
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "EVE phone approvals are not configured. Run `eve phone setup`.")
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "EVE phone approvals: %s\n", mapBool(config.Enabled, "enabled", "disabled"))
	fmt.Fprintf(stdout, "Origin: %s\n", config.origin())
	fmt.Fprintf(stdout, "Tailscale user: %s\n", config.AllowedTailscaleLogin)
	if launchAgent, pathErr := phoneLaunchAgentPath(); pathErr == nil {
		if _, statErr := os.Stat(launchAgent); statErr == nil {
			fmt.Fprintln(stdout, "LaunchAgent: installed")
		} else {
			fmt.Fprintln(stdout, "LaunchAgent: not installed")
		}
	}
	launchDomain := "gui/" + strconv.Itoa(os.Getuid()) + "/" + phoneLaunchAgentLabel
	if _, loadedErr := runner.Run("launchctl", []string{"print", launchDomain}, nil); loadedErr == nil {
		fmt.Fprintln(stdout, "LaunchAgent state: loaded")
	} else {
		fmt.Fprintln(stdout, "LaunchAgent state: not loaded")
	}
	if phoneEndpointHealthy("http://" + config.RuntimeAddr + "/api/health") {
		fmt.Fprintln(stdout, "Runtime: healthy")
	} else {
		fmt.Fprintln(stdout, "Runtime: unavailable")
	}
	fmt.Fprintf(stdout, "Devices: %d\n", len(config.Subscriptions))
	for _, subscription := range config.Subscriptions {
		state := "never delivered"
		if subscription.LastSuccessAt != "" {
			state = "last delivered " + subscription.LastSuccessAt
		}
		if subscription.LastError != "" {
			state = "error: " + subscription.LastError
		}
		fmt.Fprintf(stdout, "  %s (%s): %s\n", subscription.Label, subscription.ID, state)
	}
	pendingDeliveries := 0
	for _, delivery := range config.Deliveries {
		if delivery.State != "delivered" && delivery.State != "skipped_existing" {
			pendingDeliveries++
		}
	}
	fmt.Fprintf(stdout, "Pending notification work: %d\n", pendingDeliveries)
	if tailscale, env, locateErr := locateTailscaleCLI(); locateErr == nil {
		if version, versionErr := runner.Run(tailscale, []string{"version"}, env); versionErr == nil {
			fmt.Fprintf(stdout, "Tailscale version: %s\n", strings.TrimSpace(string(version)))
		}
		if status, statusErr := runner.Run(tailscale, []string{"status", "--json"}, env); statusErr == nil {
			if identity, identityErr := parseTailscaleIdentity(status); identityErr == nil {
				fmt.Fprintf(stdout, "Tailscale identity: %s at %s\n", identity.LoginName, identity.DNSName)
			} else {
				fmt.Fprintf(stdout, "Tailscale identity: unavailable (%v)\n", identityErr)
			}
		}
		if output, statusErr := runner.Run(tailscale, []string{"serve", "status", "--json"}, env); statusErr == nil && phoneServeStatusMatches(output, config.ServePort, "http://127.0.0.1:4317") {
			fmt.Fprintln(stdout, "Tailscale Serve: healthy")
		} else {
			fmt.Fprintln(stdout, "Tailscale Serve: mapping not healthy")
		}
	} else {
		fmt.Fprintln(stdout, "Tailscale: CLI unavailable")
	}
	if config.Enabled && verifyPhoneRemote(config.origin()+"/api/phone/status", config.AllowedTailscaleLogin, 3*time.Second) == nil {
		fmt.Fprintln(stdout, "Private identity check: authorized")
	} else if config.Enabled {
		fmt.Fprintln(stdout, "Private identity check: unavailable")
	}
	return 0
}

func runPhoneDisable(args []string, runner phoneCommandRunner, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "eve phone disable takes no arguments")
		return 2
	}
	store := defaultPhoneConfigStore()
	config, err := store.load()
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "EVE phone approvals are already disabled.")
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if config.ServePort > 0 {
		tailscale, env, locateErr := locateTailscaleCLI()
		if locateErr != nil {
			fmt.Fprintf(stderr, "cannot safely disable phone access: %v\n", locateErr)
			return 1
		}
		if output, serveErr := runner.Run(tailscale, []string{"serve", "--https=" + strconv.Itoa(config.ServePort), "off"}, env); serveErr != nil {
			fmt.Fprintf(stderr, "disable Tailscale Serve: %v\n%s", serveErr, output)
			return 1
		}
	}
	_ = bootoutPhoneLaunchAgent(runner)
	if _, err := store.update(func(config *phoneConfig) error { config.Enabled = false; return nil }); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "EVE phone approvals are disabled. Device subscriptions were preserved.")
	return 0
}

func runPhoneReset(args []string, stdin io.Reader, runner phoneCommandRunner, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("phone reset", flag.ContinueOnError)
	fs.SetOutput(stderr)
	yes := fs.Bool("yes", false, "reset without confirmation")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve phone reset takes no positional arguments")
		return 2
	}
	if !*yes {
		fmt.Fprint(stdout, "Delete all EVE phone keys and subscriptions? [y/N] ")
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(answer), "y") && !strings.EqualFold(strings.TrimSpace(answer), "yes") {
			fmt.Fprintln(stdout, "Reset cancelled.")
			return 0
		}
	}
	if code := runPhoneDisable(nil, runner, io.Discard, stderr); code != 0 {
		return code
	}
	path := phoneConfigPath()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "EVE phone configuration was reset. Installed devices must subscribe again.")
	return 0
}

func locateTailscaleCLI() (string, []string, error) {
	if configured := strings.TrimSpace(os.Getenv("EVE_TAILSCALE_CLI")); configured != "" {
		return configured, nil, nil
	}
	if path, err := exec.LookPath("tailscale"); err == nil {
		return path, nil, nil
	}
	bundled := "/Applications/Tailscale.app/Contents/MacOS/Tailscale"
	if info, err := os.Stat(bundled); err == nil && !info.IsDir() {
		return bundled, []string{"TAILSCALE_BE_CLI=1"}, nil
	}
	return "", nil, errors.New("Tailscale is not installed; install it from https://tailscale.com/download/mac and sign in, then rerun `eve phone setup`")
}

func tailscaleVersionAtLeast(value string, major, minor int) bool {
	match := phoneVersionPattern.FindStringSubmatch(value)
	if len(match) < 3 {
		return false
	}
	gotMajor, _ := strconv.Atoi(match[1])
	gotMinor, _ := strconv.Atoi(match[2])
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

func parseTailscaleIdentity(data []byte) (tailscaleIdentity, error) {
	var status struct {
		BackendState string `json:"BackendState"`
		Self         struct {
			DNSName string `json:"DNSName"`
			UserID  int64  `json:"UserID"`
		} `json:"Self"`
		User map[string]struct {
			LoginName string `json:"LoginName"`
		} `json:"User"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return tailscaleIdentity{}, fmt.Errorf("parse Tailscale status: %w", err)
	}
	if !strings.EqualFold(status.BackendState, "Running") {
		return tailscaleIdentity{}, errors.New("Tailscale is not connected; open Tailscale and sign in")
	}
	dnsName := strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
	user := status.User[strconv.FormatInt(status.Self.UserID, 10)]
	if dnsName == "" || !strings.HasSuffix(strings.ToLower(dnsName), ".ts.net") || strings.TrimSpace(user.LoginName) == "" {
		return tailscaleIdentity{}, errors.New("Tailscale status did not include a MagicDNS host and signed-in user")
	}
	return tailscaleIdentity{DNSName: dnsName, LoginName: strings.TrimSpace(user.LoginName)}, nil
}

func phoneServePortOccupied(status []byte, port int, target string) bool {
	if len(bytes.TrimSpace(status)) == 0 || string(bytes.TrimSpace(status)) == "{}" {
		return false
	}
	text := strings.ToLower(string(status))
	mentionsPort := strings.Contains(text, `"`+strconv.Itoa(port)+`"`) || strings.Contains(text, ":"+strconv.Itoa(port))
	return mentionsPort && !strings.Contains(text, strings.ToLower(target))
}

func phoneServeStatusMatches(status []byte, port int, target string) bool {
	text := strings.ToLower(string(status))
	return (strings.Contains(text, `"`+strconv.Itoa(port)+`"`) || strings.Contains(text, ":"+strconv.Itoa(port))) && strings.Contains(text, strings.ToLower(target))
}

func phoneHomeDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv("EVE_PHONE_HOME")); value != "" {
		return filepath.Abs(value)
	}
	return os.UserHomeDir()
}

func phoneLaunchAgentPath() (string, error) {
	home, err := phoneHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", phoneLaunchAgentLabel+".plist"), nil
}

func installPhoneLaunchAgent(runner phoneCommandRunner, executable string) error {
	home, err := phoneHomeDir()
	if err != nil {
		return err
	}
	path, err := phoneLaunchAgentPath()
	if err != nil {
		return err
	}
	logDir := filepath.Join(home, "Library", "Logs", "eve")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, "phone-daemon.log")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>daemon</string><string>--addr</string><string>127.0.0.1:4317</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><true/>
<key>ProcessType</key><string>Background</string>
<key>StandardOutPath</key><string>%s</string>
<key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`, phoneLaunchAgentLabel, html.EscapeString(executable), html.EscapeString(logPath), html.EscapeString(logPath))
	if err := writeAtomicFile(path, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write phone LaunchAgent: %w", err)
	}
	if output, err := runner.Run("plutil", []string{"-lint", path}, nil); err != nil {
		return fmt.Errorf("validate phone LaunchAgent: %v: %s", err, output)
	}
	_ = bootoutPhoneLaunchAgent(runner)
	domain := "gui/" + strconv.Itoa(os.Getuid())
	if output, err := runner.Run("launchctl", []string{"bootstrap", domain, path}, nil); err != nil {
		return fmt.Errorf("load phone LaunchAgent: %v: %s", err, output)
	}
	return nil
}

func bootoutPhoneLaunchAgent(runner phoneCommandRunner) error {
	domain := "gui/" + strconv.Itoa(os.Getuid()) + "/" + phoneLaunchAgentLabel
	_, err := runner.Run("launchctl", []string{"bootout", domain}, nil)
	return err
}

func writeAtomicFile(path string, data []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".eve-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func waitForPhoneHealth(endpoint string, timeout time.Duration) error {
	if strings.TrimSpace(os.Getenv("EVE_PHONE_SKIP_HEALTH")) != "" {
		return nil
	}
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("timed out waiting for the local runtime")
}

func phoneEndpointHealthy(endpoint string) bool {
	if strings.TrimSpace(os.Getenv("EVE_PHONE_SKIP_HEALTH")) != "" {
		return true
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func verifyPhoneRemote(endpoint, expectedLogin string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("remote status returned HTTP %d", response.StatusCode)
	}
	var status struct {
		TailscaleLogin *string `json:"tailscaleLogin"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&status); err != nil {
		return fmt.Errorf("decode remote phone status: %w", err)
	}
	if status.TailscaleLogin == nil || !strings.EqualFold(strings.TrimSpace(*status.TailscaleLogin), strings.TrimSpace(expectedLogin)) {
		return errors.New("remote phone status reported a different Tailscale identity")
	}
	return nil
}

func printPhoneQRCode(w io.Writer, value string) {
	code, err := qrcode.New(value, qrcode.Medium)
	if err != nil {
		return
	}
	bitmap := code.Bitmap()
	quiet := 2
	width := len(bitmap) + quiet*2
	blank := make([]bool, width)
	rows := make([][]bool, 0, len(bitmap)+quiet*2)
	for range quiet {
		rows = append(rows, blank)
	}
	for _, source := range bitmap {
		row := make([]bool, width)
		copy(row[quiet:], source)
		rows = append(rows, row)
	}
	for range quiet {
		rows = append(rows, blank)
	}
	for row := 0; row < len(rows); row += 2 {
		upper := rows[row]
		lower := blank
		if row+1 < len(rows) {
			lower = rows[row+1]
		}
		for column := range upper {
			switch {
			case upper[column] && lower[column]:
				fmt.Fprint(w, "█")
			case upper[column]:
				fmt.Fprint(w, "▀")
			case lower[column]:
				fmt.Fprint(w, "▄")
			default:
				fmt.Fprint(w, " ")
			}
		}
		fmt.Fprintln(w)
	}
}
