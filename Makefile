BINARY_NAME := svcc
INSTALL_DIR := $(HOME)/.svcc
ZSHRC := $(HOME)/.zshrc

.PHONY: install-zshrc
install-zshrc:
	@mkdir -p "$(INSTALL_DIR)"
	@go build -o "$(INSTALL_DIR)/$(BINARY_NAME)" .
	@if ! grep -qxF 'export PATH="$$HOME/.svcc:$$PATH"' "$(ZSHRC)" 2>/dev/null; then \
		printf '\nexport PATH="$$HOME/.svcc:$$PATH"\n' >> "$(ZSHRC)"; \
	fi
	@echo "Installed $(BINARY_NAME) at $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "PATH configured in $(ZSHRC). Restart your shell or run: source $(ZSHRC)"
