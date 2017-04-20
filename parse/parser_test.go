package parse_test

import (
	"testing"

	"github.com/ehimen/jaslang/lex"
	"github.com/ehimen/jaslang/parse"
	"github.com/ehimen/jaslang/testutil"

	"github.com/stretchr/testify/assert"
)

func TestSimpleFunctionCall(t *testing.T) {
	parser := getParser([]lex.Lexeme{
		testutil.MakeLexeme("print", lex.LIdentifier, 1),
		testutil.MakeLexeme("(", lex.LParenOpen, 2),
		testutil.MakeLexeme("Hello, world!", lex.LQuoted, 3),
		testutil.MakeLexeme(")", lex.LParenClose, 4),
		testutil.MakeLexeme(";", lex.LSemiColon, 5),
	})

	expected := expectStatements(
		&parse.Statement{
			ParentNode: parse.ParentNode{
				Children: []parse.Node{
					&parse.FunctionCall{
						Identifier: "print",
						ParentNode: parse.ParentNode{
							Children: []parse.Node{
								&parse.StringLiteral{
									Value: "Hello, world!",
								},
							},
						},
					},
				},
			},
		},
	)

	assert.Equal(t, expected, testParse(parser, t))
}

func TestTwoLiterals(t *testing.T) {
	parser := getParser([]lex.Lexeme{
		testutil.MakeLexeme("1.34", lex.LNumber, 1),
		testutil.MakeLexeme(";", lex.LSemiColon, 2),
		testutil.MakeLexeme(" ", lex.LWhitespace, 3),
		testutil.MakeLexeme("3.42", lex.LNumber, 4),
		testutil.MakeLexeme(";", lex.LSemiColon, 5),
	})

	expected := expectStatements(
		&parse.Statement{
			ParentNode: parse.ParentNode{
				Children: []parse.Node{
					&parse.NumberLiteral{
						Value: float64(1.34),
					},
				},
			},
		},
		&parse.Statement{
			ParentNode: parse.ParentNode{
				Children: []parse.Node{
					&parse.NumberLiteral{
						Value: float64(3.42),
					},
				},
			},
		},
	)

	assert.Equal(t, expected, testParse(parser, t))
}

func TestInvalidNumberSyntax(t *testing.T) {
	parser := getParser([]lex.Lexeme{
		testutil.MakeLexeme("1.3.2.2.422", lex.LNumber, 1),
	})

	_, err := parser.Parse()

	if _, isInvalidNumber := err.(parse.InvalidNumberError); !isInvalidNumber {
		t.Fatalf("Expected Parse() to fail on invalid number, but got: %s", err)
	}
}

func TestIncompleteInput(t *testing.T) {
	parser := getParser([]lex.Lexeme{
		testutil.MakeLexeme("true", lex.LBoolTrue, 1),
	})

	if _, err := parser.Parse(); err != parse.UnterminatedStatement {
		t.Fatalf("Expected unterminated statement error, but got: %v", err)
	}
}

func TestTrueFalse(t *testing.T) {
	parser := getParser([]lex.Lexeme{
		testutil.MakeLexeme("true", lex.LBoolTrue, 1),
		testutil.MakeLexeme(";", lex.LSemiColon, 2),
		testutil.MakeLexeme(" ", lex.LWhitespace, 3),
		testutil.MakeLexeme("false", lex.LBoolFalse, 4),
		testutil.MakeLexeme(";", lex.LSemiColon, 5),
	})

	expected := expectStatements(
		&parse.Statement{
			ParentNode: parse.ParentNode{
				Children: []parse.Node{
					&parse.BooleanLiteral{
						Value: true,
					},
				},
			},
		},
		&parse.Statement{
			ParentNode: parse.ParentNode{
				Children: []parse.Node{
					&parse.BooleanLiteral{
						Value: false,
					},
				},
			},
		},
	)

	assert.Equal(t, expected, testParse(parser, t))
}

func getParser(lexemes []lex.Lexeme) parse.Parser {
	return parse.NewParser(testutil.NewSimpleLexer(lexemes))
}

func expectStatements(statements ...*parse.Statement) parse.RootNode {
	return parse.RootNode{Statements: statements}
}

func testParse(p parse.Parser, t *testing.T) parse.RootNode {
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Unexpected Parse() error: %v", err)
	}

	return node
}