package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

const name = "Testinput"

var stderrReader *os.File
var stderrWriter *os.File

type Msg struct {
	l uint
	c uint
	s string
}

func setup() {
	var err error
	stderrReader, stderrWriter, err = os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = stderrWriter
}

func newLinter(input string) *Linter {
	reader := strings.NewReader(input)
	abuild, err := Parse(reader, name)
	if err != nil {
		panic(err)
	}

	linter := Linter{f: abuild}
	return &linter
}

func expMsg(t *testing.T, msgs ...Msg) {
	stderrWriter.Close() // Write EOF
	defer setup()

	data, err := ioutil.ReadAll(stderrReader)
	if err != nil {
		t.Fail()
	}

	str := string(data)
	lines := strings.Split(str[0:len(str)-1], "\n")
	if len(msgs) != len(lines) {
		t.Fail()
	}

	for n, m := range msgs {
		line := lines[n]
		index := len(name)

		if line[0:index] != name {
			t.Fatalf("expFail: Expected name %q - got %q",
				name, line[0:index])
		}

		if line[index] != ':' {
			t.Fatal("expFail: Missing seperator")
		}

		if m.l > 0 {
			sindex := strings.Index(line, " ")
			if sindex == -1 {
				t.Fail()
			}

			apos := line[index+1 : sindex]
			epos := fmt.Sprintf("%v:%v:", m.l, m.c)
			if apos != epos {
				t.Fatalf("expFail: Expected positon %q - got %q",
					epos, apos)
			}

			index = sindex
		} else {
			index++ // skip space
		}

		if line[index] != ' ' {
			t.Fail()
		}

		index++
		if line[index:len(line)] != m.s {
			t.Fatalf("expFail Expected string %q - got %q",
				m.s, line[index:len(line)])
		}
	}
}

func TestLintComments(t *testing.T) {
	input := `#barfoo
# foobar
#	bazbar
#foobaz`

	l := newLinter(input)
	l.lintComments()

	expMsg(t,
		Msg{1, 1, badCommentPrefix},
		Msg{3, 1, badCommentPrefix},
		Msg{4, 1, badCommentPrefix})
}

func TestLintAddressComments(t *testing.T) {
	input := `# foo: foo bar <foo@bar.com>
# faz:
# foo:
# foo:
# foo:foo bar <foo@bar.com>
# foo: …`

	l := newLinter(input)
	r := l.lintAddressComments(" foo:")
	if r != 5 || !l.v {
		t.Fail()
	}

	expMsg(t,
		Msg{3, 1, missingAddress},
		Msg{4, 1, missingAddress},
		Msg{5, 1, noAddressSeperator},
		Msg{6, 1, invalidAddress})
}

func TestLintMaintainer(t *testing.T) {
	t.Run("missingMaintainer", func(t *testing.T) {
		l := newLinter("")
		l.lintMaintainer()
		expMsg(t, Msg{0, 0, missingMaintainer})
	})

	t.Run("oneMaintainer", func(t *testing.T) {
		l := newLinter("# Maintainer: J <a@k>")
		l.lintMaintainer()
		if l.v {
			t.Fail()
		}
	})

	t.Run("tooManyMaintainers", func(t *testing.T) {
		l := newLinter(`# Maintainer: foo <foo@bar>
# Maintainer: bar <bar@foo>`)
		l.lintMaintainer()
		expMsg(t, Msg{0, 0, tooManyMaintainers})
	})
}

func TestLintContributors(t *testing.T) {
	input := `# Contributor: Mark <mark@example.com>
# Contributor: Peter <peter@example.org>`

	l := newLinter(input)
	l.lintContributers()
	if l.v {
		t.Fail()
	}
}

func TestListGlobalVariables(t *testing.T) {
	input := `pkgname=foobar
foo=42
_foo=9001
_bar=hoho
pkgver=$_bar`

	l := newLinter(input)
	l.lintGlobalVariables()

	expMsg(t,
		Msg{2, 1, invalidGlobalVar},
		Msg{0, 0, fmt.Sprintf(variableUnused, "_foo")})
}

func TestLintLocalVariables(t *testing.T) {
	input := `f1() {
foo=123
}
f2() {
local foo=123
}
f3() {
local bar=456
}`

	l := newLinter(input)
	l.lintLocalVariables()

	expMsg(t,
		Msg{2, 1, fmt.Sprintf(nonLocalVariable, "foo")})
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
