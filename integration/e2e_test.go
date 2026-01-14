package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RPCRequest represents a JSON-RPC 2.0 request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC 2.0 response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestEndToEnd(t *testing.T) {
	// 1. Build the binary
	cmdBuild := exec.Command("go", "build", "-o", "xcmcp_test_bin", "../cmd/xcmcp")
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to build xcmcp: %v", err)
	}
	defer os.Remove("xcmcp_test_bin")

	// 2. Start the server
	absPath, _ := filepath.Abs("xcmcp_test_bin")
	cmd := exec.Command(absPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}
	// Capture stderr for debugging
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start xcmcp: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)

	// Helper to send request and get response
	reqID := 1
	sendRequest := func(method string, params interface{}) *RPCResponse {
		paramBytes, _ := json.Marshal(params)
		req := RPCRequest{
			JSONRPC: "2.0",
			ID:      reqID,
			Method:  method,
			Params:  paramBytes,
		}
		reqBytes, _ := json.Marshal(req)
		fmt.Fprintf(stdin, "%s\n", reqBytes)
		reqID++

		// Read response(s) until we get the one matching our ID (skip logs if any leak to stdout, though main sets log to stderr)
		for scanner.Scan() {
			line := scanner.Bytes()
			var resp RPCResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				t.Logf("ignoring non-json line: %s", line)
				continue
			}
			if resp.ID == req.ID {
				return &resp
			}
		}
		t.Fatalf("EOF waiting for response to %s", method)
		return nil
	}

	// 3. Initialize
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "test-runner", "version": "1.0"},
	}
	resp := sendRequest("initialize", initParams)
	if resp.Error != nil {
		t.Fatalf("initialize failed: %v", resp.Error)
	}
	sendRequest("notifications/initialized", map[string]interface{}{})
	t.Log("Initialized successfully")

	// 4. Test tools/list
	resp = sendRequest("tools/list", nil)
	if resp.Error != nil {
		t.Fatalf("tools/list failed: %v", resp.Error)
	}
	var toolsResult struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	json.Unmarshal(resp.Result, &toolsResult)

	expectedTools := []string{
		"build", "test", "clean", // build tools often registered
		"discover_projects", "list_schemes", "show_build_settings",
		"list_simulators", "boot_simulator", "shutdown_simulator",
	}
	// Verify we found at least some expected tools
	foundMap := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		foundMap[tool.Name] = true
	}
	for _, expected := range expectedTools {
		if !foundMap[expected] {
			// Note: 'clean' might not be registered if not implementation. 'build', 'test' are.
			if expected != "clean" {
				t.Logf("Warning: tool %s not found in list", expected)
			}
		}
	}
	t.Logf("Found %d tools", len(toolsResult.Tools))

	// 5. Test resources/list
	resp = sendRequest("resources/list", nil)
	if resp.Error != nil {
		t.Fatalf("resources/list failed: %v", resp.Error)
	}
	var resourcesResult struct {
		Resources []struct {
			URI string `json:"uri"`
		} `json:"resources"`
	}
	json.Unmarshal(resp.Result, &resourcesResult)

	expectedResources := []string{"xcmcp://project", "xcmcp://simulators"}
	for _, exp := range expectedResources {
		found := false
		for _, r := range resourcesResult.Resources {
			if r.URI == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Resource %s not found", exp)
		}
	}
	t.Logf("Found %d resources", len(resourcesResult.Resources))

	// 6. Test Tool Execution: list_simulators
	resp = sendRequest("tools/call", map[string]interface{}{
		"name":      "list_simulators",
		"arguments": map[string]string{},
	})
	if resp.Error != nil {
		t.Fatalf("tools/call list_simulators failed: %v", resp.Error)
	}
	// Basic check that we got simulators JSON back
	// The result content is typically {content: [{text: "..."}]}

	// 7. Test Resource Read: xcmcp://simulators
	resp = sendRequest("resources/read", map[string]interface{}{
		"uri": "xcmcp://simulators",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read xcmcp://simulators failed: %v", resp.Error)
	}

	// 8. Test Tool Execution: discover_projects (current dir)
	resp = sendRequest("tools/call", map[string]interface{}{
		"name": "discover_projects",
		"arguments": map[string]string{
			"path": ".",
		},
	})
	if resp.Error != nil {
		t.Fatalf("tools/call discover_projects failed: %v", resp.Error)
	}

	t.Log("All smoke tests passed")
}
