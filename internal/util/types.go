package util

type TokenType int

const (
    TokenUnknown TokenType = iota
    TokenLabel
    TokenMnemonic
    TokenOperand
)

func IdentifyToken(s string) TokenType {
    if len(s) > 0 && s[len(s)-1] == ':' {
        return TokenLabel
    }
    // Very naive: if all uppercase, treat as mnemonic
    upper := true
    for _, r := range s {
        if r >= 'a' && r <= 'z' {
            upper = false
            break
        }
    }
    if upper {
        return TokenMnemonic
    }
    return TokenOperand
}
}