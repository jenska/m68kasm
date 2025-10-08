package asm

import (
	"fmt"
)

func (p *Parser) parseExpr() (int64, error) { return p.parseExprUntil(COMMA, NEWLINE, EOF) }

func (p *Parser) parseExprUntil(stops ...Kind) (int64, error) {
	stop := map[Kind]bool{}
	for _, k := range stops {
		stop[k] = true
	}

	out := []int64{}
	ops := []Kind{}
	wantValue := true

	precedence := func(k Kind) int {
		switch k {
		case TILDE:
			return 7
		case STAR, SLASH:
			return 6
		case PLUS, MINUS:
			return 5
		case LSHIFT, RSHIFT:
			return 4
		case AMP:
			return 3
		case CARET:
			return 2
		case PIPE:
			return 1
		default:
			return 0
		}
	}
	isUnary := func(k Kind) bool { return k == MINUS || k == PLUS || k == TILDE }

	apply := func(op Kind) error {
		if op == TILDE {
			if len(out) < 1 {
				return fmt.Errorf("unärer Operator erwartet ein Argument")
			}
			a := out[len(out)-1]
			out = out[:len(out)-1]
			out = append(out, ^a)
			return nil
		}
		if len(out) < 2 {
			return fmt.Errorf("binärer Operator erwartet zwei Argumente")
		}
		b := out[len(out)-1]
		a := out[len(out)-2]
		out = out[:len(out)-2]
		switch op {
		case PLUS:
			out = append(out, a+b)
		case MINUS:
			out = append(out, a-b)
		case STAR:
			out = append(out, a*b)
		case SLASH:
			if b == 0 {
				return fmt.Errorf("division by zero")
			}
			out = append(out, a/b)
		case LSHIFT:
			out = append(out, a<<uint64(b))
		case RSHIFT:
			out = append(out, a>>uint64(b))
		case AMP:
			out = append(out, a&b)
		case CARET:
			out = append(out, a^b)
		case PIPE:
			out = append(out, a|b)
		default:
			return fmt.Errorf("unbekannter Operator")
		}
		return nil
	}

	pushOp := func(k Kind) error {
		if wantValue && (k == PLUS || k == MINUS) {
			out = append(out, 0)
		}
		for len(ops) > 0 {
			top := ops[len(ops)-1]
			if top == LPAREN {
				break
			}
			if precedence(top) >= precedence(k) {
				ops = ops[:len(ops)-1]
				if err := apply(top); err != nil {
					return err
				}
			} else {
				break
			}
		}
		ops = append(ops, k)
		return nil
	}

loop:
	for {
		t := p.peek()
		if stop[t.Kind] {
			break
		}
		switch t.Kind {
		case NUMBER:
			p.next()
			out = append(out, t.Val)
			wantValue = false
		case IDENT:
			p.next()
			if v, ok := p.labels[t.Text]; ok {
				out = append(out, int64(v))
				wantValue = false
			} else {
				return 0, fmt.Errorf("undefiniertes Label in Ausdruck: %s", t.Text)
			}
		case LPAREN:
			p.next()
			ops = append(ops, LPAREN)
			wantValue = true
		case RPAREN:
			if stop[RPAREN] && (len(ops) == 0 || ops[len(ops)-1] != LPAREN) {
				break loop
			}
			p.next()
			for len(ops) > 0 && ops[len(ops)-1] != LPAREN {
				op := ops[len(ops)-1]
				ops = ops[:len(ops)-1]
				if err := apply(op); err != nil {
					return 0, err
				}
			}
			if len(ops) == 0 {
				return 0, fmt.Errorf("expected: ')'")
			}
			ops = ops[:len(ops)-1]
			wantValue = false
		case PLUS, MINUS, STAR, SLASH, LSHIFT, RSHIFT, AMP, PIPE, CARET, TILDE:
			p.next()
			if isUnary(t.Kind) && wantValue && t.Kind != TILDE {
				if err := pushOp(t.Kind); err != nil {
					return 0, err
				}
			} else if t.Kind == TILDE {
				ops = append(ops, TILDE)
			} else {
				if err := pushOp(t.Kind); err != nil {
					return 0, err
				}
				wantValue = true
			}
		default:
			break loop
		}
	}
	for len(ops) > 0 {
		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		if op == LPAREN {
			return 0, fmt.Errorf("expected: '('")
		}
		if err := apply(op); err != nil {
			return 0, err
		}
	}
	if len(out) != 1 {
		return 0, fmt.Errorf("ungültiger Ausdruck")
	}
	return out[0], nil
}
