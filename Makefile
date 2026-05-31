SHELL := /usr/bin/env bash

.PHONY: test eval train feedback

test:
	GOWORK=off go test ./...

eval:
	mkdir -p ./datasets/generated/eval ./.cache/eval-model-zh ./.cache/eval-model-en
	GOWORK=off go run ./cmd/chat-nlu-train \
		-lang zh \
		-extra-csv ./datasets/default/zh_business.csv,./datasets/default/zh_skill_routing.csv \
		-out ./.cache/eval-model-zh \
		-version eval.zh.seed42 \
		-threshold 0.55 \
		-split-enabled=true \
		-train-ratio 0.8 \
		-val-ratio 0.1 \
		-test-ratio 0.1 \
		-seed 42 \
		-auto-calibrate-thresholds=true \
		-disable-taxonomy=true \
		-eval-report ./datasets/generated/eval/zh_eval.json \
		-source-name default-business \
		-source-version repository \
		-source-revision zh
	GOWORK=off go run ./cmd/chat-nlu-train \
		-lang en \
		-extra-csv ./datasets/default/en_business.csv,./datasets/default/en_skill_routing.csv \
		-out ./.cache/eval-model-en \
		-version eval.en.seed42 \
		-threshold 0.55 \
		-split-enabled=true \
		-train-ratio 0.8 \
		-val-ratio 0.1 \
		-test-ratio 0.1 \
		-seed 42 \
		-auto-calibrate-thresholds=true \
		-disable-taxonomy=true \
		-eval-report ./datasets/generated/eval/en_eval.json \
		-source-name default-business \
		-source-version repository \
		-source-revision en

train:
	./scripts/train_chatterbot_models.sh

feedback:
	@echo "Usage: ./scripts/feedback_loop.sh --input <feedback.csv>"
