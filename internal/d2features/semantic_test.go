package d2features

import "testing"

func TestSemanticTokensForReturnsEncodedASTTokens(t *testing.T) {
	tokens, err := SemanticTokensFor("file:///diagram.d2", "# hi\nserver: {shape: rectangle}\nserver -> database\n")
	if err != nil {
		t.Fatalf("semantic tokens: %v", err)
	}

	decoded := decodeSemanticTokens(tokens.Data)
	assertSemanticToken(t, decoded, semanticToken{line: 0, start: 0, length: len("# hi"), tokenType: semanticTokenComment})
	assertSemanticToken(t, decoded, semanticToken{line: 1, start: 0, length: len("server"), tokenType: semanticTokenProperty})
	assertSemanticToken(t, decoded, semanticToken{line: 1, start: len("server: {"), length: len("shape"), tokenType: semanticTokenKeyword})
	assertSemanticToken(t, decoded, semanticToken{line: 1, start: len("server: {shape: "), length: len("rectangle"), tokenType: semanticTokenString})
	assertSemanticToken(t, decoded, semanticToken{line: 2, start: 0, length: len("server"), tokenType: semanticTokenVariable})
	assertSemanticToken(t, decoded, semanticToken{line: 2, start: len("server -> "), length: len("database"), tokenType: semanticTokenVariable})
}

func TestSemanticTokensForUsesUTF16Lengths(t *testing.T) {
	tokens, err := SemanticTokensFor("file:///diagram.d2", "🙂: 1\n")
	if err != nil {
		t.Fatalf("semantic tokens: %v", err)
	}

	decoded := decodeSemanticTokens(tokens.Data)
	assertSemanticToken(t, decoded, semanticToken{line: 0, start: 0, length: 2, tokenType: semanticTokenProperty})
	assertSemanticToken(t, decoded, semanticToken{line: 0, start: 4, length: 1, tokenType: semanticTokenNumber})
}

func TestSemanticTokensForReturnsEmptyForInvalidDocument(t *testing.T) {
	tokens, err := SemanticTokensFor("file:///diagram.d2", "x: {\n")
	if err != nil {
		t.Fatalf("semantic tokens: %v", err)
	}
	if len(tokens.Data)%5 != 0 {
		t.Fatalf("expected semantic token data groups of five, got %#v", tokens.Data)
	}
}

func decodeSemanticTokens(data []uint32) []semanticToken {
	var tokens []semanticToken
	line := 0
	start := 0
	for i := 0; i+4 < len(data); i += 5 {
		deltaLine := int(data[i])
		deltaStart := int(data[i+1])
		if deltaLine == 0 {
			start += deltaStart
		} else {
			line += deltaLine
			start = deltaStart
		}
		tokens = append(tokens, semanticToken{
			line:      line,
			start:     start,
			length:    int(data[i+2]),
			tokenType: int(data[i+3]),
		})
	}
	return tokens
}

func assertSemanticToken(t *testing.T, tokens []semanticToken, want semanticToken) {
	t.Helper()
	for _, got := range tokens {
		if got == want {
			return
		}
	}
	t.Fatalf("missing token %#v in %#v", want, tokens)
}
