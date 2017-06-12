package runtime

import "errors"

type AddNumbers struct {
}

func (a AddNumbers) String() string {
	return "addition <native>"
}

func (a AddNumbers) Type() Type {
	return TypeInvokable
}

func (a AddNumbers) Invoke(context *Context, args []Value) (error, Value) {
	if one, isNumber := args[0].(Number); isNumber {
		if two, isNumber := args[1].(Number); isNumber {
			return nil, Number{Value: one.Value + two.Value}
		}
	}

	return errors.New("Invalid operands. Number addition requires two numbers"), nil
}

type SubtractNumbers struct {
}

func (a SubtractNumbers) String() string {
	return "subtraction <native>"
}

func (a SubtractNumbers) Type() Type {
	return TypeInvokable
}

func (a SubtractNumbers) Invoke(context *Context, args []Value) (error, Value) {
	if one, isNumber := args[0].(Number); isNumber {
		if two, isNumber := args[1].(Number); isNumber {
			return nil, Number{Value: one.Value - two.Value}
		}
	}

	return errors.New("Invalid operands. Subtraction requires two numbers"), nil
}
