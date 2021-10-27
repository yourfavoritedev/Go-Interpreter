package vm

import (
	"fmt"

	"github.com/yourfavoritedev/golang-interpreter/code"
	"github.com/yourfavoritedev/golang-interpreter/compiler"
	"github.com/yourfavoritedev/golang-interpreter/object"
)

const StackSize = 10

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}

// VM is the struct for our virtual-machine. It holds the bytecode instructions and constants-pool generated by the compiler.
// A VM implements a stack, as it executes the bytecode, it organizes (push, pop, etc) the evaluated constants on the stack.
// The field sp helps keep track of the position of the next item in the stack (top to bottom).
type VM struct {
	constants    []object.Object
	instructions code.Instructions
	stack        []object.Object
	// sp always points to the next free slot in the stack. If there's one element on the stack,
	// located at index 0, the value of sp would be 1 and to access that element we'd use stack[sp-1].
	sp int
}

// New initializes a new VM using the bytecode generated by the compiler.
// VM's are initialized with an sp of 0 (the initial top). The stack
// will have a preallocated number of elements (StackSize).
func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,
		stack:        make([]object.Object, StackSize),
		sp:           0,
	}
}

// Run will start the VM. The VM will execute the bytecode and handle
// the specific instructions (opcode + operands) that it was provided
// from the compiler. It executes the fetch-decode-execute cycle.
func (vm *VM) Run() error {
	// iterate across all bytecode instructions
	for ip := 0; ip < len(vm.instructions); ip++ {
		// FETCH the instruction (opcode + operand) at the specific position (ip, the instruction pointer)
		// then convert the instruction's first-byte into an Opcode (which is what we expect it to be)
		op := code.Opcode(vm.instructions[ip])
		// DECODE SECTION
		switch op {
		// OpConstant has an operand to decode
		case code.OpConstant:
			// grab the two-byte operand for the OpConstant instruction (the operand starts right after the Opcode byte)
			operand := vm.instructions[ip+1:]
			// decode the operand, getting back the identifier for the constant's position in the constants pool
			constIndex := code.ReadUint16(operand)
			// increment the instruction-pointer by 2 because OpConstant has one two-byte wide operand
			ip += 2
			// EXECUTE, grab the constant from the pool and push it on to the stack
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		// Execute the binary operation for the Opcode arithmetic instruction.
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		// Execute the comparison operation for the Opcode comparison instruction.
		case code.OpGreaterThan, code.OpEqual, code.OpNotEqual:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		// Execute the minus "-" operation for this Opcode instruction.
		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		// Execute the bang "!" operation for this Opcode instruction.
		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}

		// Execute the boolean Opcode instructions. Simply push the corresponding Object.Boolean to the stack.
		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}
		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		// Execute OpJump instruction to jump to the next instruction byte after compiing a truthy condition.
		case code.OpJump:
			operand := vm.instructions[ip+1:]
			// decode the operand and get back the absolute position of the byte to jump to
			pos := int(code.ReadUint16(operand))
			// since we're in a loop that increments ip with each iteration, we need to set ip
			// to the offset right before the one we want. That lets the loop do its work
			// and ip gets set to the value we want in the next cycle to process that instruction
			ip = pos - 1

		case code.OpJumpNotTruthy:
			operand := vm.instructions[ip+1:]
			// decode the operand and get back the absolute position of the byte to jump to if condition is not truthy
			pos := int(code.ReadUint16(operand))
			// increment the instruction-pointer by 2 because OpJumpNotTruthy has one two-byte wide operand
			// this would prepare us for the next iteration to evaluate the OpConstant - the result of a truthy condition
			ip += 2

			// pop the condition constant (True or False) and determine where we need to jump
			condition := vm.pop()
			if !isTruthy(condition) {
				// jump pass the consequence when the condition is falsey to process the next instruction
				ip = pos - 1
			}

		// OpPop has no operands and simply pops an element from the stack
		case code.OpPop:
			// EXECUTE: pop the element before the stack pointer
			vm.pop()
		}
	}

	return nil
}

