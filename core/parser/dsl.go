package parser

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/hyperterse/hyperterse/core/types"
)

// Parser holds the state of the parsing process
type Parser struct {
	input string
	pos   int
}

// NewParser creates a new Parser instance
func NewParser(input string) *Parser {
	return &Parser{input: input, pos: 0}
}

// Parse parses the input string into a Protobuf Model
func (p *Parser) Parse() (*pb.Model, error) {
	model := &pb.Model{}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		if strings.HasPrefix(p.input[p.pos:], "adapter") {
			adapter, err := p.parseAdapter()
			if err != nil {
				return nil, err
			}
			model.Adapters = append(model.Adapters, adapter)
		} else if strings.HasPrefix(p.input[p.pos:], "query") {
			query, err := p.parseQuery()
			if err != nil {
				return nil, err
			}
			model.Queries = append(model.Queries, query)
		} else {
			return nil, fmt.Errorf("unexpected token at position %d: %s", p.pos, p.peek(10))
		}
	}

	return model, nil
}

func (p *Parser) parseAdapter() (*pb.Adapter, error) {
	if !p.consume("adapter") {
		return nil, fmt.Errorf("expected 'adapter'")
	}
	p.skipWhitespace()
	name, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if !p.consume("{") {
		return nil, fmt.Errorf("expected '{' after adapter name")
	}

	adapter := &pb.Adapter{Name: name}

	for {
		p.skipWhitespace()
		if p.consume("}") {
			break
		}

		key, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':' after key '%s'", key)
		}
		p.skipWhitespace()

		switch key {
		case "connector":
			val, err := p.parseIdentifier() // or string?
			if err != nil {
				// Try parsing as string literal if identifier fails?
				// Grammar says `connector=Adapters` where Adapters is 'postgres' etc.
				// Let's assume identifier or keyword.
				return nil, err
			}
			connectorEnum, err := types.StringToConnectorEnum(val)
			if err != nil {
				return nil, fmt.Errorf("invalid connector '%s': %w", val, err)
			}
			adapter.Connector = connectorEnum
		case "connection_string":
			val, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			adapter.ConnectionString = val
		case "options":
			opts, err := p.parseAdapterOptions()
			if err != nil {
				return nil, err
			}
			adapter.Options = opts
		default:
			return nil, fmt.Errorf("unknown adapter field: %s", key)
		}
	}
	return adapter, nil
}

func (p *Parser) parseAdapterOptions() (*pb.AdapterOptions, error) {
	if !p.consume("{") {
		return nil, fmt.Errorf("expected '{' for options")
	}
	opts := &pb.AdapterOptions{
		Options: make(map[string]string),
	}
	for {
		p.skipWhitespace()
		if p.consume("}") {
			break
		}
		key, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':'")
		}
		p.skipWhitespace()

		// Parse value - can be string literal or other types
		// For now, we'll parse as string and let connectors handle conversion
		if p.input[p.pos] == '"' {
			val, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			opts.Options[key] = val
		} else {
			// Parse until whitespace or }
			start := p.pos
			for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && p.input[p.pos] != '}' {
				p.pos++
			}
			opts.Options[key] = p.input[start:p.pos]
		}
	}
	return opts, nil
}

func (p *Parser) parseQuery() (*pb.Query, error) {
	if !p.consume("query") {
		return nil, fmt.Errorf("expected 'query'")
	}
	p.skipWhitespace()
	name, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if !p.consume("{") {
		return nil, fmt.Errorf("expected '{'")
	}

	query := &pb.Query{Name: name}

	for {
		p.skipWhitespace()
		if p.consume("}") {
			break
		}

		key, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':'")
		}
		p.skipWhitespace()

		switch key {
		case "use":
			val, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			query.Use = append(query.Use, val)
		case "statement":
			val, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			query.Statement = val
		case "description":
			val, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			query.Description = val
		case "inputs":
			inputs, err := p.parseInputs()
			if err != nil {
				return nil, err
			}
			query.Inputs = inputs
		case "data":
			data, err := p.parseData()
			if err != nil {
				return nil, err
			}
			query.Data = data
		default:
			return nil, fmt.Errorf("unknown query field: %s", key)
		}
	}
	return query, nil
}

