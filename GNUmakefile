PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man
DOCDIR ?= $(PREFIX)/doc/abuild-lint

IMPORTPATH=src/github.com/nmeum/abuild-lint
export GOPATH=$(CURDIR)

all: abuild-lint
$(IMPORTPATH): $(GOPATH)
	mkdir -p $(shell dirname $@)
	ln -fs $< $@

check: $(IMPORTPATH)
	cd $< && go test
abuild-lint: $(IMPORTPATH)
	cd $< && go build -o $@

install: abuild-lint abuild-lint.1 README.md
	install -Dm755 abuild-lint "$(DESTDIR)$(BINDIR)/abuild-lint"
	install -Dm644 abuild-lint.1 "$(DESTDIR)$(MANDIR)/man1/abuild-lint.1"
	install -Dm644 README.md "$(DESTDIR)$(DOCDIR)/README.md"

.PHONY: all check install
