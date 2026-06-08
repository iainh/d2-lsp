package lsp

import (
	"encoding/json"

	"github.com/iainh/d2-lsp/internal/d2diagnostics"
	"github.com/iainh/d2-lsp/internal/d2features"
)

type requestMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
}

type responseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

func (m responseMessage) MarshalJSON() ([]byte, error) {
	if m.Error != nil {
		return json.Marshal(struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Error   *rpcError       `json:"error"`
		}{
			JSONRPC: m.JSONRPC,
			ID:      m.ID,
			Error:   m.Error,
		})
	}

	return json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  interface{}     `json:"result"`
	}{
		JSONRPC: m.JSONRPC,
		ID:      m.ID,
		Result:  m.Result,
	})
}

type notificationMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProcessID             *int              `json:"processId,omitempty"`
	ClientInfo            *clientInfo       `json:"clientInfo,omitempty"`
	RootURI               *string           `json:"rootUri,omitempty"`
	WorkspaceFolders      []workspaceFolder `json:"workspaceFolders,omitempty"`
	InitializationOptions interface{}       `json:"initializationOptions,omitempty"`
	Capabilities          interface{}       `json:"capabilities,omitempty"`
}

type workspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   serverInfo         `json:"serverInfo"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type serverCapabilities struct {
	PositionEncoding           string                  `json:"positionEncoding,omitempty"`
	TextDocumentSync           textDocumentSyncOptions `json:"textDocumentSync"`
	CompletionProvider         completionOptions       `json:"completionProvider,omitempty"`
	DocumentFormattingProvider bool                    `json:"documentFormattingProvider,omitempty"`
	DocumentSymbolProvider     bool                    `json:"documentSymbolProvider,omitempty"`
	FoldingRangeProvider       bool                    `json:"foldingRangeProvider,omitempty"`
	ReferencesProvider         bool                    `json:"referencesProvider,omitempty"`
	DefinitionProvider         bool                    `json:"definitionProvider,omitempty"`
	DocumentHighlightProvider  bool                    `json:"documentHighlightProvider,omitempty"`
	HoverProvider              bool                    `json:"hoverProvider,omitempty"`
	InlayHintProvider          bool                    `json:"inlayHintProvider,omitempty"`
	SemanticTokensProvider     semanticTokensOptions   `json:"semanticTokensProvider,omitempty"`
	RenameProvider             renameOptions           `json:"renameProvider,omitempty"`
	SelectionRangeProvider     bool                    `json:"selectionRangeProvider,omitempty"`
	DocumentLinkProvider       documentLinkOptions     `json:"documentLinkProvider,omitempty"`
	CodeActionProvider         bool                    `json:"codeActionProvider,omitempty"`
	WorkspaceSymbolProvider    bool                    `json:"workspaceSymbolProvider,omitempty"`
	Workspace                  workspaceOptions        `json:"workspace,omitempty"`
	ColorProvider              bool                    `json:"colorProvider,omitempty"`
}

type workspaceOptions struct {
	WorkspaceFolders workspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

type workspaceFoldersServerCapabilities struct {
	Supported           bool `json:"supported,omitempty"`
	ChangeNotifications bool `json:"changeNotifications,omitempty"`
}

type textDocumentSyncOptions struct {
	OpenClose bool                        `json:"openClose"`
	Change    int                         `json:"change"`
	Save      textDocumentSyncSaveOptions `json:"save,omitempty"`
}

type textDocumentSyncSaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

type completionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type semanticTokensOptions struct {
	Legend semanticTokensLegend `json:"legend"`
	Full   bool                 `json:"full"`
	Range  bool                 `json:"range,omitempty"`
}

type semanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type renameOptions struct {
	PrepareProvider bool `json:"prepareProvider,omitempty"`
}

type documentLinkOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type didOpenTextDocumentParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeTextDocumentParams struct {
	TextDocument   versionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []textDocumentContentChangeEvent `json:"contentChanges"`
}

type didCloseTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type didSaveTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

type didChangeWorkspaceFoldersParams struct {
	Event workspaceFoldersChangeEvent `json:"event"`
}

type workspaceFoldersChangeEvent struct {
	Added   []workspaceFolder `json:"added"`
	Removed []workspaceFolder `json:"removed"`
}

type didChangeWatchedFilesParams struct {
	Changes []fileEvent `json:"changes"`
}

type fileEvent struct {
	URI  string `json:"uri"`
	Type int    `json:"type"`
}

type didChangeConfigurationParams struct {
	Settings interface{} `json:"settings"`
}

type completionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type formattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

type documentFormattingParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Options      formattingOptions      `json:"options"`
}

type documentSymbolParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type foldingRangeParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type referenceParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
	Context      referenceContext       `json:"context"`
}

type referenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type definitionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type documentHighlightParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type hoverParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type inlayHintParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Range        rangePosition          `json:"range"`
}

type semanticTokensParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type prepareRenameParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type renameParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
	NewName      string                 `json:"newName"`
}

type selectionRangeParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Positions    []position             `json:"positions"`
}

type documentLinkParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type documentColorParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type codeActionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Range        rangePosition          `json:"range"`
	Context      codeActionContext      `json:"context"`
}

type codeActionContext struct {
	Diagnostics []d2diagnostics.Diagnostic `json:"diagnostics"`
	Only        []string                   `json:"only,omitempty"`
}

type colorPresentationParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Color        d2features.Color       `json:"color"`
	Range        rangePosition          `json:"range"`
}

type workspaceSymbolParams struct {
	Query string `json:"query"`
}

type textDocumentContentChangeEvent struct {
	Range       *rangePosition `json:"range,omitempty"`
	RangeLength *int           `json:"rangeLength,omitempty"`
	Text        string         `json:"text"`
}

type textEdit struct {
	Range   rangePosition `json:"range"`
	NewText string        `json:"newText"`
}

type rangePosition struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type publishDiagnosticsParams struct {
	URI         string                     `json:"uri"`
	Diagnostics []d2diagnostics.Diagnostic `json:"diagnostics"`
	Version     *int                       `json:"version,omitempty"`
}

type completionList struct {
	IsIncomplete bool                        `json:"isIncomplete"`
	Items        []d2features.CompletionItem `json:"items"`
}

type workspaceEdit struct {
	Changes map[string][]textEdit `json:"changes,omitempty"`
}

type codeAction struct {
	Title string        `json:"title"`
	Kind  string        `json:"kind,omitempty"`
	Edit  workspaceEdit `json:"edit,omitempty"`
}

type workspaceSymbol struct {
	Name     string   `json:"name"`
	Kind     int      `json:"kind"`
	Location location `json:"location"`
}

type location struct {
	URI   string        `json:"uri"`
	Range rangePosition `json:"range"`
}
