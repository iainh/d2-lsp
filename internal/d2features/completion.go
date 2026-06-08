package d2features

import (
	"strings"

	"oss.terrastruct.com/d2/d2lsp"
	"oss.terrastruct.com/d2/d2target"
)

type CompletionItem struct {
	Label         string         `json:"label"`
	Kind          int            `json:"kind,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	Documentation *MarkupContent `json:"documentation,omitempty"`
	InsertText    string         `json:"insertText,omitempty"`
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
		completion := CompletionItem{
			Label:      item.Label,
			Kind:       completionKind(item.Kind),
			Detail:     item.Detail,
			InsertText: item.InsertText,
		}
		enrichCompletionItem(&completion)
		completions = append(completions, completion)
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

func enrichCompletionItem(item *CompletionItem) {
	label := strings.ToLower(item.Label)
	switch {
	case styleHoverDescriptions[label] != "":
		if item.Detail == "" {
			item.Detail = "D2 style key"
		}
		item.Documentation = completionDocumentation(styleHoverDescriptions[label])
	case keywordHoverDescriptions[label] != "":
		if item.Detail == "" {
			item.Detail = "D2 keyword"
		}
		item.Documentation = completionDocumentation(keywordHoverDescriptions[label])
	case d2target.IsShape(label):
		if item.Detail == "" {
			item.Detail = "D2 shape"
		}
		item.Documentation = completionDocumentation("`" + label + "` is a D2 shape.")
	}
}

func completionDocumentation(value string) *MarkupContent {
	return &MarkupContent{
		Kind:  markupKindMarkdown,
		Value: value,
	}
}