func (p *Parser) parseInputs() ([]*pb.Input, error) {
	if !p.consume("{") {
		return nil, fmt.Errorf("expected '{' for inputs")
	}
	var inputs []*pb.Input
	for {
		p.skipWhitespace()
		if p.consume("}") {
			break
		}

		// Input name logic: Name followed by optional '?'
		name, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		optional := false
		if p.consume("?") {
			optional = true
		}

		p.skipWhitespace()
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':' after input name")
		}
		p.skipWhitespace()
		if !p.consume("{") {
			return nil, fmt.Errorf("expected '{' for input body")
		}

		input := &pb.Input{
			Name:     name,
			Optional: optional,
		}

		// Parse input body
		for {
			p.skipWhitespace()
			if p.consume("}") {
				break
			}
			k, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			p.skipWhitespace()
			p.consume(":")
			p.skipWhitespace()

			if k == "type" {
				val, err := p.parseIdentifier() // types are identifiers like string, int
				if err != nil {
					return nil, err
				}
				input.Type = val
			} else if k == "description" {
				val, err := p.parseStringLiteral()
				if err != nil {
					return nil, err
				}
				input.Description = val
			} else if k == "default_value" {
				// Could be int or string or bool
				// Simplified: parse until newline or } or try to identify
				// For now let's support integer literals and string literals
				// The grammar supports: INTEGER | FLOAT | DATETIME | BOOLEAN | STRING
				// We'll peek.
				if p.input[p.pos] == '"' {
					val, _ := p.parseStringLiteral()
					input.DefaultValue = val
				} else {
					// Read until newline or '}'
					// Quick hack: read word
					start := p.pos
					for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && p.input[p.pos] != '}' {
						p.pos++
					}
					input.DefaultValue = p.input[start:p.pos]
				}
			}
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func (p *Parser) parseData() ([]*pb.Data, error) {
	if !p.consume("{") {
		return nil, fmt.Errorf("expected '{' for data")
	}
	var dataList []*pb.Data
	for {
		p.skipWhitespace()
		if p.consume("}") {
			break
		}
		name, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		optional := false
		if p.consume("?") {
			optional = true
		}
		p.skipWhitespace()
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':'")
		}
		p.skipWhitespace()
		if !p.consume("{") {
			return nil, fmt.Errorf("expected '{'")
		}

		d := &pb.Data{
			Name:     name,
			Optional: optional,
		}

		for {
			p.skipWhitespace()
			if p.consume("}") {
				break
			}
			k, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			p.skipWhitespace()
			p.consume(":")
			p.skipWhitespace()

			if k == "type" {
				val, err := p.parseIdentifier()
				if err != nil {
					return nil, err
				}
				d.Type = val
			} else if k == "description" {
				val, err := p.parseStringLiteral()
				if err != nil {
					return nil, err
				}
				d.Description = val
			} else if k == "map_to" {
				val, err := p.parseStringLiteral()
				if err != nil {
					return nil, err
				}
				d.MapTo = val
			}
		}
		dataList = append(dataList, d)
	}
	return dataList, nil
}

// Helper methods

func (p *Parser) consume(token string) bool {
	if strings.HasPrefix(p.input[p.pos:], token) {
		p.pos += len(token)
		return true
	}
	return false
}

func (p *Parser) parseIdentifier() (string, error) {
	p.skipWhitespace()
	start := p.pos
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("unexpected EOF")
	}
	// Identifiers start with letter or _
	r := rune(p.input[p.pos])
	if !unicode.IsLetter(r) && r != '_' {
		return "", fmt.Errorf("invalid identifier start")
	}
	p.pos++
	for p.pos < len(p.input) {
		r = rune(p.input[p.pos])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos], nil
}

func (p *Parser) parseStringLiteral() (string, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) || p.input[p.pos] != '"' {
		return "", fmt.Errorf("expected string literal")
	}
	p.pos++ // consume opening quote
	var sb strings.Builder
	for p.pos < len(p.input) {
		r := p.input[p.pos]
		if r == '"' {
			p.pos++ // consume closing quote
			return sb.String(), nil
		}
		if r == '\\' {
			p.pos++
			if p.pos >= len(p.input) {
				return "", fmt.Errorf("unexpected EOF in string")
			}
			// Simplified escape handling
			sb.WriteByte(p.input[p.pos])
			p.pos++
		} else {
			sb.WriteByte(r)
			p.pos++
		}
	}
	return "", fmt.Errorf("unclosed string literal")
}

func (p *Parser) skipWhitespace() {
	// Loop to handle whitespace and comments
	for {
		startPos := p.pos
		// standard whitespace
		for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
			p.pos++
		}

		// check for single line comment //
		if p.pos+2 <= len(p.input) && p.input[p.pos:p.pos+2] == "//" {
			// consume until newline
			p.pos += 2
			for p.pos < len(p.input) && p.input[p.pos] != '\n' {
				p.pos++
			}
		}

		// check for multi line comment /* */
		if p.pos+2 <= len(p.input) && p.input[p.pos:p.pos+2] == "/*" {
			p.pos += 2
			for p.pos+2 <= len(p.input) {
				if p.input[p.pos:p.pos+2] == "*/" {
					p.pos += 2
					break
				}
				p.pos++
			}
		}

		// If no advancement was made in this iteration, we are done skipping
		if p.pos == startPos {
			break
		}
	}
}

func (p *Parser) peek(n int) string {
	if p.pos+n > len(p.input) {
		return p.input[p.pos:]
	}
	return p.input[p.pos : p.pos+n]
}

