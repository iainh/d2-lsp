package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iainh/d2-lsp/internal/d2features"
)

func TestServerInitializeAdvertisesCapabilities(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params":  map[string]interface{}{},
	}), &output)
	if err != nil {
		t.Fatalf("handle initialize: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		ID     int              `json:"id"`
		Result initializeResult `json:"result"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Result.ServerInfo.Name != "d2-lsp" {
		t.Fatalf("unexpected server name %q", response.Result.ServerInfo.Name)
	}
	if response.Result.Capabilities.PositionEncoding != positionEncodingUTF16 {
		t.Fatalf("expected utf-16 position encoding, got %q", response.Result.Capabilities.PositionEncoding)
	}
	if !response.Result.Capabilities.TextDocumentSync.OpenClose {
		t.Fatal("expected openClose sync")
	}
	if response.Result.Capabilities.TextDocumentSync.Change != textDocumentSyncKindIncremental {
		t.Fatalf("expected incremental sync, got %d", response.Result.Capabilities.TextDocumentSync.Change)
	}
	if !response.Result.Capabilities.TextDocumentSync.Save.IncludeText {
		t.Fatal("expected save sync with text")
	}
	if len(response.Result.Capabilities.CompletionProvider.TriggerCharacters) == 0 {
		t.Fatal("expected completion trigger characters")
	}
	if !response.Result.Capabilities.DocumentFormattingProvider {
		t.Fatal("expected document formatting provider")
	}
	if !response.Result.Capabilities.DocumentSymbolProvider {
		t.Fatal("expected document symbol provider")
	}
	if !response.Result.Capabilities.FoldingRangeProvider {
		t.Fatal("expected folding range provider")
	}
	if !response.Result.Capabilities.ReferencesProvider {
		t.Fatal("expected references provider")
	}
	if !response.Result.Capabilities.DefinitionProvider {
		t.Fatal("expected definition provider")
	}
	if !response.Result.Capabilities.DocumentHighlightProvider {
		t.Fatal("expected document highlight provider")
	}
	if !response.Result.Capabilities.HoverProvider {
		t.Fatal("expected hover provider")
	}
	if !response.Result.Capabilities.InlayHintProvider {
		t.Fatal("expected inlay hint provider")
	}
	if !response.Result.Capabilities.SemanticTokensProvider.Full {
		t.Fatal("expected full semantic tokens provider")
	}
	if len(response.Result.Capabilities.SemanticTokensProvider.Legend.TokenTypes) == 0 {
		t.Fatal("expected semantic token legend")
	}
	if !response.Result.Capabilities.RenameProvider.PrepareProvider {
		t.Fatal("expected prepare rename provider")
	}
	if !response.Result.Capabilities.SelectionRangeProvider {
		t.Fatal("expected selection range provider")
	}
	if response.Result.Capabilities.DocumentLinkProvider.ResolveProvider {
		t.Fatal("expected document link resolve provider to be disabled")
	}
	if !response.Result.Capabilities.CodeActionProvider {
		t.Fatal("expected code action provider")
	}
	if len(response.Result.Capabilities.ExecuteCommandProvider.Commands) == 0 {
		t.Fatal("expected execute command provider")
	}
	if !response.Result.Capabilities.WorkspaceSymbolProvider {
		t.Fatal("expected workspace symbol provider")
	}
	if !response.Result.Capabilities.Workspace.WorkspaceFolders.Supported {
		t.Fatal("expected workspace folder support")
	}
	if !response.Result.Capabilities.Workspace.WorkspaceFolders.ChangeNotifications {
		t.Fatal("expected workspace folder change notifications")
	}
	if !response.Result.Capabilities.ColorProvider {
		t.Fatal("expected color provider")
	}
}

func TestServerPreservesStringRequestID(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "helix-1",
		"method":  methodInitialize,
		"params":  map[string]interface{}{},
	}), &output)
	if err != nil {
		t.Fatalf("handle initialize: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID != "helix-1" {
		t.Fatalf("expected string id to be preserved, got %q", response.ID)
	}
}

func TestInitializeStoresRootURI(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	root := t.TempDir()
	rootURI := uriFromPath(root)

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": rootURI,
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle initialize: %v", err)
	}

	if len(server.rootPaths) != 1 || server.rootPaths[0] != root {
		t.Fatalf("expected root path %q, got %#v", root, server.rootPaths)
	}
}

func TestInitializeStoresWorkspaceFolders(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	rootA := t.TempDir()
	rootB := t.TempDir()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(rootA),
			"workspaceFolders": []map[string]interface{}{
				{"uri": uriFromPath(rootA), "name": "a"},
				{"uri": uriFromPath(rootB), "name": "b"},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle initialize: %v", err)
	}

	if len(server.rootPaths) != 2 {
		t.Fatalf("expected two root paths, got %#v", server.rootPaths)
	}
	if server.rootPaths[0] != rootA || server.rootPaths[1] != rootB {
		t.Fatalf("unexpected root paths %#v", server.rootPaths)
	}
}

func TestRequestBeforeInitializeReturnsServerNotInitialized(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodTextDocumentCompletion,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      0,
				"character": 0,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle completion: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != errServerNotInitialized {
		t.Fatalf("expected ServerNotInitialized, got %#v", response.Error)
	}
}

func TestMalformedJSONReturnsParseErrorWithNullID(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	_, err := server.handle([]byte(`{"jsonrpc":"2.0",`), &output)
	if err != nil {
		t.Fatalf("handle malformed json: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		ID    *int      `json:"id"`
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID != nil {
		t.Fatalf("expected null id, got %#v", response.ID)
	}
	if response.Error == nil || response.Error.Code != errParseError {
		t.Fatalf("expected ParseError, got %#v", response.Error)
	}
}

func TestRequestWithoutMethodReturnsInvalidRequest(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
	}), &output)
	if err != nil {
		t.Fatalf("handle missing method: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != errInvalidRequest {
		t.Fatalf("expected InvalidRequest, got %#v", response.Error)
	}
}

func TestServeContinuesAfterMalformedJSON(t *testing.T) {
	server := NewServer()
	var input bytes.Buffer
	input.Write(encodeForTestRaw([]byte(`{"jsonrpc":"2.0",`)))
	input.Write(encodeForTest(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params":  map[string]interface{}{},
	}))

	var output bytes.Buffer
	if err := server.Serve(&input, &output); err != nil {
		t.Fatalf("serve: %v", err)
	}

	outputReader := bufio.NewReader(bytes.NewReader(output.Bytes()))
	first, err := readMessage(outputReader)
	if err != nil {
		t.Fatalf("read first output message: %v", err)
	}
	second, err := readMessage(outputReader)
	if err != nil {
		t.Fatalf("read second output message: %v", err)
	}

	var parseResponse struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(first, &parseResponse); err != nil {
		t.Fatalf("unmarshal parse response: %v", err)
	}
	if parseResponse.Error == nil || parseResponse.Error.Code != errParseError {
		t.Fatalf("expected ParseError, got %#v", parseResponse.Error)
	}

	var initializeResponse struct {
		Result initializeResult `json:"result"`
	}
	if err := json.Unmarshal(second, &initializeResponse); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if initializeResponse.Result.ServerInfo.Name != "d2-lsp" {
		t.Fatalf("unexpected initialize response %#v", initializeResponse.Result)
	}
}

func TestInitializedNotificationIsAcceptedAfterInitialize(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodInitialized,
		"params":  map[string]interface{}{},
	}), &output)
	if err != nil {
		t.Fatalf("handle initialized: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("initialized notification should not produce output, got %q", output.String())
	}
}

func TestDuplicateInitializeReturnsInvalidRequest(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodInitialize,
		"params":  map[string]interface{}{},
	}), &output)
	if err != nil {
		t.Fatalf("handle duplicate initialize: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != errInvalidRequest {
		t.Fatalf("expected InvalidRequest, got %#v", response.Error)
	}
}

func TestSuccessfulNilResultIsEncodedAsNullResult(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer

	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodShutdown,
	}), &output)
	if err != nil {
		t.Fatalf("handle shutdown: %v", err)
	}

	message := readOutputMessage(t, &output)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(message, &raw); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := raw["result"]; !ok {
		t.Fatalf("expected result member in %s", message)
	}
	if string(raw["result"]) != "null" {
		t.Fatalf("expected null result, got %s", raw["result"])
	}
	if _, ok := raw["error"]; ok {
		t.Fatalf("did not expect error member in %s", message)
	}
}

func TestRequestAfterShutdownReturnsInvalidRequest(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodShutdown,
	}), &output)
	if err != nil {
		t.Fatalf("handle shutdown: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  methodTextDocumentDocumentSymbol,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle documentSymbol after shutdown: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != errInvalidRequest {
		t.Fatalf("expected InvalidRequest, got %#v", response.Error)
	}
}

func TestErrorResponseOmitsResult(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "unknown/method",
	}), &output)
	if err != nil {
		t.Fatalf("handle unknown method: %v", err)
	}

	message := readOutputMessage(t, &output)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(message, &raw); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := raw["error"]; !ok {
		t.Fatalf("expected error member in %s", message)
	}
	if _, ok := raw["result"]; ok {
		t.Fatalf("did not expect result member in %s", message)
	}
}

func TestDidOpenStoresDocumentAndPublishesDiagnostics(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	uri := "file:///diagram.d2"
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidOpen,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        uri,
				"languageId": "d2",
				"version":    3,
				"text":       "x: {\n",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didOpen: %v", err)
	}

	doc, ok := server.DocumentForTest(uri)
	if !ok {
		t.Fatal("expected document to be stored")
	}
	if doc.Text != "x: {\n" || doc.Version != 3 {
		t.Fatalf("unexpected stored document %#v", doc)
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Method != methodTextDocumentPublishDiagnostic {
		t.Fatalf("unexpected method %q", notification.Method)
	}
	if notification.Params.URI != uri {
		t.Fatalf("unexpected uri %q", notification.Params.URI)
	}
	if len(notification.Params.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if notification.Params.Version == nil || *notification.Params.Version != 3 {
		t.Fatalf("unexpected version %#v", notification.Params.Version)
	}
}

func TestDidChangeReplacesDocumentText(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	uri := "file:///diagram.d2"
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidChange,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":     uri,
				"version": 4,
			},
			"contentChanges": []map[string]interface{}{
				{"text": "a -> b\n"},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didChange: %v", err)
	}

	doc, ok := server.DocumentForTest(uri)
	if !ok {
		t.Fatal("expected document to be stored")
	}
	if doc.Text != "a -> b\n" || doc.Version != 4 {
		t.Fatalf("unexpected stored document %#v", doc)
	}

	notification := readDiagnosticsNotification(t, &output)
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected clean diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestDidChangeAppliesIncrementalTextEdit(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)

	uri := "file:///diagram.d2"
	server.setDocument(document{URI: uri, Version: 1, Text: "a -> b\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidChange,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":     uri,
				"version": 2,
			},
			"contentChanges": []map[string]interface{}{
				{
					"range": map[string]interface{}{
						"start": map[string]interface{}{
							"line":      0,
							"character": len("a -> "),
						},
						"end": map[string]interface{}{
							"line":      0,
							"character": len("a -> b"),
						},
					},
					"text": "c",
				},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didChange: %v", err)
	}

	doc, ok := server.DocumentForTest(uri)
	if !ok {
		t.Fatal("expected document to be stored")
	}
	if doc.Text != "a -> c\n" || doc.Version != 2 {
		t.Fatalf("unexpected stored document %#v", doc)
	}

	notification := readDiagnosticsNotification(t, &output)
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected clean diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestDidSaveUpdatesDocumentTextWhenIncluded(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)

	uri := "file:///diagram.d2"
	server.setDocument(document{URI: uri, Version: 1, Text: "x: {\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidSave,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": uri,
			},
			"text": "a -> b\n",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didSave: %v", err)
	}

	doc, ok := server.DocumentForTest(uri)
	if !ok {
		t.Fatal("expected document to be stored")
	}
	if doc.Text != "a -> b\n" {
		t.Fatalf("unexpected stored text %q", doc.Text)
	}

	notification := readDiagnosticsNotification(t, &output)
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected clean diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestDidSavePublishesDiagnosticsForStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)

	uri := "file:///diagram.d2"
	server.setDocument(document{URI: uri, Version: 5, Text: "x: {\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidSave,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": uri,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didSave: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if len(notification.Params.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if notification.Params.Version == nil || *notification.Params.Version != 5 {
		t.Fatalf("unexpected diagnostic version %#v", notification.Params.Version)
	}
}

func TestDidOpenDiagnosticsUseOpenImportedDocument(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.d2")
	importPath := filepath.Join(root, "ok.d2")
	if err := os.WriteFile(importPath, []byte("okay: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write invalid import: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	server.setDocument(document{
		URI:     uriFromPath(importPath),
		Version: 1,
		Text:    "okay\n",
	})
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidOpen,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        uriFromPath(indexPath),
				"languageId": "d2",
				"version":    2,
				"text":       "hey: @ok\nhey.okay\n",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didOpen: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected open import buffer to suppress disk diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestInitializedPublishesWorkspaceDiagnostics(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("x: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodInitialized,
	}), &output)
	if err != nil {
		t.Fatalf("handle initialized: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Params.URI != uriFromPath(diagramPath) {
		t.Fatalf("unexpected diagnostics uri %q", notification.Params.URI)
	}
	if len(notification.Params.Diagnostics) == 0 {
		t.Fatal("expected workspace diagnostics")
	}
	if notification.Params.Version != nil {
		t.Fatalf("expected nil version for workspace diagnostics, got %#v", notification.Params.Version)
	}
}

func TestInitializedHonorsDiagnosticsConfiguration(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("x: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
			"initializationOptions": map[string]interface{}{
				"diagnosticsOnInitialize": false,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodInitialized,
	}), &output)
	if err != nil {
		t.Fatalf("handle initialized: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no initialized diagnostics, got %q", output.String())
	}
}

func TestInitializedWorkspaceDiagnosticsUseOpenImportedDocument(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.d2")
	importPath := filepath.Join(root, "ok.d2")
	if err := os.WriteFile(indexPath, []byte("hey: @ok\nhey.okay\n"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(importPath, []byte("okay: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write invalid import: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	server.setDocument(document{
		URI:     uriFromPath(importPath),
		Version: 1,
		Text:    "okay\n",
	})
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodInitialized,
	}), &output)
	if err != nil {
		t.Fatalf("handle initialized: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Params.URI != uriFromPath(indexPath) {
		t.Fatalf("unexpected diagnostics uri %q", notification.Params.URI)
	}
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected open import buffer to suppress disk diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestDidChangeConfigurationUpdatesDiagnosticsSettings(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("x: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodWorkspaceDidChangeConfig,
		"params": map[string]interface{}{
			"settings": map[string]interface{}{
				"d2-lsp": map[string]interface{}{
					"diagnosticsOnWatchedFiles": false,
				},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/didChangeConfiguration: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no response to configuration change, got %q", output.String())
	}

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodWorkspaceDidChangeWatchedFiles,
		"params": map[string]interface{}{
			"changes": []map[string]interface{}{
				{"uri": uriFromPath(diagramPath), "type": 2},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/didChangeWatchedFiles: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("expected watched diagnostics to be disabled, got %q", output.String())
	}
}

func TestDidChangeWatchedFilesPublishesWorkspaceDiagnostics(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("x: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodWorkspaceDidChangeWatchedFiles,
		"params": map[string]interface{}{
			"changes": []map[string]interface{}{
				{"uri": uriFromPath(diagramPath), "type": 2},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/didChangeWatchedFiles: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Params.URI != uriFromPath(diagramPath) {
		t.Fatalf("unexpected diagnostics uri %q", notification.Params.URI)
	}
	if len(notification.Params.Diagnostics) == 0 {
		t.Fatal("expected workspace diagnostics after watched file change")
	}
}

func TestDidChangeWatchedFilesClearsDeletedD2Diagnostics(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("x: {shape: not-a-shape}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := os.Remove(diagramPath); err != nil {
		t.Fatalf("remove diagram: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodWorkspaceDidChangeWatchedFiles,
		"params": map[string]interface{}{
			"changes": []map[string]interface{}{
				{"uri": uriFromPath(diagramPath), "type": fileChangeTypeDeleted},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/didChangeWatchedFiles: %v", err)
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Params.URI != uriFromPath(diagramPath) {
		t.Fatalf("unexpected diagnostics uri %q", notification.Params.URI)
	}
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected cleared diagnostics for deleted file, got %#v", notification.Params.Diagnostics)
	}
}

func TestDidCloseDeletesDocumentAndClearsDiagnostics(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x: {\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodTextDocumentDidClose,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle didClose: %v", err)
	}

	if _, ok := server.DocumentForTest("file:///diagram.d2"); ok {
		t.Fatal("expected document to be deleted")
	}

	notification := readDiagnosticsNotification(t, &output)
	if notification.Params.Version != nil {
		t.Fatalf("expected nil version, got %#v", notification.Params.Version)
	}
	if len(notification.Params.Diagnostics) != 0 {
		t.Fatalf("expected cleared diagnostics, got %#v", notification.Params.Diagnostics)
	}
}

func TestCompletionUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x: { style."})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentCompletion,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      0,
				"character": len("x: { style."),
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle completion: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result completionList `json:"result"`
		Error  *rpcError      `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal completion response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected completion error: %#v", response.Error)
	}
	if len(response.Result.Items) == 0 {
		t.Fatal("expected completion items")
	}
}

func TestFormattingReturnsWholeDocumentEdit(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x:{y:z}\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentFormatting,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"options": map[string]interface{}{
				"tabSize":      2,
				"insertSpaces": true,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle formatting: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []textEdit `json:"result"`
		Error  *rpcError  `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal formatting response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected formatting error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one text edit, got %#v", response.Result)
	}
	if response.Result[0].NewText != "x: {y: z}\n" {
		t.Fatalf("unexpected formatted text %q", response.Result[0].NewText)
	}
	if response.Result[0].Range.End.Line != 1 || response.Result[0].Range.End.Character != 0 {
		t.Fatalf("unexpected range %#v", response.Result[0].Range)
	}
}

func TestFormattingReturnsNoEditsForInvalidDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x: {\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentFormatting,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"options": map[string]interface{}{
				"tabSize":      2,
				"insertSpaces": true,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle formatting: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []textEdit `json:"result"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal formatting response: %v", err)
	}
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits, got %#v", response.Result)
	}
}

func TestCodeActionReturnsFormatSourceAction(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x:{y:z}\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentCodeAction,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 0},
			},
			"context": map[string]interface{}{},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle codeAction: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []codeAction `json:"result"`
		Error  *rpcError    `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal codeAction response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected codeAction error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one code action, got %#v", response.Result)
	}
	action := response.Result[0]
	if action.Title != "Format D2 document" || action.Kind != "source.format" {
		t.Fatalf("unexpected code action %#v", action)
	}
	edits := action.Edit.Changes["file:///diagram.d2"]
	if len(edits) != 1 || edits[0].NewText != "x: {y: z}\n" {
		t.Fatalf("unexpected code action edit %#v", action.Edit)
	}
}

func TestCodeActionHonorsOnlyFilter(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{URI: "file:///diagram.d2", Version: 1, Text: "x:{y:z}\n"})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentCodeAction,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 0},
			},
			"context": map[string]interface{}{
				"only": []string{"quickfix"},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle codeAction: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []codeAction `json:"result"`
		Error  *rpcError    `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal codeAction response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected codeAction error: %#v", response.Error)
	}
	if len(response.Result) != 0 {
		t.Fatalf("expected no code actions, got %#v", response.Result)
	}
}

