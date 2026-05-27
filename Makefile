.PHONY: build build-wrappers test install clean

BINARY     := google-workspace-mcp-auth
SLACK      := slack-mcp-auth
HUBSPOT    := hubspot-mcp-auth
ALLY       := ally3p
CMD_BINARY := ./cmd/google-workspace-mcp-auth
CMD_SLACK  := ./cmd/slack-mcp-auth
CMD_HUBSPOT := ./cmd/hubspot-mcp-auth
CMD_ALLY   := ./cmd/ally-claude-3p
EMBED_DIR  := cmd/ally-claude-3p/embedded
SERVICES   := gmail drive calendar chat people

build: build-bin build-slack build-hubspot embed build-ally

build-bin:
	go build -o bin/$(BINARY) $(CMD_BINARY)

build-slack:
	go build -o bin/$(SLACK) $(CMD_SLACK)

build-hubspot:
	go build -o bin/$(HUBSPOT) $(CMD_HUBSPOT)

# Copy freshly built helpers into the embed dir so ally3p bundles them.
embed: build-bin build-slack build-hubspot
	cp bin/$(BINARY) $(EMBED_DIR)/$(BINARY)
	cp bin/$(SLACK) $(EMBED_DIR)/$(SLACK)
	cp bin/$(HUBSPOT) $(EMBED_DIR)/$(HUBSPOT)

build-ally:
	go build -o bin/$(ALLY) $(CMD_ALLY)

install: build
	install -m 755 bin/$(BINARY) /usr/local/bin/$(BINARY)
	install -m 755 bin/$(ALLY)   /usr/local/bin/$(ALLY)
	@for s in $(SERVICES); do \
	  WRAPPER="/usr/local/bin/$(BINARY)-$$s"; \
	  printf '#!/bin/sh\nexec "%s" %s "$$@"\n' "/usr/local/bin/$(BINARY)" "$$s" > "$$WRAPPER"; \
	  chmod 755 "$$WRAPPER"; \
	done

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
	rm -f $(EMBED_DIR)/$(BINARY) $(EMBED_DIR)/$(SLACK) $(EMBED_DIR)/$(HUBSPOT)
