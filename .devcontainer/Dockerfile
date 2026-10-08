FROM golang:1.27-bookworm

RUN curl -fsSL https://chatgpt.com/codex/install.sh | sh

ENV PATH="/root/.local/bin:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends byobu \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /root/.codex