func TestEndPositionUsesUTF16Characters(t *testing.T) {
	got := endPosition("a🙂\nb")
	want := position{Line: 1, Character: 1}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestDocumentSymbolUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "server: {\n  shape: rectangle\n}\nserver -> database\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDocumentSymbol,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle documentSymbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.DocumentSymbol `json:"result"`
		Error  *rpcError                   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal documentSymbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected documentSymbol error: %#v", response.Error)
	}
	if len(response.Result) != 2 {
		t.Fatalf("expected two symbols, got %#v", response.Result)
	}
	if response.Result[0].Name != "server" {
		t.Fatalf("unexpected first symbol %q", response.Result[0].Name)
	}
	if len(response.Result[0].Children) != 1 {
		t.Fatalf("expected child symbol, got %#v", response.Result[0].Children)
	}
}

func TestWorkspaceSymbolReturnsWorkspaceSymbols(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("server: {shape: rectangle}\ndatabase\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceSymbol,
		"params": map[string]interface{}{
			"query": "serv",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/symbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []workspaceSymbol `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal workspace/symbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected workspace/symbol error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one symbol, got %#v", response.Result)
	}
	if response.Result[0].Name != "server" {
		t.Fatalf("unexpected symbol %#v", response.Result[0])
	}
	if response.Result[0].Location.URI != uriFromPath(diagramPath) {
		t.Fatalf("unexpected symbol uri %#v", response.Result[0].Location)
	}
}

func TestWorkspaceSymbolReturnsEmptyArrayWhenNoSymbolsMatch(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("server: {shape: rectangle}\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceSymbol,
		"params": map[string]interface{}{
			"query": "missing",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/symbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []workspaceSymbol `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal workspace/symbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected workspace/symbol error: %#v", response.Error)
	}
	if response.Result == nil {
		t.Fatal("expected empty workspace symbol slice, got nil")
	}
	if len(response.Result) != 0 {
		t.Fatalf("expected no symbols, got %#v", response.Result)
	}
}

func TestWorkspaceSymbolUsesWorkspaceFolders(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	pathA := filepath.Join(rootA, "a.d2")
	pathB := filepath.Join(rootB, "b.d2")
	if err := os.WriteFile(pathA, []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"workspaceFolders": []map[string]interface{}{
				{"uri": uriFromPath(rootA), "name": "a"},
				{"uri": uriFromPath(rootB), "name": "b"},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceSymbol,
		"params": map[string]interface{}{
			"query": "",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/symbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []workspaceSymbol `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal workspace/symbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected workspace/symbol error: %#v", response.Error)
	}
	if len(response.Result) != 2 {
		t.Fatalf("expected two symbols, got %#v", response.Result)
	}
	names := map[string]bool{}
	for _, symbol := range response.Result {
		names[symbol.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Fatalf("expected symbols from both roots, got %#v", response.Result)
	}
}

func TestWorkspaceSymbolUsesChangedWorkspaceFolders(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	pathA := filepath.Join(rootA, "a.d2")
	pathB := filepath.Join(rootB, "b.d2")
	if err := os.WriteFile(pathA, []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"workspaceFolders": []map[string]interface{}{
				{"uri": uriFromPath(rootA), "name": "a"},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  methodWorkspaceDidChangeFolders,
		"params": map[string]interface{}{
			"event": map[string]interface{}{
				"added": []map[string]interface{}{
					{"uri": uriFromPath(rootB), "name": "b"},
				},
				"removed": []map[string]interface{}{
					{"uri": uriFromPath(rootA), "name": "a"},
				},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/didChangeWorkspaceFolders: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no response to workspace folder notification, got %q", output.String())
	}
	if len(server.rootPaths) != 1 || server.rootPaths[0] != rootB {
		t.Fatalf("expected changed root path %q, got %#v", rootB, server.rootPaths)
	}

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceSymbol,
		"params": map[string]interface{}{
			"query": "",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/symbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []workspaceSymbol `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal workspace/symbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected workspace/symbol error: %#v", response.Error)
	}
	if len(response.Result) != 1 || response.Result[0].Name != "beta" {
		t.Fatalf("expected only beta from changed roots, got %#v", response.Result)
	}
}

func TestWorkspaceSymbolUsesOpenDocumentOverDisk(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	if err := os.WriteFile(diagramPath, []byte("disk\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	server.setDocument(document{
		URI:     uriFromPath(diagramPath),
		Version: 1,
		Text:    "buffer\n",
	})
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceSymbol,
		"params": map[string]interface{}{
			"query": "",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle workspace/symbol: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []workspaceSymbol `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal workspace/symbol response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected workspace/symbol error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one symbol, got %#v", response.Result)
	}
	if response.Result[0].Name != "buffer" {
		t.Fatalf("expected open-buffer symbol, got %#v", response.Result[0])
	}
}

func TestExecuteCommandRendersSVG(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodWorkspaceExecuteCommand,
		"params": map[string]interface{}{
			"command": commandRenderSVG,
			"arguments": []map[string]interface{}{
				{"textDocument": map[string]interface{}{"uri": "file:///diagram.d2"}},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle executeCommand: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result renderSVGResult `json:"result"`
		Error  *rpcError       `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal executeCommand response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected executeCommand error: %#v", response.Error)
	}
	if response.Result.URI != "file:///diagram.d2" {
		t.Fatalf("unexpected render uri %q", response.Result.URI)
	}
	if response.Result.MimeType != "image/svg+xml" {
		t.Fatalf("unexpected render mime type %q", response.Result.MimeType)
	}
	if !strings.Contains(response.Result.Content, "<svg") {
		t.Fatalf("expected svg content, got %q", response.Result.Content[:min(len(response.Result.Content), 80)])
	}
}

func TestFoldingRangeUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "server: {\n  api: {\n    shape: rectangle\n  }\n}\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentFoldingRange,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle foldingRange: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.FoldingRange `json:"result"`
		Error  *rpcError                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal foldingRange response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected foldingRange error: %#v", response.Error)
	}
	if len(response.Result) != 2 {
		t.Fatalf("expected two folding ranges, got %#v", response.Result)
	}
	if response.Result[0].StartLine != 0 || response.Result[0].EndLine != 4 {
		t.Fatalf("unexpected first range %#v", response.Result[0])
	}
}

func TestReferencesUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentReferences,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
			"context": map[string]interface{}{
				"includeDeclaration": true,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle references: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.Location `json:"result"`
		Error  *rpcError             `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal references response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected references error: %#v", response.Error)
	}
	if len(response.Result) != 2 {
		t.Fatalf("expected two locations, got %#v", response.Result)
	}
	if response.Result[0].URI != "file:///diagram.d2" {
		t.Fatalf("unexpected uri %q", response.Result[0].URI)
	}
}

func TestPrepareRenameReturnsCurrentReferenceRange(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentPrepareRename,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle prepareRename: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result rangePosition `json:"result"`
		Error  *rpcError     `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal prepareRename response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected prepareRename error: %#v", response.Error)
	}
	if response.Result.Start.Line != 1 || response.Result.Start.Character != 0 || response.Result.End.Character != 1 {
		t.Fatalf("unexpected prepareRename range %#v", response.Result)
	}
}

