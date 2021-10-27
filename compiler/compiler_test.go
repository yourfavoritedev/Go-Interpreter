package compiler

import (
	"fmt"
	"testing"

	"github.com/yourfavoritedev/golang-interpreter/ast"
	"github.com/yourfavoritedev/golang-interpreter/code"
	"github.com/yourfavoritedev/golang-interpreter/lexer"
	"github.com/yourfavoritedev/golang-interpreter/object"
	"github.com/yourfavoritedev/golang-interpreter/parser"
)

type compilerTestCase struct {
	input                string
	expectedConstants    []interface{}
	expectedInstructions []code.Instructions
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: "1 + 2",
			// 1 is the first constant, so its position is 0
			// 2 is the second constant, so its position is 1
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// the operand is an identifier for the position of the
				// the evaluated constant in the constant pool
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpAdd),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1; 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpPop),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 - 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpSub),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 * 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpMul),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 / 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpDiv),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "-1",
			expectedConstants: []interface{}{1},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpMinus),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func runCompilerTests(t *testing.T, tests []compilerTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		compiler := New()
		err := compiler.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		bytecode := compiler.Bytecode()

		err = testInstructions(tt.expectedInstructions, bytecode.Instructions)
		if err != nil {
			t.Fatalf("testInstructions failed: %s", err)
		}

		err = testConstants(t, tt.expectedConstants, bytecode.Constants)
		if err != nil {
			t.Fatalf("testConstants failed: %s", err)
		}
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testInstructions(
	expected []code.Instructions,
	actual code.Instructions,
) error {
	concatted := concatInstructions(expected)

	if len(actual) != len(concatted) {
		return fmt.Errorf("wrong instructions length.\nwant=%q\ngot=%q",
			concatted, actual)
	}

	for i, ins := range concatted {
		if actual[i] != ins {
			return fmt.Errorf("wrong instruction at %d.\nwant=%q\ngot=%q",
				i, concatted, actual)
		}
	}

	return nil
}

// flattens s (the instructions) from a slice of slices of byte to a single byte-slice.
func concatInstructions(s []code.Instructions) code.Instructions {
	out := code.Instructions{}

	// iterate over all byte-slices to construct the new byte-slice
	for _, ins := range s {
		// spread over all bytes in the instruction
		out = append(out, ins...)
	}

	return out
}

func testConstants(
	t *testing.T,
	expected []interface{},
	actual []object.Object,
) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("wrong number of constants. got=%d, want=%d",
			len(actual), len(expected))
	}

	for i, constant := range expected {
		switch constant := constant.(type) {
		case int:
			err := testIntegerObject(int64(constant), actual[i])
			if err != nil {
				return fmt.Errorf("constant %d - testIntegerObject failed: %s",
					i, err)
			}
		}
	}

	return nil
}

func testIntegerObject(expected int64, actual object.Object) error {
	// assert actual is an integer object
	result, ok := actual.(*object.Integer)
	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)", actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
	}

	return nil
}

func TestBooleanExpressions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "true",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpTrue),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "false",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpFalse),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 > 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpGreaterThan),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 < 2",
			expectedConstants: []interface{}{2, 1},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpGreaterThan),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 == 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpEqual),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 != 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpNotEqual),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "true == false",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpTrue),
				code.Make(code.OpFalse),
				code.Make(code.OpEqual),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "true != false",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpTrue),
				code.Make(code.OpFalse),
				code.Make(code.OpNotEqual),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "!true",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpTrue),
				code.Make(code.OpBang),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestConditionals(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: `
			if (true) { 10 }; 3333;
			`,
			expectedConstants: []interface{}{10, 3333},
			expectedInstructions: []code.Instructions{
				// 0000
				code.Make(code.OpTrue), // 1 byte wide
				// 0001
				code.Make(code.OpJumpNotTruthy, 10), // 3 bytes wide
				// 0004
				code.Make(code.OpConstant, 0), // 3 bytes wide
				// 0007
				code.Make(code.OpJump, 11), // 3 bytes wide
				// 0010
				code.Make(code.OpNull), // 1 byte wide
				// 0011
				code.Make(code.OpPop), // 1 byte wide
				// 0012
				code.Make(code.OpConstant, 1), // 3 bytes wide
				// 0011
				code.Make(code.OpPop), // 1 byte wide
			},
		},
		{
			input: `
			if (true) { 10 } else { 20 }; 3333;
			`,
			expectedConstants: []interface{}{10, 20, 3333},
			expectedInstructions: []code.Instructions{
				// 0000
				code.Make(code.OpTrue), // 1 byte wide
				// 0001
				code.Make(code.OpJumpNotTruthy, 10), // 3 bytes wide
				// 0004
				code.Make(code.OpConstant, 0), // 3 bytes wide
				// 0007
				code.Make(code.OpJump, 13), // 3 bytes wide
				// 0010
				code.Make(code.OpConstant, 1), // 3 bytes wide
				// 0013
				code.Make(code.OpPop), // 1 byte wide
				// 0014
				code.Make(code.OpConstant, 2), // 3 bytes wide
				// 0017
				code.Make(code.OpPop), // 1 byte wide
			},
		},
		{
			input: `
			if (false) { 10 } else { 20 };
			`,
			expectedConstants: []interface{}{10, 20},
			expectedInstructions: []code.Instructions{
				// 0000
				code.Make(code.OpFalse), // 1 byte wide
				// 0001
				code.Make(code.OpJumpNotTruthy, 10), // 3 bytes wide
				// 0004
				code.Make(code.OpConstant, 0), // 3 bytes wide
				// 0007
				code.Make(code.OpJump, 13), // 3 bytes wide
				// 0010
				code.Make(code.OpConstant, 1), // 3 bytes wide
				// 0013
				code.Make(code.OpPop), // 1 byte wide
			},
		},
		{
			input: `
			if (false) { 10 };
			`,
			expectedConstants: []interface{}{10},
			expectedInstructions: []code.Instructions{
				// 0000
				code.Make(code.OpFalse), // 1 byte wide
				// 0001
				code.Make(code.OpJumpNotTruthy, 10), // 3 bytes wide
				// 0004
				code.Make(code.OpConstant, 0), // 3 bytes wide
				// 0007
				code.Make(code.OpJump, 11), // 3 bytes wide
				// 0010
				code.Make(code.OpNull), // 1 byte wide
				// 0011
				code.Make(code.OpPop), // 1 byte wide
			},
		},
	}

	runCompilerTests(t, tests)
}
