package d2features

import "oss.terrastruct.com/d2/d2lsp"

type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

const (
	completionItemKindProperty = 10
	completionItemKindValue    = 12
	completionItemKindKeyword  = 14
)

func Complete(text string, line, character int) ([]CompletionItem, error) {
	items, err := d2lsp.GetCompletionItems(text, line, character)
	if err != nil {
		return nil, err
	}

	completions := make([]CompletionItem, 0, len(items))
	for _, item := range items {
		completions = append(completions, CompletionItem{
			Label:      item.Label,
			Kind:       completionKind(item.Kind),
			Detail:     item.Detail,
			InsertText: item.InsertText,
		})
	}
	return completions, nil
}

func completionKind(kind d2lsp.CompletionKind) int {
	switch kind {
	case d2lsp.KeywordCompletion:
		return completionItemKindKeyword
	case d2lsp.StyleCompletion:
		return completionItemKindProperty
	case d2lsp.ShapeCompletion:
		return completionItemKindValue
	default:
		return completionItemKindKeyword
	}
}