// isTruthy simply asserts the provided object to be an object.Boolean
// and returns its boolean value
func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	default:
		return true
	}
}

// push validates the stack size and adds the provided object (o) to the
// next available slot in the stack, finally it preps the stackpointer (sp),
// incrementing it to designate the next slot to be allocated
func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

// LastPoppedStackElem helps identify the last element that was popped from the stack as the VM executes through it.
// If a stack had two elements [a, b], sp would be at index 2. If the vm pops an element,
// it would pop the element at [sp-1], so index 1, and then sp is moved to index 1.
// Leaving b to be the last popped stack element.
func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

// pop simply grabs the constant sittng 1 position above the stackpointer,
// it then decrements the stack pointer to be aware of the updated position,
// leaving that slot to be eventually overwritten with a new constant
func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

// executeBinaryOperation pops the two constants above the stack-pointer
// and validates what type of binary operation to run with them. If the combination
// of types do not have a valid operation an error is returned.
func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		return vm.executeBinaryIntegerOperation(op, left, right)
	}

	return fmt.Errorf("unsupported types for binary operation: %s, %s",
		leftType, rightType)
}

// executeBinaryIntegerOperation will perform an arithmetic operation
// with the provided operator and objects. If the operation is successful,
// the new evaluated object is pushed on to the stack.
func (vm *VM) executeBinaryIntegerOperation(
	op code.Opcode,
	left, right object.Object,
) error {
	// assert the Objects to grab their integer value
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64
	// handle arithmetic operation
	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operation: %d", op)
	}

	// push the Object to the stack
	return vm.push(&object.Integer{Value: result})
}

// executeComparison will compare the two constants directly above the stack-pointer
// and then push the result on to the stack. It validates the type of the two constants (object.Object)
// to determine what comparison helper to run this pattern.
func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}

	// compare of pointer-addresses. For boolean objects,
	// right and left are holding the constants TRUE and FALSE listed, and we
	// are reusing those constants so we can compare their pointer-addresses.
	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d, (%s %s)",
			op, leftType, rightType)
	}
}

// executeIntegerComparison is the helper to compare two integer constants. It asserts
// the two constants as *object.Integers and compares their values. With the result
// of the comparison, it constructs a Boolean Object and pushes it to the stack.
func (vm *VM) executeIntegerComparison(
	op code.Opcode,
	left, right object.Object,
) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result *object.Boolean
	switch op {
	case code.OpGreaterThan:
		result = nativeBoolToBooleanObject(leftValue > rightValue)
	case code.OpEqual:
		result = nativeBoolToBooleanObject(leftValue == rightValue)
	case code.OpNotEqual:
		result = nativeBoolToBooleanObject(leftValue != rightValue)
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}

	return vm.push(result)

}

// nativeBoolToBooleanObject simply converts a traditional boolean
// to an *object.Boolean
func nativeBoolToBooleanObject(b bool) *object.Boolean {
	if b {
		return True
	}
	return False
}

// executeBangOperator handles the execution of an instruction for a OpBang Opcode.
// It pops the constant above the stack pointer and negates it with the "!" prefix.
// If the constant is truthy we will push False to the stack. If the constant is falsey
// we will push True to the stack.
func (vm *VM) executeBangOperator() error {
	operand := vm.pop()

	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

// executeMinusOperator handles the execution of an isntruction for an OpMinus Opcode.
// It pops the constant above the stack pointer and negates it with the "-" prefix.
// It will construct a new Integer Object, with its value inversed and push that to the stack.
func (vm *VM) executeMinusOperator() error {
	right := vm.pop()

	if right.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported type for negation: %s", right.Type())
	}

	rightValue := right.(*object.Integer).Value

	return vm.push(&object.Integer{Value: -rightValue})
}
