FROM golang:1.27-bookworm

RUN curl -fsSL https://chatgpt.com/codex/install.sh | sh

ENV PATH="/root/.local/bin:${PATH}"

RUN mkdir -p /root/.codex
