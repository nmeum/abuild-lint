# abuild-lint

A linting utility for Alpine Linux APKBUILDs.

## Scope

Alpine Linux currently doesn't have a policy document describing how
APKBUILDs should be written. This tool tries to enforce some of
unwritten style practises and thus make it easier for contributors to
check their APKBUILDs for mistakes regarding style. It is not intended
to replace a policy document though.

## Installation

abuild-lint can either be installed using `go get` or using the provided
`GNUmakefile`. The latter installation method is preferred and boils
down to the following commands:

	$ make
	$ make install

## Documentation

End user documentation, which also documents which checks are performed
by abuild-lint, is available in the form of a man page.

The source code itself is documented using the standard go documentation
format. The documentation can be viewed using:

	$ go doc -cmd -u

## Tests

abuild-lint comes with a unit testsuite which can either be run using
`go test` or using the `check` target of the provided `GNUmakefile`.

## FAQ

*Q:* Why not write a code formating tool like `go fmt` instead?
*A:* The purpose of a formating tool is formating source code while the
purpose of this tool is warning about style mistakes. Some of the
mistakes abuild-lint currently warns about could be automatically fixed
by a formating tool and might be handled by a formating tool one day.

*Q:* What's the difference between `abuild sanitycheck` and abuild-lint?
*A:* `abuild sanitycheck` is concerned with the semantical correctness
of an APKBUILD while abuild-lint is concerned with the syntactical
correctness.

## License

This program is free software: you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the
Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
Public License for more details.

You should have received a copy of the GNU General Public License along
with this program. If not, see <http://www.gnu.org/licenses/>.