func TestRenameReturnsWorkspaceEditForReferences(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentRename,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
			"newName": "renamed",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle rename: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result workspaceEdit `json:"result"`
		Error  *rpcError     `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal rename response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected rename error: %#v", response.Error)
	}
	edits := response.Result.Changes["file:///diagram.d2"]
	if len(edits) != 2 {
		t.Fatalf("expected two edits, got %#v", response.Result.Changes)
	}
	for _, edit := range edits {
		if edit.NewText != "renamed" {
			t.Fatalf("unexpected edit text %#v", edit)
		}
	}
}

func TestRenameRejectsInvalidNewName(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentRename,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
			"newName": "bad: value",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle rename: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Error *rpcError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal rename response: %v", err)
	}
	if response.Error == nil || response.Error.Code != errInvalidParams {
		t.Fatalf("expected invalid params error, got %#v", response.Error)
	}
}

func TestRenameReturnsWorkspaceEditForImportedReferences(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///workspace/index.d2",
		Version: 1,
		Text:    "hey: @ok\nhey.okay\n",
	})
	server.setDocument(document{
		URI:     "file:///workspace/ok.d2",
		Version: 1,
		Text:    "okay\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentRename,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///workspace/index.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": len("hey.ok"),
			},
			"newName": "renamed",
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle rename: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result workspaceEdit `json:"result"`
		Error  *rpcError     `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal rename response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected rename error: %#v", response.Error)
	}
	if len(response.Result.Changes["file:///workspace/ok.d2"]) != 1 {
		t.Fatalf("expected imported file edit, got %#v", response.Result.Changes)
	}
	if response.Result.Changes["file:///workspace/ok.d2"][0].NewText != "renamed" {
		t.Fatalf("unexpected imported edit %#v", response.Result.Changes["file:///workspace/ok.d2"][0])
	}
}

func TestDefinitionUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDefinition,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle definition: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result *d2features.Location `json:"result"`
		Error  *rpcError            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal definition response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected definition error: %#v", response.Error)
	}
	if response.Result == nil {
		t.Fatal("expected definition location")
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("expected definition on line 0, got %#v", response.Result)
	}
}

func TestDefinitionUsesImportedOpenDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///workspace/index.d2",
		Version: 1,
		Text:    "hey: @ok\nhey.okay\n",
	})
	server.setDocument(document{
		URI:     "file:///workspace/ok.d2",
		Version: 1,
		Text:    "okay\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDefinition,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///workspace/index.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": len("hey.ok"),
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle definition: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result *d2features.Location `json:"result"`
		Error  *rpcError            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal definition response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected definition error: %#v", response.Error)
	}
	if response.Result == nil {
		t.Fatal("expected definition location")
	}
	if response.Result.URI != "file:///workspace/ok.d2" {
		t.Fatalf("unexpected definition uri %q", response.Result.URI)
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("expected definition on imported line 0, got %#v", response.Result)
	}
}

func TestDefinitionUsesImportedWorkspaceDocument(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.d2")
	importPath := filepath.Join(root, "ok.d2")
	if err := os.WriteFile(importPath, []byte("okay\n"), 0644); err != nil {
		t.Fatalf("write import: %v", err)
	}

	server := NewServer()
	var output bytes.Buffer
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params": map[string]interface{}{
			"rootUri": uriFromPath(root),
		},
	}), &output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	server.setDocument(document{
		URI:     uriFromPath(indexPath),
		Version: 1,
		Text:    "hey: @ok\nhey.okay\n",
	})
	output.Reset()

	_, err = server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDefinition,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": uriFromPath(indexPath),
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": len("hey.ok"),
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle definition: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result *d2features.Location `json:"result"`
		Error  *rpcError            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal definition response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected definition error: %#v", response.Error)
	}
	if response.Result == nil {
		t.Fatal("expected definition location")
	}
	if response.Result.URI != uriFromPath(importPath) {
		t.Fatalf("unexpected definition uri %q", response.Result.URI)
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("expected definition on imported line 0, got %#v", response.Result)
	}
}

func TestDocumentHighlightUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x\nx -> y\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDocumentHighlight,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      1,
				"character": 0,
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle documentHighlight: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.DocumentHighlight `json:"result"`
		Error  *rpcError                      `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal documentHighlight response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected documentHighlight error: %#v", response.Error)
	}
	if len(response.Result) != 2 {
		t.Fatalf("expected two highlights, got %#v", response.Result)
	}
	if response.Result[0].Range.Start.Line != 0 {
		t.Fatalf("expected first highlight on line 0, got %#v", response.Result[0])
	}
}

func TestHoverUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x: {shape: rectangle}\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentHover,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"position": map[string]interface{}{
				"line":      0,
				"character": len("x: {shape: rec"),
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle hover: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result *d2features.Hover `json:"result"`
		Error  *rpcError         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal hover response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected hover error: %#v", response.Error)
	}
	if response.Result == nil {
		t.Fatal("expected hover")
	}
	if response.Result.Contents.Kind != "markdown" {
		t.Fatalf("unexpected hover kind %q", response.Result.Contents.Kind)
	}
	if response.Result.Contents.Value != "`rectangle` is a D2 shape." {
		t.Fatalf("unexpected hover content %q", response.Result.Contents.Value)
	}
}

func TestInlayHintReturnsImportPathHints(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///workspace/index.d2",
		Version: 1,
		Text:    "hey: @ok\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentInlayHint,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///workspace/index.d2",
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 20},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle inlayHint: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.InlayHint `json:"result"`
		Error  *rpcError              `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal inlayHint response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected inlayHint error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one inlay hint, got %#v", response.Result)
	}
	if response.Result[0].Label != " => /workspace/ok.d2" {
		t.Fatalf("unexpected inlay hint %#v", response.Result[0])
	}
}

