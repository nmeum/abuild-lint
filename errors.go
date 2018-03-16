package main

const (
	variableUnused      = "Global variable %q is unused"
	nonLocalVariable    = "Variable %q was not declared using the local keyword"
	wrongFuncOrder      = "Function %q should be declared after function %q"
	trivialLongParamExp = "Parameter Expansion \"${%s}\" can be replaced by a short Expansion \"$%s\""
	metadataAfterFunc   = "Variable %q should be declared after the last function declaration"
	metadataBeforeFunc  = "Variable %q should be declared before the first function declaration"

	badCommentPrefix      = "Comment doesn't start with a space"
	missingMaintainer     = "Maintainer is missing"
	missingAddress        = "Comment is missing an RFC 5322 address"
	noAddressSeparator    = "Mail address should be seperated from prefix with a space"
	invalidAddress        = "Mail address doesn't conform to RFC 5322"
	tooManyMaintainers    = "Only one maintainer can be specified"
	invalidGlobalVar      = "Custom global variables should start with an '_'"
	cmdSubstInGlobalVar   = "$(…) shouldn't be used in global variables"
	maintainerAfterAssign = "The maintainer comment should be declared before any assignment"
	repeatedAddrComment   = "Contributor comment with this RFC 5322 has already been defined"
	wrongAddrCommentOrder = "Contributor comment should be defined before the maintainer comment"
)
