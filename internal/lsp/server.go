package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/iainh/d2-lsp/internal/d2diagnostics"
	"github.com/iainh/d2-lsp/internal/d2features"
)

const (
	jsonRPCVersion = "2.0"

	positionEncodingUTF16 = "utf-16"

	textDocumentSyncKindIncremental = 2

	methodInitialize                     = "initialize"
	methodInitialized                    = "initialized"
	methodShutdown                       = "shutdown"
	methodExit                           = "exit"
	methodTextDocumentDidOpen            = "textDocument/didOpen"
	methodTextDocumentDidChange          = "textDocument/didChange"
	methodTextDocumentDidClose           = "textDocument/didClose"
	methodTextDocumentDidSave            = "textDocument/didSave"
	methodTextDocumentPublishDiagnostic  = "textDocument/publishDiagnostics"
	methodTextDocumentCompletion         = "textDocument/completion"
	methodTextDocumentFormatting         = "textDocument/formatting"
	methodTextDocumentDocumentSymbol     = "textDocument/documentSymbol"
	methodTextDocumentFoldingRange       = "textDocument/foldingRange"
	methodTextDocumentReferences         = "textDocument/references"
	methodTextDocumentDefinition         = "textDocument/definition"
	methodTextDocumentDocumentHighlight  = "textDocument/documentHighlight"
	methodTextDocumentHover              = "textDocument/hover"
	methodTextDocumentSemanticTokensFull = "textDocument/semanticTokens/full"
	methodTextDocumentPrepareRename      = "textDocument/prepareRename"
	methodTextDocumentRename             = "textDocument/rename"
	methodTextDocumentSelectionRange     = "textDocument/selectionRange"
	methodTextDocumentDocumentLink       = "textDocument/documentLink"
	methodTextDocumentDocumentColor      = "textDocument/documentColor"
	methodTextDocumentColorPresentation  = "textDocument/colorPresentation"
	methodTextDocumentCodeAction         = "textDocument/codeAction"
	methodWorkspaceSymbol                = "workspace/symbol"
	methodWorkspaceDidChangeFolders      = "workspace/didChangeWorkspaceFolders"
	methodWorkspaceDidChangeWatchedFiles = "workspace/didChangeWatchedFiles"
)

const fileChangeTypeDeleted = 3

const (
	errParseError           = -32700
	errMethodNotFound       = -32601
	errInvalidParams        = -32602
	errInternalError        = -32603
	errServerNotInitialized = -32002
	errInvalidRequest       = -32600
)

var nullID = json.RawMessage("null")

type Server struct {
	mu        sync.Mutex
	ready     bool
	shutdown  bool
	rootPaths []string
	documents map[string]document
}

type document struct {
	URI     string
	Version int
	Text    string
}

func NewServer() *Server {
	return &Server{
		documents: make(map[string]document),
	}
}

