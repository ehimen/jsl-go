package parse

import (
	"fmt"

	"strconv"

	"errors"

	"github.com/ehimen/jaslang/dfa"
	"github.com/ehimen/jaslang/lex"
)

type Parser interface {
	Parse() (RootNode, error)
}

type parser struct {
	lexer          lex.Lexer
	dfa            dfa.Machine
	current        lex.Lexeme
	nodeStack      []Node
	statementStack []Statement
}

type UnexpectedTokenError struct {
	l lex.Lexeme
}

func (err UnexpectedTokenError) Error() string {
	return fmt.Sprintf("Unexpected token \"%s\" at position %d", err.l.Value, err.l.Start)
}

type InvalidNumberError struct {
	UnexpectedTokenError
}

func (err InvalidNumberError) Error() string {
	return fmt.Sprintf("Invalid number token \"%s\" at position %d", err.l.Value, err.l.Start)
}

var UnterminatedStatement = errors.New("Unterminated statement!")

func (p *parser) Parse() (RootNode, error) {
	root := &RootNode{}

	p.nodeStack = []Node{root}

	for {
		lexeme, err := p.lexer.GetNext()

		if err != nil {
			break
		}

		// We don't care about whitespace
		if lexeme.Type == lex.LWhitespace {
			continue
		}

		p.current = lexeme

		if err := p.dfa.Transition(string(lexeme.Type)); err != nil {
			return *root, err
		}
	}

	if err := p.dfa.Finish(); err != nil {
		return *root, UnterminatedStatement
	}

	return *root, nil
}

func NewParser(lexer lex.Lexer) Parser {
	parser := parser{lexer: lexer}

	builder := dfa.NewMachineBuilder()

	start := "start"
	identifier := string(lex.LIdentifier)
	parenOpen := string(lex.LParenOpen)
	parenClose := string(lex.LParenClose)
	quoted := string(lex.LQuoted)
	term := string(lex.LSemiColon)
	number := string(lex.LNumber)
	true := string(lex.LBoolTrue)
	false := string(lex.LBoolFalse)

	literals := []string{number, quoted, true, false}

	builder.Paths([]string{start}, append(literals, identifier))
	builder.Path(identifier, parenOpen)
	builder.Path(parenOpen, quoted)
	builder.Path(quoted, parenClose)
	builder.Path(parenClose, term)
	builder.Paths(literals, []string{term})
	builder.Paths([]string{term}, literals)

	builder.WhenEntering(identifier, parser.createIdentifier)
	builder.WhenEntering(quoted, parser.createStringLiteral)
	builder.WhenEntering(parenClose, parser.closeNode)
	builder.WhenEntering(term, parser.closeNode)
	builder.WhenEntering(number, parser.createNumberLiteral)
	builder.WhenEntering(true, parser.createBooleanLiteral)
	builder.WhenEntering(false, parser.createBooleanLiteral)

	builder.Accept(term)

	machine, err := builder.Start(start)

	if err != nil {
		panic(fmt.Sprintf("Cannot build parse machine: %v", err))
	}

	parser.dfa = machine

	return &parser
}

func (p *parser) createIdentifier() error {
	p.push(NewFunctionCall(p.current.Value))

	return nil
}

func (p *parser) createStringLiteral() error {
	p.push(NewString(p.current.Value))

	return nil
}

func (p *parser) createBooleanLiteral() error {
	p.push(NewBoolean(p.current.Type == lex.LBoolTrue))

	return nil
}

func (p *parser) createNumberLiteral() error {
	if number, err := strconv.ParseFloat(p.current.Value, 64); err == nil {
		p.push(NewNumber(number))
	} else {
		return InvalidNumberError{UnexpectedTokenError{p.current}}
	}

	return nil
}

func (p *parser) closeNode() error {
	p.nodeStack = p.nodeStack[0 : len(p.nodeStack)-1]

	return nil
}

func (p *parser) push(node Node) {
	context := getContext(p)

	// Insert a statement if we need to.
	if root, isRoot := context.(*RootNode); isRoot {
		statement := &Statement{}
		root.PushStatement(statement)
		p.nodeStack = append(p.nodeStack, statement)
	}

	context = getContext(p)

	if parent, isParent := context.(ContainsChildren); isParent {
		parent.Push(node)
	}

	if parent, isParent := node.(ContainsChildren); isParent {
		p.nodeStack = append(p.nodeStack, parent)
	}
}

func getContext(p *parser) Node {
	return p.nodeStack[len(p.nodeStack)-1]
}
