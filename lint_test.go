package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
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

type byPos []string

func (p byPos) Len() int {
	return len(p)
}

func (p byPos) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p byPos) Less(i, j int) bool {
	linei := p[i]
	linej := p[j]

	li, ci, _ := parseLine(linei)
	if li == 0 {
		return true
	}

	lj, cj, _ := parseLine(linej)
	if li == lj {
		return ci < cj
	}

	return li < lj
}

func setup() {
	var err error
	stderrReader, stderrWriter, err = os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = stderrWriter
}

func parseLine(line string) (uint, uint, string) {
	sep := strings.Index(line, ":")
	if line[sep+1] == ' ' {
		return 0, 0, line[sep+2:]
	}
	index := sep + 1

	lineLen := strings.Index(line[index:], ":")
	lineInfo := line[index : index+lineLen]

	l, err := strconv.ParseUint(lineInfo, 10, 16)
	if err != nil {
		panic(err)
	}
	index += lineLen + 1

	columLen := strings.Index(line[index:], ":")
	columnInfo := line[index : index+columLen]

	c, err := strconv.ParseUint(columnInfo, 10, 16)
	if err != nil {
		panic(err)
	}

	index += columLen + 2
	return uint(l), uint(c), line[index:]
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
		t.Fatal("ioutil.ReadAll failed:", err)
	}

	str := string(data)
	lines := strings.Split(str[0:len(str)-1], "\n")
	if len(msgs) != len(lines) {
		t.Fatalf("Expected %d violations, got %d",
			len(msgs), len(lines))
	}

	// Output of some linter functions is non-deterministic,
	// e.g. lintLocalVariables → sort output by token position.
	sort.Sort(byPos(lines))

	for n, m := range msgs {
		line, column, text := parseLine(lines[n])
		if line != m.l {
			t.Fatalf("expFail: Line didn't match, expected %d - got %d", m.l, line)
		}
		if column != m.c {
			t.Fatalf("expFail: Column didn't match, expected %d - got %d", m.c, column)
		}
		if text != m.s {
			t.Fatalf("expFail Expected string %q - got %q", m.s, text)
		}
	}
}

func TestLintComments(t *testing.T) {
	input := `#barfoo
#
# foobar
#	bazbar
#foobaz`

	l := newLinter(input)
	l.lintComments()

	expMsg(t,
		Msg{1, 1, badCommentPrefix},
		Msg{4, 1, badCommentPrefix},
		Msg{5, 1, badCommentPrefix})
}

func TestLintAddressComments(t *testing.T) {
	input := `# foo: foo bar <foo@bar.com>
# faz:
# foo:
# foo:
# foo:foo bar <foo@bar.com>
# foo: …`

	l := newLinter(input)
	n, addrs := l.lintAddressComments(" foo:")
	if n != 5 || len(addrs) != 1 || !l.v {
		t.Fail()
	}

	if addrs[0].a.String() != "\"foo bar\" <foo@bar.com>" {
		t.Fatalf("Expected %q - got %q", "foo",
			addrs[0].a.String())
	}

	expMsg(t,
		Msg{3, 1, missingAddress},
		Msg{4, 1, missingAddress},
		Msg{5, 1, noAddressSeparator},
		Msg{6, 1, invalidAddress})
}

func TestLintMaintainerAndContributors(t *testing.T) {
	t.Run("missingMaintainer", func(t *testing.T) {
		l := newLinter("")
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{0, 0, missingMaintainer})
	})

	t.Run("emptyMaintainer", func(t *testing.T) {
		l := newLinter("# Maintainer:")
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{1, 1, missingAddress})
	})

	t.Run("tooManyMaintainers", func(t *testing.T) {
		l := newLinter(`# Maintainer: A <a@a>
# Maintainer: B <b@b>`)
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{2, 1, tooManyMaintainers})
	})

	t.Run("maintainerAfterAssign", func(t *testing.T) {
		l := newLinter(`pkgname=foo
# Maintainer: A <a@b>`)
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{2, 1, maintainerAfterAssign})
	})

	t.Run("wrongAddrCommentOrder", func(t *testing.T) {
		l := newLinter(`# Maintainer: A <a@b>
# Contributor: B <b@c>`)
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{2, 1, wrongAddrCommentOrder})
	})

	t.Run("repeatedAddrComment", func(t *testing.T) {
		l := newLinter(`# Contributor: A <a@b>
# Contributor: A <a@b>
# Maintainer: M <m@m>`)
		l.lintMaintainerAndContributors()
		expMsg(t, Msg{2, 1, repeatedAddrComment})
	})

	t.Run("oneMaintainer", func(t *testing.T) {
		l := newLinter("# Maintainer: J <a@k>")
		l.lintMaintainerAndContributors()
		if l.v {
			t.Fail()
		}
	})

	t.Run("oneMaintainerAndContributors", func(t *testing.T) {
		l := newLinter(`# Contributor: A <a@a>
# Contributor: B <b@b>
# Maintainer: C <c@c>`)
		l.lintMaintainerAndContributors()
		if l.v {
			t.Fail()
		}
	})
}

func TestListGlobalVariables(t *testing.T) {
	input := `pkgname=foobar
foo=42
_foo=9001
__foo=bar
export ENV=23`

	l := newLinter(input)
	l.lintGlobalVariables()

	expMsg(t,
		Msg{2, 1, fmt.Sprintf(invalidGlobalVar, "foo")},
		Msg{4, 1, fmt.Sprintf(invalidGlobalVar, "__foo")})
}

