package main

const (
	badCommentPrefix   = "Comment doesn't start with a space"
	missingMaintainer  = "Maintainer is missing"
	missingAddress     = "Comment is missing an RFC 5322 address"
	noAddressSeperator = "Mail address should be seperated from prefix with a space"
	invalidAddress     = "Mail address doesn't conform to RFC 5322"
	tooManyMaintainers = "Only one maintainer can be specified"
	invalidGlobalVar   = "Custom global variables should start with an '_'"
)
