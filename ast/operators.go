package ast

// Operator identifies written operations without importing a dialect's opcode.
type Operator int

const (
	OperatorUnknown Operator = iota
	OperatorLogicAnd
	OperatorLeftShift
	OperatorRightShift
	OperatorLogicOr
	OperatorGE
	OperatorLE
	OperatorEQ
	OperatorNE
	OperatorLT
	OperatorGT
	OperatorPlus
	OperatorMinus
	OperatorAnd
	OperatorOr
	OperatorMod
	OperatorXor
	OperatorDiv
	OperatorMul
	OperatorNot
	OperatorBitNeg
	OperatorIntDiv
	OperatorLogicXor
	OperatorNullEQ
	OperatorIn
	OperatorLike
	OperatorCase
	OperatorRegexp
	OperatorIsNull
	OperatorIsTruth
	OperatorIsFalsity
	OperatorJSONExtract
	OperatorJSONExtractText
	OperatorJSONPathExtract
	OperatorJSONPathExtractText
	OperatorJSONContains
	OperatorJSONContained
	OperatorJSONExists
	OperatorJSONAllExists
	OperatorJSONAnyExists
)
