NAME = abuild-lint
VER  = 0.5

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man
DOCDIR ?= $(PREFIX)/share/doc/$(NAME)

IMPORTPATH=src/github.com/nmeum/$(NAME)
export GOPATH=$(CURDIR)

all: $(NAME)
$(IMPORTPATH): $(GOPATH)
	mkdir -p $(shell dirname $@)
	ln -fs $< $@

check: $(IMPORTPATH)
	cd $< && go test
$(NAME): $(IMPORTPATH)
	cd $< && go build -o $@

install: $(NAME) $(NAME).1 README.md
	install -Dm755 $(NAME) "$(DESTDIR)$(BINDIR)/$(NAME)"
	install -Dm644 $(NAME).1 "$(DESTDIR)$(MANDIR)/man1/$(NAME).1"
	install -Dm644 README.md "$(DESTDIR)$(DOCDIR)/README.md"

dist:
	mkdir -p $(NAME)-$(VER)
	cp -R $(wildcard *.go) $(wildcard *.md) GNUmakefile \
		$(NAME).1 vendor $(NAME)-$(VER)
	find $(NAME)-$(VER) -name '.git' -exec rm -rf {} +
	tar -czf $(NAME)-$(VER).tar.gzip $(NAME)-$(VER)
	rm -rf $(NAME)-$(VER)

.PHONY: all check install dist
