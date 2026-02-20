.PHONY: all quick logger field formatter handler concurrent compare profile clean help

# Benchmark targets

all: ## Run all benchmarks
	@echo "Running all benchmarks..."
	go test -bench=. -benchmem | tee benchmark_results.txt

quick: ## Run quick benchmarks (100ms)
	@echo "Running quick benchmarks..."
	go test -bench=. -benchmem -benchtime=100ms

logger: ## Run logger creation benchmarks
	@echo "Running logger benchmarks..."
	go test -bench='BenchmarkLogger|BenchmarkWith' -benchmem

field: ## Run field type benchmarks
	@echo "Running field benchmarks..."
	go test -bench='BenchmarkFieldTypes|BenchmarkInfo.*Field' -benchmem

formatter: ## Run formatter benchmarks
	@echo "Running formatter benchmarks..."
	go test -bench='BenchmarkFormatters|BenchmarkWriterFormatter' -benchmem

handler: ## Run handler benchmarks
	@echo "Running handler benchmarks..."
	go test -bench='BenchmarkSyncVsAsync|BenchmarkFileHandler|BenchmarkMultiHandler|BenchmarkOverflow' -benchmem

concurrent: ## Run concurrent logging benchmarks
	@echo "Running concurrent benchmarks..."
	go test -bench='BenchmarkConcurrentLogging' -benchmem

profile: ## Run benchmarks with profiling
	@echo "Running benchmarks with CPU profiling..."
	go test -bench=BenchmarkInfo -benchmem -cpuprofile=cpu.prof -benchtime=3s
	@echo "Running benchmarks with memory profiling..."
	go test -bench=BenchmarkInfo -benchmem -memprofile=mem.prof -benchtime=3s
	@echo ""
	@echo "Profile files created:"
	@echo "  - cpu.prof"
	@echo "  - mem.prof"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof cpu.prof"
	@echo "  go tool pprof mem.prof"

compare: ## Compare with baseline (usage: make compare BASELINE=old.txt)
	@if [ -z "$(BASELINE)" ]; then \
		echo "Error: BASELINE variable required"; \
		echo "Usage: make compare BASELINE=old.txt"; \
		exit 1; \
	fi
	@if [ ! -f "$(BASELINE)" ]; then \
		echo "Error: Baseline file '$(BASELINE)' not found"; \
		exit 1; \
	fi
	@echo "Creating new benchmark results..."
	@go test -bench=. -benchmem > new_results.txt
	@echo ""
	@echo "Comparing with baseline..."
	@echo "Tip: Install benchstat for detailed comparison:"
	@echo "  go install golang.org/x/perf/cmd/benchstat@latest"
	@echo ""
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat $(BASELINE) new_results.txt; \
	else \
		echo "benchstat not found. Install it with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

baseline: ## Create a baseline benchmark file
	@echo "Creating baseline benchmark results..."
	go test -bench=. -benchmem > baseline.txt
	@echo "Baseline saved to baseline.txt"

clean: ## Clean benchmark artifacts
	@echo "Cleaning benchmark artifacts..."
	rm -f *.prof *.test *.txt *.out

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help

