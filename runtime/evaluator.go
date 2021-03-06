package runtime

import (
	"io"

	"errors"
	"fmt"

	"github.com/ehimen/jaslang/parse"
)

type Evaluator interface {
	Evaluate(parse.Node) error
}

type evaluator struct {
	context *Context
}

func NewEvaluator(input io.Reader, output io.Writer, error io.Writer) Evaluator {
	table := NewTable()

	table.AddType("string", TypeString)
	table.AddType("boolean", TypeBoolean)
	table.AddType("number", TypeNumber)

	table.AddFunction("println", Println{})
	table.AddOperator("+", Types([]Type{TypeNumber, TypeNumber}), AddNumbers{})
	table.AddOperator("-", Types([]Type{TypeNumber, TypeNumber}), SubtractNumbers{})
	table.AddOperator("*", Types([]Type{TypeNumber, TypeNumber}), MultiplyNumbers{})
	table.AddOperator("/", Types([]Type{TypeNumber, TypeNumber}), DivideNumbers{})
	table.AddOperator("+", Types([]Type{TypeString, TypeString}), StringConcatenation{})
	table.AddOperator("&&", Types([]Type{TypeBoolean, TypeBoolean}), LogicAnd{})
	table.AddOperator("||", Types([]Type{TypeBoolean, TypeBoolean}), LogicOr{})
	table.AddOperator("==", Types([]Type{TypeNumber, TypeNumber}), Equality{})
	table.AddOperator("<", Types([]Type{TypeNumber, TypeNumber}), LessThan{})
	table.AddOperator(">", Types([]Type{TypeNumber, TypeNumber}), GreaterThan{})

	return &evaluator{context: &Context{Table: table, Input: input, Output: output, Error: error}}
}

func (e *evaluator) Evaluate(node parse.Node) error {

	if err, _ := e.evaluate(node); err != nil {
		return err
	}

	return nil
}

func (e *evaluator) evaluate(node parse.Node) (error, Value) {
	args := []Value{}

	if i, isIf := node.(*parse.If); isIf {
		return e.evaluateIf(i), nil
	}

	if parent, isParent := node.(parse.ContainsChildren); isParent {
		for _, child := range parent.Children() {
			// TODO: not recursion to avoid stack overflows.
			if err, arg := e.evaluate(child); err != nil {
				return err, nil
			} else {
				args = append(args, arg)
			}
		}
	} else if root, isRoot := node.(parse.RootNode); isRoot {
		for _, child := range root.Statements {
			// TODO: not recursion to avoid stack overflows.
			if err, arg := e.evaluate(child); err != nil {
				return err, nil
			} else {
				args = append(args, arg)
			}
		}
	}

	if str, isStr := node.(*parse.String); isStr {
		return nil, String{Value: str.Value}
	}

	if num, isNum := node.(*parse.Number); isNum {
		return nil, Number{Value: num.Value}
	}

	if boolean, isBool := node.(*parse.Boolean); isBool {
		return nil, Boolean{Value: boolean.Value}
	}

	if fn, isFn := node.(*parse.FunctionCall); isFn {
		return e.evaluateFunctionCall(fn, args)
	}

	if operator, isOperator := node.(*parse.Operator); isOperator {
		return e.evaluateOperator(operator, args)
	}

	if let, isLet := node.(*parse.Let); isLet {
		return e.evaluateLet(let, args)
	}

	if identifier, isIdentifier := node.(*parse.Identifier); isIdentifier {
		return e.evaluateIdentifier(identifier, args)
	}

	if assignment, isAssignment := node.(*parse.Assignment); isAssignment {
		return e.evaluateAssignment(assignment, args)
	}

	if _, isGroup := node.(*parse.Group); isGroup {
		if len(args) != 1 {
			return errors.New(fmt.Sprintf("Group should not have more than 1 child, actually has: %d", len(args))), nil
		}

		return nil, args[0]
	}

	// Nothing to do with statements/root as these are AST constructs (for now).
	if _, isStmt := node.(*parse.Statement); isStmt {
		return nil, nil
	}

	if _, isRoot := node.(parse.RootNode); isRoot {
		return nil, nil
	}

	return errors.New(fmt.Sprintf("Handling for %#v not yet implemented.", node)), nil
}

func (e *evaluator) evaluateFunctionCall(fn *parse.FunctionCall, args []Value) (error, Value) {
	if invokable, err := e.context.Table.Invokable(fn.Identifier.Identifier); err != nil {
		return err, nil
	} else {
		invokable.Invoke(e.context, args)
	}

	return nil, nil
}

func (e *evaluator) evaluateOperator(operator *parse.Operator, args []Value) (error, Value) {
	operands := Types([]Type{})

	for _, arg := range args {
		operands = append(operands, arg.Type())
	}

	if invokable, err := e.context.Table.Operator(operator.Operator, operands); err != nil {
		if unknownOperator, isUnknownOperator := err.(UnknownOperator); isUnknownOperator {
			unknownOperator.node = operator

			return unknownOperator, nil
		}

		return err, nil
	} else {
		return invokable.Invoke(e.context, args)
	}
}

func (e *evaluator) evaluateLet(let *parse.Let, args []Value) (error, Value) {
	if len(args) > 1 {
		return errors.New("Assignment with declaration must have at most one value"), nil
	}

	if valueType, err := e.context.Table.Type(let.Type.Identifier); err != nil {
		return err, nil
	} else if err := e.context.Table.Define(let.Identifier.Identifier, valueType); err != nil {
		return err, nil
	} else if len(args) == 1 {
		return e.setValue(*let.Identifier, args[0]), nil
	} else {
		return nil, nil
	}
}

func (e *evaluator) evaluateIdentifier(identifier *parse.Identifier, args []Value) (error, Value) {
	val, err := e.context.Table.Get(identifier.Identifier)

	return applyUnknownIdentifierNode(err, *identifier), val
}

func (e *evaluator) evaluateAssignment(assignment *parse.Assignment, args []Value) (error, Value) {
	if len(args) != 1 {
		return errors.New("Assignment must have at exactly one value"), nil
	}

	return e.setValue(*assignment.Identifier, args[0]), nil
}

func (e *evaluator) evaluateIf(node *parse.If) error {
	err, result := e.evaluate(node.Condition())

	if err != nil {
		return err
	}

	if boolValue, isBool := result.(Boolean); !isBool {
		return errors.New("If condition must evaluate to boolean")
	} else {
		if boolValue.Value {
			for _, child := range node.ParentNode.Children() {
				e.evaluate(child)
			}
		}
	}

	return nil
}

func (e evaluator) setValue(identifier parse.Identifier, value Value) error {
	err := e.context.Table.Set(identifier.Identifier, value)

	if invalidType, isInvalidType := err.(InvalidType); isInvalidType {
		invalidType.node = identifier

		return invalidType
	}

	return applyUnknownIdentifierNode(err, identifier)
}

func applyUnknownIdentifierNode(err error, identifier parse.Identifier) error {
	if unknownIdentifier, isUnknownIdentifier := err.(UnknownIdentifier); isUnknownIdentifier {
		unknownIdentifier.node = identifier

		return unknownIdentifier
	}

	return nil
}
