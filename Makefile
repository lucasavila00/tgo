.PHONY: ci install-hooks

ci:
	python3 scripts/check_markdown.py

install-hooks:
	git config --local core.hooksPath .githooks