func TestInlayHintFiltersByRange(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///workspace/index.d2",
		Version: 1,
		Text:    "hey: @ok\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentInlayHint,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///workspace/index.d2",
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 3},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle inlayHint: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.InlayHint `json:"result"`
		Error  *rpcError              `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal inlayHint response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected inlayHint error: %#v", response.Error)
	}
	if len(response.Result) != 0 {
		t.Fatalf("expected no inlay hints, got %#v", response.Result)
	}
}

func TestSemanticTokensUseStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "server: {shape: rectangle}\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentSemanticTokensFull,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle semanticTokens/full: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result d2features.SemanticTokens `json:"result"`
		Error  *rpcError                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal semanticTokens response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected semanticTokens error: %#v", response.Error)
	}
	if len(response.Result.Data) == 0 {
		t.Fatal("expected semantic token data")
	}
	if len(response.Result.Data)%5 != 0 {
		t.Fatalf("expected semantic token data groups of five, got %#v", response.Result.Data)
	}
}

func TestSelectionRangeUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "server: {shape: rectangle}\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentSelectionRange,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"positions": []map[string]interface{}{
				{
					"line":      0,
					"character": len("server: {shape: rec"),
				},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle selectionRange: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []*d2features.SelectionRange `json:"result"`
		Error  *rpcError                    `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal selectionRange response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected selectionRange error: %#v", response.Error)
	}
	if len(response.Result) != 1 || response.Result[0] == nil {
		t.Fatalf("expected one selection range, got %#v", response.Result)
	}
	if response.Result[0].Range.Start.Character != len("server: {shape: ") {
		t.Fatalf("unexpected selection range %#v", response.Result[0])
	}
	if response.Result[0].Parent == nil {
		t.Fatal("expected parent selection range")
	}
}

func TestDocumentLinkUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x: {link: https://example.com}\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDocumentLink,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle documentLink: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.DocumentLink `json:"result"`
		Error  *rpcError                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal documentLink response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected documentLink error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one document link, got %#v", response.Result)
	}
	if response.Result[0].Target != "https://example.com" {
		t.Fatalf("unexpected document link target %q", response.Result[0].Target)
	}
}

func TestDocumentColorUsesStoredDocument(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	server.setDocument(document{
		URI:     "file:///diagram.d2",
		Version: 1,
		Text:    "x.style.fill: '#00ff00'\n",
	})
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentDocumentColor,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle documentColor: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.DocumentColor `json:"result"`
		Error  *rpcError                  `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal documentColor response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected documentColor error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one document color, got %#v", response.Result)
	}
	if response.Result[0].Color.Green != 1 {
		t.Fatalf("unexpected document color %#v", response.Result[0])
	}
}

func TestColorPresentationReturnsHexLabel(t *testing.T) {
	server := NewServer()
	var output bytes.Buffer
	initialize(t, server, &output)
	output.Reset()

	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  methodTextDocumentColorPresentation,
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file:///diagram.d2",
			},
			"color": map[string]interface{}{
				"red":   1,
				"green": 0.5,
				"blue":  0,
				"alpha": 1,
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 7},
			},
		},
	}), &output)
	if err != nil {
		t.Fatalf("handle colorPresentation: %v", err)
	}

	message := readOutputMessage(t, &output)
	var response struct {
		Result []d2features.ColorPresentation `json:"result"`
		Error  *rpcError                      `json:"error,omitempty"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal colorPresentation response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected colorPresentation error: %#v", response.Error)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one presentation, got %#v", response.Result)
	}
	if response.Result[0].Label != "#ff8000" {
		t.Fatalf("unexpected presentation %#v", response.Result[0])
	}
	if response.Result[0].TextEdit.NewText != "#ff8000" {
		t.Fatalf("unexpected presentation text edit %#v", response.Result[0].TextEdit)
	}
	if response.Result[0].TextEdit.Range.Start.Line != 0 || response.Result[0].TextEdit.Range.End.Character != 7 {
		t.Fatalf("unexpected presentation text edit range %#v", response.Result[0].TextEdit.Range)
	}
}

func initialize(t *testing.T, server *Server, output *bytes.Buffer) {
	t.Helper()
	_, err := server.handle(mustMarshal(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  methodInitialize,
		"params":  map[string]interface{}{},
	}), output)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

func readOutputMessage(t *testing.T, output *bytes.Buffer) []byte {
	t.Helper()
	msg, err := readMessage(bufio.NewReader(bytes.NewReader(output.Bytes())))
	if err != nil {
		t.Fatalf("read output message: %v", err)
	}
	return msg
}

func readDiagnosticsNotification(t *testing.T, output *bytes.Buffer) struct {
	Method string                   `json:"method"`
	Params publishDiagnosticsParams `json:"params"`
} {
	t.Helper()

	msg := readOutputMessage(t, output)
	var notification struct {
		Method string                   `json:"method"`
		Params publishDiagnosticsParams `json:"params"`
	}
	if err := json.Unmarshal(msg, &notification); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}
	return notification
}

func mustMarshal(t *testing.T, value interface{}) []byte {
	t.Helper()
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}
