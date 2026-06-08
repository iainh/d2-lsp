package d2features

import (
	"errors"
	"sort"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
)

type SemanticTokens struct {
	Data []uint32 `json:"data"`
}

var SemanticTokenTypes = []string{
	"keyword",
	"property",
	"string",
	"number",
	"comment",
	"variable",
	"operator",
}

const (
	semanticTokenKeyword = iota
	semanticTokenProperty
	semanticTokenString
	semanticTokenNumber
	semanticTokenComment
	semanticTokenVariable
	semanticTokenOperator
)

type semanticToken struct {
	line      int
	start     int
	length    int
	tokenType int
}

func SemanticTokensFor(path, text string) (SemanticTokens, error) {
	ast, err := d2parser.Parse(path, strings.NewReader(text), &d2parser.ParseOptions{
		UTF16Pos: true,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if !errors.As(err, &pe) {
			return SemanticTokens{}, err
		}
	}
	if ast == nil {
		return SemanticTokens{}, nil
	}

	tokens := make([]semanticToken, 0, len(ast.Nodes)*2)
	collectMapSemanticTokens(ast, &tokens)
	sortSemanticTokens(tokens)
	return SemanticTokens{Data: encodeSemanticTokens(tokens)}, nil
}

func collectMapSemanticTokens(m *d2ast.Map, tokens *[]semanticToken) {
	for _, node := range m.Nodes {
		switch {
		case node.Comment != nil:
			addSemanticToken(tokens, node.Comment.Range, semanticTokenComment)
		case node.BlockComment != nil:
			addSemanticToken(tokens, node.BlockComment.Range, semanticTokenComment)
		case node.Import != nil:
			collectImportSemanticTokens(node.Import, tokens)
		case node.Substitution != nil:
			collectSubstitutionSemanticTokens(node.Substitution, tokens)
		case node.MapKey != nil:
			collectKeySemanticTokens(node.MapKey, tokens)
		}
	}
}

func collectKeySemanticTokens(mk *d2ast.Key, tokens *[]semanticToken) {
	if mk.Key != nil {
		collectKeyPathSemanticTokens(mk.Key, semanticTokenProperty, tokens)
	}
	for _, edge := range mk.Edges {
		collectEdgeSemanticTokens(edge, tokens)
	}
	if mk.EdgeIndex != nil {
		if mk.EdgeIndex.Int != nil {
			addSemanticToken(tokens, mk.EdgeIndex.Range, semanticTokenNumber)
		} else if mk.EdgeIndex.Glob {
			addSemanticToken(tokens, mk.EdgeIndex.Range, semanticTokenOperator)
		}
	}
	if mk.EdgeKey != nil {
		collectKeyPathSemanticTokens(mk.EdgeKey, semanticTokenProperty, tokens)
	}
	collectValueSemanticTokens(mk.Value.Unbox(), tokens)
}

func collectEdgeSemanticTokens(edge *d2ast.Edge, tokens *[]semanticToken) {
	if edge == nil {
		return
	}
	if edge.Src != nil {
		collectKeyPathSemanticTokens(edge.Src, semanticTokenVariable, tokens)
	}
	if edge.Dst != nil {
		collectKeyPathSemanticTokens(edge.Dst, semanticTokenVariable, tokens)
	}
}

func collectValueSemanticTokens(value d2ast.Value, tokens *[]semanticToken) {
	switch v := value.(type) {
	case nil:
		return
	case *d2ast.Null, *d2ast.Boolean, *d2ast.Suspension:
		addSemanticToken(tokens, value.GetRange(), semanticTokenKeyword)
	case *d2ast.Number:
		addSemanticToken(tokens, v.Range, semanticTokenNumber)
	case d2ast.String:
		addSemanticToken(tokens, v.GetRange(), semanticTokenString)
	case *d2ast.Import:
		collectImportSemanticTokens(v, tokens)
	case *d2ast.Array:
		for _, node := range v.Nodes {
			collectArrayNodeSemanticTokens(node, tokens)
		}
	case *d2ast.Map:
		collectMapSemanticTokens(v, tokens)
	}
}

func collectArrayNodeSemanticTokens(node d2ast.ArrayNodeBox, tokens *[]semanticToken) {
	switch {
	case node.Comment != nil:
		addSemanticToken(tokens, node.Comment.Range, semanticTokenComment)
	case node.BlockComment != nil:
		addSemanticToken(tokens, node.BlockComment.Range, semanticTokenComment)
	case node.Substitution != nil:
		collectSubstitutionSemanticTokens(node.Substitution, tokens)
	case node.Import != nil:
		collectImportSemanticTokens(node.Import, tokens)
	default:
		if value, ok := node.Unbox().(d2ast.Value); ok {
			collectValueSemanticTokens(value, tokens)
		}
	}
}

func collectImportSemanticTokens(imp *d2ast.Import, tokens *[]semanticToken) {
	if imp == nil {
		return
	}
	for _, part := range imp.Path {
		if part.Unbox() != nil {
			addSemanticToken(tokens, part.Unbox().GetRange(), semanticTokenVariable)
		}
	}
}

func collectSubstitutionSemanticTokens(substitution *d2ast.Substitution, tokens *[]semanticToken) {
	if substitution == nil {
		return
	}
	addSemanticToken(tokens, substitution.Range, semanticTokenVariable)
}

func collectKeyPathSemanticTokens(path *d2ast.KeyPath, fallbackType int, tokens *[]semanticToken) {
	for _, part := range path.Path {
		scalar := part.Unbox()
		if scalar == nil {
			continue
		}
		tokenType := fallbackType
		if isKeyword(scalar.ScalarString()) {
			tokenType = semanticTokenKeyword
		}
		addSemanticToken(tokens, scalar.GetRange(), tokenType)
	}
}

func isKeyword(value string) bool {
	lower := strings.ToLower(value)
	if _, ok := d2ast.ReservedKeywords[lower]; ok {
		return true
	}
	if _, ok := d2ast.StyleKeywords[lower]; ok {
		return true
	}
	if _, ok := d2ast.BoardKeywords[lower]; ok {
		return true
	}
	return false
}

func addSemanticToken(tokens *[]semanticToken, r d2ast.Range, tokenType int) {
	token := tokenFromRange(r, tokenType)
	if token.length <= 0 {
		return
	}
	*tokens = append(*tokens, token)
}

func tokenFromRange(r d2ast.Range, tokenType int) semanticToken {
	if r.Start.Line != r.End.Line {
		return semanticToken{}
	}
	start := nonnegative(r.Start.Column)
	end := nonnegative(r.End.Column)
	return semanticToken{
		line:      nonnegative(r.Start.Line),
		start:     start,
		length:    end - start,
		tokenType: tokenType,
	}
}

func sortSemanticTokens(tokens []semanticToken) {
	sort.SliceStable(tokens, func(i, j int) bool {
		if tokens[i].line != tokens[j].line {
			return tokens[i].line < tokens[j].line
		}
		if tokens[i].start != tokens[j].start {
			return tokens[i].start < tokens[j].start
		}
		return tokens[i].length < tokens[j].length
	})
}

func encodeSemanticTokens(tokens []semanticToken) []uint32 {
	data := make([]uint32, 0, len(tokens)*5)
	prevLine := 0
	prevStart := 0
	for i, token := range tokens {
		deltaLine := token.line
		deltaStart := token.start
		if i > 0 {
			deltaLine = token.line - prevLine
			if deltaLine == 0 {
				deltaStart = token.start - prevStart
			}
		}
		data = append(data,
			uint32(deltaLine),
			uint32(deltaStart),
			uint32(token.length),
			uint32(token.tokenType),
			0,
		)
		prevLine = token.line
		prevStart = token.start
	}
	return data
}
