.PHONY: ci install-hooks

ci:
	python3 scripts/check_markdown.py lines
	python3 scripts/check_markdown.py width

install-hooks:
	git config --local core.hooksPath .githooks