func (s *Server) Serve(reader io.Reader, writer io.Writer) error {
	bufferedReader := bufio.NewReader(reader)
	for {
		body, err := readMessage(bufferedReader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		shouldExit, err := s.handle(body, writer)
		if err != nil {
			return err
		}
		if shouldExit {
			return nil
		}
	}
}

func (s *Server) handle(body []byte, writer io.Writer) (bool, error) {
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id,omitempty"`
		Method  string          `json:"method,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false, writeError(writer, nullID, errParseError, "parse error")
	}

	if envelope.Method == methodExit {
		return true, nil
	}

	if len(envelope.ID) == 0 {
		err := s.handleNotification(envelope.Method, envelope.Params, writer)
		return false, err
	}

	result, rpcErr := s.handleRequest(envelope.Method, envelope.Params)
	response := responseMessage{
		JSONRPC: jsonRPCVersion,
		ID:      envelope.ID,
		Result:  result,
		Error:   rpcErr,
	}
	return false, writeJSON(writer, response)
}

func (s *Server) handleRequest(method string, params json.RawMessage) (interface{}, *rpcError) {
	if method == "" {
		return nil, &rpcError{Code: errInvalidRequest, Message: "missing method"}
	}
	if rpcErr := s.lifecycleRequestError(method); rpcErr != nil {
		return nil, rpcErr
	}

	switch method {
	case methodInitialize:
		var initParams initializeParams
		if len(params) > 0 {
			if err := json.Unmarshal(params, &initParams); err != nil {
				return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
			}
		}

		s.mu.Lock()
		s.ready = true
		s.rootPaths = rootPathsFromInitialize(initParams)
		s.mu.Unlock()

		return initializeResult{
			Capabilities: serverCapabilities{
				PositionEncoding: positionEncodingUTF16,
				TextDocumentSync: textDocumentSyncOptions{
					OpenClose: true,
					Change:    textDocumentSyncKindIncremental,
					Save: textDocumentSyncSaveOptions{
						IncludeText: true,
					},
				},
				CompletionProvider: completionOptions{
					TriggerCharacters: []string{".", ":"},
				},
				DocumentFormattingProvider: true,
				DocumentSymbolProvider:     true,
				FoldingRangeProvider:       true,
				ReferencesProvider:         true,
				DefinitionProvider:         true,
				DocumentHighlightProvider:  true,
				HoverProvider:              true,
				SemanticTokensProvider: semanticTokensOptions{
					Legend: semanticTokensLegend{
						TokenTypes:     d2features.SemanticTokenTypes,
						TokenModifiers: []string{},
					},
					Full: true,
				},
				RenameProvider: renameOptions{
					PrepareProvider: true,
				},
				SelectionRangeProvider: true,
				DocumentLinkProvider: documentLinkOptions{
					ResolveProvider: false,
				},
				CodeActionProvider:      true,
				WorkspaceSymbolProvider: true,
				Workspace: workspaceOptions{
					WorkspaceFolders: workspaceFoldersServerCapabilities{
						Supported:           true,
						ChangeNotifications: true,
					},
				},
				ColorProvider: true,
			},
			ServerInfo: serverInfo{
				Name:    "d2-lsp",
				Version: "0.1.0",
			},
		}, nil
	case methodTextDocumentCompletion:
		var completion completionParams
		if err := json.Unmarshal(params, &completion); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(completion.TextDocument.URI)
		if !ok {
			return completionList{IsIncomplete: false, Items: []d2features.CompletionItem{}}, nil
		}

		items, err := d2features.Complete(doc.Text, completion.Position.Line, completion.Position.Character)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return completionList{IsIncomplete: false, Items: items}, nil
	case methodTextDocumentFormatting:
		var formatting documentFormattingParams
		if err := json.Unmarshal(params, &formatting); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(formatting.TextDocument.URI)
		if !ok {
			return []textEdit{}, nil
		}

		formatted, changed, err := d2features.Format(doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		if !changed {
			return []textEdit{}, nil
		}

		return []textEdit{{
			Range: rangePosition{
				Start: position{Line: 0, Character: 0},
				End:   endPosition(doc.Text),
			},
			NewText: formatted,
		}}, nil
	case methodTextDocumentDocumentSymbol:
		var symbolParams documentSymbolParams
		if err := json.Unmarshal(params, &symbolParams); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(symbolParams.TextDocument.URI)
		if !ok {
			return []d2features.DocumentSymbol{}, nil
		}

		symbols, err := d2features.Symbols(doc.URI, doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return symbols, nil
	case methodTextDocumentFoldingRange:
		var foldingParams foldingRangeParams
		if err := json.Unmarshal(params, &foldingParams); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(foldingParams.TextDocument.URI)
		if !ok {
			return []d2features.FoldingRange{}, nil
		}

		ranges, err := d2features.FoldingRanges(doc.URI, doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return ranges, nil
	case methodTextDocumentReferences:
		var references referenceParams
		if err := json.Unmarshal(params, &references); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(references.TextDocument.URI)
		if !ok {
			return []d2features.Location{}, nil
		}
		path, fs, uriByPath := s.documentFilesystem(doc)

		locations, err := d2features.ReferencesInFiles(
			doc.URI,
			path,
			fs,
			uriByPath,
			references.Position.Line,
			references.Position.Character,
			references.Context.IncludeDeclaration,
		)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return locations, nil
	case methodTextDocumentDefinition:
		var definition definitionParams
		if err := json.Unmarshal(params, &definition); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(definition.TextDocument.URI)
		if !ok {
			return nil, nil
		}
		path, fs, uriByPath := s.documentFilesystem(doc)

		location, err := d2features.DefinitionInFiles(
			doc.URI,
			path,
			fs,
			uriByPath,
			definition.Position.Line,
			definition.Position.Character,
		)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return location, nil
	case methodTextDocumentDocumentHighlight:
		var highlight documentHighlightParams
		if err := json.Unmarshal(params, &highlight); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(highlight.TextDocument.URI)
		if !ok {
			return []d2features.DocumentHighlight{}, nil
		}
		path, fs, uriByPath := s.documentFilesystem(doc)

		highlights, err := d2features.DocumentHighlightsInFiles(
			doc.URI,
			path,
			fs,
			uriByPath,
			highlight.Position.Line,
			highlight.Position.Character,
		)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return highlights, nil
	case methodTextDocumentHover:
		var hover hoverParams
		if err := json.Unmarshal(params, &hover); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(hover.TextDocument.URI)
		if !ok {
			return nil, nil
		}

		result, err := d2features.HoverAt(doc.URI, doc.Text, hover.Position.Line, hover.Position.Character)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return result, nil
	case methodTextDocumentSemanticTokensFull:
		var semanticTokens semanticTokensParams
		if err := json.Unmarshal(params, &semanticTokens); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(semanticTokens.TextDocument.URI)
		if !ok {
			return d2features.SemanticTokens{}, nil
		}

		result, err := d2features.SemanticTokensFor(doc.URI, doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return result, nil
	case methodTextDocumentPrepareRename:
		var prepareRename prepareRenameParams
		if err := json.Unmarshal(params, &prepareRename); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(prepareRename.TextDocument.URI)
		if !ok {
			return nil, nil
		}
		path, fs, uriByPath := s.documentFilesystem(doc)

		locations, err := d2features.ReferencesInFiles(
			doc.URI,
			path,
			fs,
			uriByPath,
			prepareRename.Position.Line,
			prepareRename.Position.Character,
			true,
		)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		for _, location := range locations {
			if location.URI == doc.URI && containsLSPPosition(location.Range, prepareRename.Position) {
				r := rangeFromFeature(location.Range)
				return r, nil
			}
		}
		return nil, nil
	case methodTextDocumentRename:
		var rename renameParams
		if err := json.Unmarshal(params, &rename); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(rename.TextDocument.URI)
		if !ok {
			return workspaceEdit{Changes: map[string][]textEdit{}}, nil
		}
		path, fs, uriByPath := s.documentFilesystem(doc)

		locations, err := d2features.ReferencesInFiles(
			doc.URI,
			path,
			fs,
			uriByPath,
			rename.Position.Line,
			rename.Position.Character,
			true,
		)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		edit := workspaceEdit{Changes: make(map[string][]textEdit)}
		for _, location := range locations {
			edit.Changes[location.URI] = append(edit.Changes[location.URI], textEdit{
				Range:   rangeFromFeature(location.Range),
				NewText: rename.NewName,
			})
		}
		return edit, nil
	case methodTextDocumentSelectionRange:
		var selectionRange selectionRangeParams
		if err := json.Unmarshal(params, &selectionRange); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(selectionRange.TextDocument.URI)
		if !ok {
			return make([]*d2features.SelectionRange, len(selectionRange.Positions)), nil
		}

		positions := make([]d2features.Position, 0, len(selectionRange.Positions))
		for _, pos := range selectionRange.Positions {
			positions = append(positions, d2features.Position{
				Line:      pos.Line,
				Character: pos.Character,
			})
		}
		ranges, err := d2features.SelectionRanges(doc.URI, doc.Text, positions)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return ranges, nil
	case methodTextDocumentDocumentLink:
		var documentLink documentLinkParams
		if err := json.Unmarshal(params, &documentLink); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(documentLink.TextDocument.URI)
		if !ok {
			return []d2features.DocumentLink{}, nil
		}

		links, err := d2features.DocumentLinks(doc.URI, doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return links, nil
	case methodTextDocumentDocumentColor:
		var documentColor documentColorParams
		if err := json.Unmarshal(params, &documentColor); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		doc, ok := s.document(documentColor.TextDocument.URI)
		if !ok {
			return []d2features.DocumentColor{}, nil
		}

		colors, err := d2features.DocumentColors(doc.URI, doc.Text)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return colors, nil
	case methodTextDocumentColorPresentation:
		var colorPresentation colorPresentationParams
		if err := json.Unmarshal(params, &colorPresentation); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		return d2features.ColorPresentations(colorPresentation.Color, featureRange(colorPresentation.Range)), nil
	case methodTextDocumentCodeAction:
		var codeActionParams codeActionParams
		if err := json.Unmarshal(params, &codeActionParams); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		actions, err := s.codeActions(codeActionParams)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return actions, nil
	case methodWorkspaceSymbol:
		var workspaceSymbol workspaceSymbolParams
		if err := json.Unmarshal(params, &workspaceSymbol); err != nil {
			return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
		}

		symbols, err := s.workspaceSymbols(workspaceSymbol.Query)
		if err != nil {
			return nil, &rpcError{Code: errInternalError, Message: err.Error()}
		}
		return symbols, nil
	case methodShutdown:
		s.mu.Lock()
		s.shutdown = true
		s.mu.Unlock()
		return nil, nil
	default:
		return nil, &rpcError{Code: errMethodNotFound, Message: fmt.Sprintf("method not found: %s", method)}
	}
}

func writeError(writer io.Writer, id json.RawMessage, code int, message string) error {
	return writeJSON(writer, responseMessage{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	})
}

func (s *Server) handleNotification(method string, params json.RawMessage, writer io.Writer) error {
	if method == "" {
		return nil
	}

	if !s.canHandleNotification(method) {
		return nil
	}

	switch method {
	case methodInitialized:
		return s.publishWorkspaceDiagnostics(writer)
	case methodTextDocumentDidOpen:
		var open didOpenTextDocumentParams
		if err := json.Unmarshal(params, &open); err != nil {
			return err
		}
		doc := document{
			URI:     open.TextDocument.URI,
			Version: open.TextDocument.Version,
			Text:    open.TextDocument.Text,
		}
		s.setDocument(doc)
		return s.publishDiagnostics(writer, doc)
	case methodTextDocumentDidChange:
		var change didChangeTextDocumentParams
		if err := json.Unmarshal(params, &change); err != nil {
			return err
		}
		if len(change.ContentChanges) == 0 {
			return nil
		}
		previous, _ := s.document(change.TextDocument.URI)
		text, err := applyContentChanges(previous.Text, change.ContentChanges)
		if err != nil {
			return err
		}
		doc := document{
			URI:     change.TextDocument.URI,
			Version: change.TextDocument.Version,
			Text:    text,
		}
		s.setDocument(doc)
		return s.publishDiagnostics(writer, doc)
	case methodTextDocumentDidSave:
		var save didSaveTextDocumentParams
		if err := json.Unmarshal(params, &save); err != nil {
			return err
		}
		doc, ok := s.document(save.TextDocument.URI)
		if !ok && save.Text == nil {
			return nil
		}
		if save.Text != nil {
			doc = document{
				URI:  save.TextDocument.URI,
				Text: *save.Text,
			}
			s.setDocument(doc)
		}
		return s.publishDiagnostics(writer, doc)
	case methodTextDocumentDidClose:
		var close didCloseTextDocumentParams
		if err := json.Unmarshal(params, &close); err != nil {
			return err
		}
		s.deleteDocument(close.TextDocument.URI)
		params := publishDiagnosticsParams{
			URI:         close.TextDocument.URI,
			Diagnostics: []d2diagnostics.Diagnostic{},
		}
		return writeJSON(writer, notificationMessage{
			JSONRPC: jsonRPCVersion,
			Method:  methodTextDocumentPublishDiagnostic,
			Params:  params,
		})
	case methodWorkspaceDidChangeFolders:
		var change didChangeWorkspaceFoldersParams
		if err := json.Unmarshal(params, &change); err != nil {
			return err
		}
		s.changeWorkspaceFolders(change.Event.Added, change.Event.Removed)
		return nil
	case methodWorkspaceDidChangeWatchedFiles:
		var change didChangeWatchedFilesParams
		if err := json.Unmarshal(params, &change); err != nil {
			return err
		}
		return s.publishWatchedFileDiagnostics(writer, change.Changes)
	default:
		return nil
	}
}

func (s *Server) lifecycleRequestError(method string) *rpcError {
	s.mu.Lock()
	ready := s.ready
	shutdown := s.shutdown
	s.mu.Unlock()

	if !ready && method != methodInitialize {
		return &rpcError{
			Code:    errServerNotInitialized,
			Message: "server has not received initialize request",
		}
	}
	if shutdown {
		return &rpcError{
			Code:    errInvalidRequest,
			Message: "server is shut down",
		}
	}
	if ready && method == methodInitialize {
		return &rpcError{
			Code:    errInvalidRequest,
			Message: "server is already initialized",
		}
	}
	return nil
}

func (s *Server) canHandleNotification(method string) bool {
	if method == methodInitialized {
		s.mu.Lock()
		ready := s.ready
		shutdown := s.shutdown
		s.mu.Unlock()
		return ready && !shutdown
	}

	s.mu.Lock()
	ready := s.ready
	shutdown := s.shutdown
	s.mu.Unlock()
	return ready && !shutdown
}

func (s *Server) setDocument(doc document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[doc.URI] = doc
}

func (s *Server) deleteDocument(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, uri)
}

func (s *Server) document(uri string) (document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.documents[uri]
	return doc, ok
}

func (s *Server) documentFilesystem(active document) (string, map[string]string, map[string]string) {
	activePath := pathFromURI(active.URI)
	fs, uriByPath := s.workspaceFiles()
	fs[activePath] = active.Text
	uriByPath[activePath] = active.URI
	return activePath, fs, uriByPath
}

func (s *Server) changeWorkspaceFolders(added, removed []workspaceFolder) {
	s.mu.Lock()
	defer s.mu.Unlock()

	removedPaths := make(map[string]struct{})
	for _, folder := range removed {
		if folder.URI == "" {
			continue
		}
		removedPaths[filepath.Clean(pathFromURI(folder.URI))] = struct{}{}
	}

	next := make([]string, 0, len(s.rootPaths)+len(added))
	seen := make(map[string]struct{})
	for _, rootPath := range s.rootPaths {
		path := filepath.Clean(rootPath)
		if _, ok := removedPaths[path]; ok {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		next = append(next, path)
	}
	for _, folder := range added {
		if folder.URI == "" {
			continue
		}
		path := filepath.Clean(pathFromURI(folder.URI))
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		next = append(next, path)
	}
	s.rootPaths = next
}

func rootPathsFromInitialize(params initializeParams) []string {
	seen := make(map[string]struct{})
	var roots []string
	for _, folder := range params.WorkspaceFolders {
		if folder.URI == "" {
			continue
		}
		path := filepath.Clean(pathFromURI(folder.URI))
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		roots = append(roots, path)
	}
	if len(roots) == 0 && params.RootURI != nil {
		path := filepath.Clean(pathFromURI(*params.RootURI))
		roots = append(roots, path)
	}
	return roots
}

func workspaceD2Files(rootPath string) (map[string]string, map[string]string) {
	files := make(map[string]string)
	uriByPath := make(map[string]string)
	if rootPath == "" {
		return files, uriByPath
	}

	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		return files, uriByPath
	}

	_ = filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".hg", ".jj", ".direnv", "node_modules":
				if path != rootPath {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".d2" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		cleanPath := filepath.Clean(path)
		files[cleanPath] = string(content)
		uriByPath[cleanPath] = uriFromPath(cleanPath)
		return nil
	})
	return files, uriByPath
}

func workspacesD2Files(rootPaths []string) (map[string]string, map[string]string) {
	files := make(map[string]string)
	uriByPath := make(map[string]string)
	for _, rootPath := range rootPaths {
		rootFiles, rootURIs := workspaceD2Files(rootPath)
		for path, text := range rootFiles {
			files[path] = text
		}
		for path, uri := range rootURIs {
			uriByPath[path] = uri
		}
	}
	return files, uriByPath
}

func (s *Server) workspaceSymbols(query string) ([]workspaceSymbol, error) {
	files, uriByPath := s.workspaceFiles()
	query = strings.ToLower(strings.TrimSpace(query))

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var symbols []workspaceSymbol
	for _, path := range paths {
		docSymbols, err := d2features.Symbols(path, files[path])
		if err != nil {
			return nil, err
		}
		uri := uriByPath[path]
		if uri == "" {
			uri = uriFromPath(path)
		}
		appendWorkspaceSymbols(&symbols, uri, query, docSymbols)
	}
	return symbols, nil
}

func (s *Server) workspaceFiles() (map[string]string, map[string]string) {
	s.mu.Lock()
	rootPaths := append([]string(nil), s.rootPaths...)
	docs := make([]document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	s.mu.Unlock()

	files, uriByPath := workspacesD2Files(rootPaths)
	for _, doc := range docs {
		path := pathFromURI(doc.URI)
		files[path] = doc.Text
		uriByPath[path] = doc.URI
	}
	return files, uriByPath
}

func (s *Server) codeActions(params codeActionParams) ([]codeAction, error) {
	doc, ok := s.document(params.TextDocument.URI)
	if !ok || !codeActionKindAllowed(params.Context.Only, "source.format") {
		return []codeAction{}, nil
	}

	formatted, changed, err := d2features.Format(doc.Text)
	if err != nil || !changed {
		return []codeAction{}, err
	}

	return []codeAction{{
		Title: "Format D2 document",
		Kind:  "source.format",
		Edit: workspaceEdit{
			Changes: map[string][]textEdit{
				doc.URI: {{
					Range: rangePosition{
						Start: position{Line: 0, Character: 0},
						End:   endPosition(doc.Text),
					},
					NewText: formatted,
				}},
			},
		},
	}}, nil
}

func codeActionKindAllowed(only []string, kind string) bool {
	if len(only) == 0 {
		return true
	}
	for _, allowed := range only {
		if allowed == kind || strings.HasPrefix(kind, allowed+".") {
			return true
		}
	}
	return false
}

func appendWorkspaceSymbols(out *[]workspaceSymbol, uri, query string, symbols []d2features.DocumentSymbol) {
	for _, symbol := range symbols {
		if query == "" || strings.Contains(strings.ToLower(symbol.Name), query) {
			*out = append(*out, workspaceSymbol{
				Name: symbol.Name,
				Kind: symbol.Kind,
				Location: location{
					URI:   uri,
					Range: rangeFromFeature(symbol.SelectionRange),
				},
			})
		}
		appendWorkspaceSymbols(out, uri, query, symbol.Children)
	}
}

func (s *Server) DocumentForTest(uri string) (document, bool) {
	return s.document(uri)
}

func (s *Server) publishDiagnostics(writer io.Writer, doc document) error {
	version := doc.Version
	path, files, _ := s.documentFilesystem(doc)
	params := publishDiagnosticsParams{
		URI:         doc.URI,
		Diagnostics: d2diagnostics.ParseInFiles(path, doc.Text, files),
		Version:     &version,
	}
	return writeJSON(writer, notificationMessage{
		JSONRPC: jsonRPCVersion,
		Method:  methodTextDocumentPublishDiagnostic,
		Params:  params,
	})
}

func (s *Server) publishWorkspaceDiagnostics(writer io.Writer) error {
	files, uriByPath := s.workspaceFiles()
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		uri := uriByPath[path]
		if uri == "" {
			uri = uriFromPath(path)
		}
		params := publishDiagnosticsParams{
			URI:         uri,
			Diagnostics: d2diagnostics.ParseInFiles(path, files[path], files),
		}
		if err := writeJSON(writer, notificationMessage{
			JSONRPC: jsonRPCVersion,
			Method:  methodTextDocumentPublishDiagnostic,
			Params:  params,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) publishWatchedFileDiagnostics(writer io.Writer, changes []fileEvent) error {
	for _, change := range changes {
		if change.Type != fileChangeTypeDeleted || filepath.Ext(pathFromURI(change.URI)) != ".d2" {
			continue
		}
		if err := writeJSON(writer, notificationMessage{
			JSONRPC: jsonRPCVersion,
			Method:  methodTextDocumentPublishDiagnostic,
			Params: publishDiagnosticsParams{
				URI:         change.URI,
				Diagnostics: []d2diagnostics.Diagnostic{},
			},
		}); err != nil {
			return err
		}
	}
	return s.publishWorkspaceDiagnostics(writer)
}

func endPosition(text string) position {
	line := 0
	character := 0
	for _, r := range text {
		if r == '\n' {
			line++
			character = 0
			continue
		}
		if r > 0xFFFF {
			character += 2
		} else {
			character++
		}
	}
	return position{Line: line, Character: character}
}

func rangeFromFeature(r d2features.Range) rangePosition {
	return rangePosition{
		Start: position{
			Line:      r.Start.Line,
			Character: r.Start.Character,
		},
		End: position{
			Line:      r.End.Line,
			Character: r.End.Character,
		},
	}
}

func featureRange(r rangePosition) d2features.Range {
	return d2features.Range{
		Start: d2features.Position{
			Line:      r.Start.Line,
			Character: r.Start.Character,
		},
		End: d2features.Position{
			Line:      r.End.Line,
			Character: r.End.Character,
		},
	}
}

func containsLSPPosition(r d2features.Range, pos position) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}
