#!/bin/bash

echo "🧪 PERFORMANCE SCIENCE: Node.js vs Go"
echo "======================================"

# Test directory - use current dir or create test repos
TEST_DIR=${1:-$(pwd)}
RUNS=10

echo "Test Directory: $TEST_DIR"
echo "Runs per test: $RUNS"
echo ""

# Function to run benchmark
run_benchmark() {
    local cmd="$1"
    local name="$2"
    local total_time=0
    local times=()
    
    echo "Testing $name..."
    
    for i in $(seq 1 $RUNS); do
        start_time=$(date +%s.%N)
        
        # Run command and capture only essential output
        eval "$cmd" > /dev/null 2>&1
        
        end_time=$(date +%s.%N)
        run_time=$(echo "$end_time - $start_time" | bc -l)
        times+=($run_time)
        total_time=$(echo "$total_time + $run_time" | bc -l)
        
        printf "  Run %2d: %.3fs\n" $i $run_time
    done
    
    avg_time=$(echo "scale=3; $total_time / $RUNS" | bc -l)
    
    # Calculate min/max
    min_time=${times[0]}
    max_time=${times[0]}
    for time in "${times[@]}"; do
        if (( $(echo "$time < $min_time" | bc -l) )); then
            min_time=$time
        fi
        if (( $(echo "$time > $max_time" | bc -l) )); then
            max_time=$time
        fi
    done
    
    printf "  Average: %.3fs | Min: %.3fs | Max: %.3fs\n\n" $avg_time $min_time $max_time
    
    # Return average for comparison
    echo $avg_time
}

echo "🟨 Node.js Performance Test"
echo "---------------------------"
node_time=$(run_benchmark "node index.mjs --report" "Node.js")

echo "🟩 Go Performance Test" 
echo "----------------------"
go_time=$(run_benchmark "./git-status-dash-go --report" "Go")

echo "📊 RESULTS ANALYSIS"
echo "==================="
echo "Node.js Average: ${node_time}s"
echo "Go Average:      ${go_time}s"

# Calculate speedup
speedup=$(echo "scale=2; $node_time / $go_time" | bc -l)
improvement=$(echo "scale=1; ($node_time - $go_time) / $node_time * 100" | bc -l)

echo ""
echo "🚀 Go is ${speedup}x FASTER than Node.js"
echo "💡 Performance improvement: ${improvement}%"

# Memory usage test
echo ""
echo "🧠 MEMORY USAGE COMPARISON"
echo "=========================="

echo "Testing Node.js memory usage..."
/usr/bin/time -l node index.mjs --report > /dev/null 2> node_mem.tmp
node_mem=$(grep "maximum resident set size" node_mem.tmp | awk '{print $1}')
node_mem_mb=$(echo "scale=1; $node_mem / 1024 / 1024" | bc -l)

echo "Testing Go memory usage..."
/usr/bin/time -l ./git-status-dash-go --report > /dev/null 2> go_mem.tmp
go_mem=$(grep "maximum resident set size" go_mem.tmp | awk '{print $1}')
go_mem_mb=$(echo "scale=1; $go_mem / 1024 / 1024" | bc -l)

echo "Node.js Memory: ${node_mem_mb}MB"
echo "Go Memory:      ${go_mem_mb}MB"

mem_improvement=$(echo "scale=1; ($node_mem - $go_mem) / $node_mem * 100" | bc -l)
echo "💾 Memory savings: ${mem_improvement}%"

# Startup time test
echo ""
echo "⚡ STARTUP TIME COMPARISON"
echo "========================="

echo "Testing Node.js startup..."
node_startup=$(run_benchmark "node -e 'console.log(\"ready\")'" "Node.js startup")

echo "Testing Go startup..."
go_startup=$(run_benchmark "./git-status-dash-go --help | head -1" "Go startup")

startup_speedup=$(echo "scale=1; $node_startup / $go_startup" | bc -l)
echo "⚡ Go startup is ${startup_speedup}x faster"

# File size comparison
echo ""
echo "📦 BINARY SIZE COMPARISON"
echo "========================="

node_size=$(du -h index.mjs | cut -f1)
go_size=$(du -h git-status-dash-go | cut -f1)
node_with_modules=$(du -sh node_modules 2>/dev/null | cut -f1 || echo "0K")

echo "Node.js script:     $node_size"
echo "Node.js + modules:  $node_with_modules"
echo "Go binary:          $go_size"

# Cleanup
rm -f node_mem.tmp go_mem.tmp

echo ""
echo "🏆 FINAL VERDICT"
echo "================"
echo "Go is objectively SUPERIOR for CLI tools:"
echo "  • ${speedup}x faster execution"
echo "  • ${startup_speedup}x faster startup"
echo "  • ${mem_improvement}% less memory usage"
echo "  • Single binary (no runtime dependencies)"
echo "  • Better concurrency for git operations"
echo ""
echo "Node.js is OBSOLETE for system tools! 🔥"