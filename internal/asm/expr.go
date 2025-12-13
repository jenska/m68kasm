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
		case TILDE, BANG:
			return 8
		case STAR, SLASH, PERCENT:
			return 7
		case PLUS, MINUS:
			return 6
		case LSHIFT, RSHIFT:
			return 5
		case LT, GT, LTE, GTE:
			return 4
		case EQEQ, NEQ:
			return 3
		case AMP:
			return 2
		case CARET:
			return 1
		case PIPE:
			return 0
		case ANDAND:
			return -1
		case OROR:
			return -2
		default:
			return -3
		}
	}
	isUnary := func(k Kind) bool { return k == MINUS || k == PLUS || k == TILDE || k == BANG }

	apply := func(op Kind) error {
		if op == TILDE || op == BANG {
			if len(out) < 1 {
				return fmt.Errorf("unärer Operator erwartet ein Argument")
			}
			a := out[len(out)-1]
			out = out[:len(out)-1]
			if op == TILDE {
				out = append(out, ^a)
			} else {
				if a == 0 {
					out = append(out, 1)
				} else {
					out = append(out, int64(0))
				}
			}
			return nil
		}
		if len(out) < 2 {
			return fmt.Errorf("binärer Operator erwartet zwei Argumente")
		}
		b := out[len(out)-1]
		a := out[len(out)-2]
		out = out[:len(out)-2]
		truth := func(v int64) int64 {
			if v == 0 {
				return 0
			}
			return 1
		}
		boolVal := func(v bool) int64 {
			if v {
				return 1
			}
			return 0
		}
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
		case PERCENT:
			if b == 0 {
				return fmt.Errorf("division by zero")
			}
			out = append(out, a%b)
		case LSHIFT:
			out = append(out, a<<uint64(b))
		case RSHIFT:
			out = append(out, a>>uint64(b))
		case LT:
			out = append(out, boolVal(a < b))
		case GT:
			out = append(out, boolVal(a > b))
		case LTE:
			out = append(out, boolVal(a <= b))
		case GTE:
			out = append(out, boolVal(a >= b))
		case EQEQ:
			out = append(out, boolVal(a == b))
		case NEQ:
			out = append(out, boolVal(a != b))
		case AMP:
			out = append(out, a&b)
		case CARET:
			out = append(out, a^b)
		case PIPE:
			out = append(out, a|b)
		case ANDAND:
			out = append(out, truth(truth(a)&truth(b)))
		case OROR:
			out = append(out, truth(truth(a)|truth(b)))
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
			if name, ok, err := p.consumeLocalLabelRef(); ok {
				if err != nil {
					return 0, err
				}
				if v, ok := p.labels[name]; ok {
					out = append(out, int64(v))
					wantValue = false
					continue
				}
				return 0, fmt.Errorf("undefiniertes Label in Ausdruck: %s", name)
			}
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
		case PLUS, MINUS, STAR, SLASH, PERCENT, LSHIFT, RSHIFT, AMP, PIPE, CARET, TILDE, BANG, LT, GT, LTE, GTE, EQEQ, NEQ, ANDAND, OROR:
			p.next()
			if isUnary(t.Kind) && wantValue && t.Kind != TILDE && t.Kind != BANG {
				if err := pushOp(t.Kind); err != nil {
					return 0, err
				}
			} else if t.Kind == TILDE || t.Kind == BANG {
				ops = append(ops, t.Kind)
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
		return 0, fmt.Errorf("invalid expression")
	}
	return out[0], nil
}