func TestLintUnusedVariables(t *testing.T) {
	input := `pkgname=foobar
_foo=23
_bar=42
f1() {
foo=lol
}
f2() {
echo $_bar
}
f3() {
FOO=bar make
}
f4() {
export ENV=42
}`

	l := newLinter(input)
	l.lintUnusedVariables()

	expMsg(t,
		Msg{2, 1, fmt.Sprintf(variableUnused, "_foo")},
		Msg{5, 1, fmt.Sprintf(variableUnused, "foo")})
}

func TestLintGlobalCmdSubsts(t *testing.T) {
	input := `pkgname=bar
_bar=$(ls)
f1() {
local v1=${_bar}
}
_baz=$(cp -h)
f2() {
local v2=${_baz}
}
_baz=${foo} bar`

	l := newLinter(input)
	l.lintGlobalCmdSubsts()

	expMsg(t,
		Msg{2, 6, cmdSubstInGlobalVar},
		Msg{6, 6, cmdSubstInGlobalVar})
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
}
f4() {
for foobar in "a" "b" "c"; do echo "$foobar"; done
}
f5() {
export foo="bar"; echo "$foo"
}
VARFORCALLEXPR=23 ls`

	l := newLinter(input)
	l.lintLocalVariables()

	expMsg(t,
		Msg{2, 1, fmt.Sprintf(nonLocalVariable, "foo")},
		Msg{11, 5, fmt.Sprintf(nonLocalVariable, "foobar")})
}

func TestLintParamExpression(t *testing.T) {
	input := `# foobar
foo=${pkgname}
bar=$foo
# barfoo
foo=${pkgname##.*}
foo=${foobar}foobar
foo=${foobar}.$barfoo`

	l := newLinter(input)
	l.lintParamExpression()

	expMsg(t,
		Msg{2, 5, fmt.Sprintf(trivialLongParamExp, "pkgname", "pkgname")},
		Msg{7, 5, fmt.Sprintf(trivialLongParamExp, "foobar", "foobar")})
}

func TestVariablePlacement(t *testing.T) {
	input := `sha512sums=foobar
myfunc() {
echo myfunc
}
pkgname=barfoo`

	l := newLinter(input)
	l.lintMetadataPlacement()

	expMsg(t,
		Msg{1, 1, fmt.Sprintf(metadataAfterFunc, "sha512sums")},
		Msg{5, 1, fmt.Sprintf(metadataBeforeFunc, "pkgname")})
}

func TestLintRequiredMetadata(t *testing.T) {
	t.Run("allVarsDefined", func(t *testing.T) {
		input := `pkgname=foobar
pkgver=1337
pkgrel=2342
pkgdesc="foobar"
url=http://example.org
arch=all
license=MIT
sha512sums=1234`

		l := newLinter(input)
		l.lintRequiredMetadata()
		if l.v {
			t.Fail()
		}
	})

	t.Run("missingOneVar", func(t *testing.T) {
		input := `pkgname=foobar
pkgrel=2342
pkgdesc="foobar"
url=http://example.org
arch=all
license=MIT
sha512sums=1234`

		l := newLinter(input)
		l.lintRequiredMetadata()

		expMsg(t,
			Msg{0, 0, fmt.Sprintf(missingMetadata, "pkgver")})
	})
}

func TestLintFunctionOrder(t *testing.T) {
	t.Run("wrongFuncOrder", func(t *testing.T) {
		input := `package() {
}
build() {
}`

		l := newLinter(input)
		l.lintFunctionOrder()

		expMsg(t,
			Msg{1, 1, fmt.Sprintf(wrongFuncOrder, "package", "build")})
	})

	t.Run("rightFuncOrder", func(t *testing.T) {
		input := `prepare() {
}
build() {
}
check() {
}
package() {
}`

		l := newLinter(input)
		l.lintFunctionOrder()

		if l.v {
			t.Fail()
		}
	})
}

func TestLintBashisms(t *testing.T) {
	input := `[[ -e "$builddir" ]] && foo=bar
bar=*(foo bar)
echo >(true)
let x
declare -A foobar
readonly x
typeset -r x
nameref foo
echo ${#foo}
select s in foo bar baz; do
	echo $s
done
function f() {
return 1
}`

	l := newLinter(input)
	l.lintBashisms()

	expMsg(t,
		Msg{1, 1, fmt.Sprintf(forbiddenBashism, "test clause")},
		Msg{2, 5, fmt.Sprintf(forbiddenBashism, "extended globbing expression")},
		Msg{3, 6, fmt.Sprintf(forbiddenBashism, "process substitution")},
		Msg{4, 1, fmt.Sprintf(forbiddenBashism, "let clause")},
		Msg{5, 1, fmt.Sprintf(forbiddenBashism, "declare")},
		Msg{6, 1, fmt.Sprintf(forbiddenBashism, "readonly")},
		Msg{7, 1, fmt.Sprintf(forbiddenBashism, "typeset")},
		Msg{8, 1, fmt.Sprintf(forbiddenBashism, "nameref")},
		Msg{9, 6, fmt.Sprintf(forbiddenBashism, "advanced parameter expression")},
		Msg{10, 1, fmt.Sprintf(forbiddenBashism, "select clause")},
		Msg{13, 1, fmt.Sprintf(forbiddenBashism, "non-POSIX function declaration")})
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
