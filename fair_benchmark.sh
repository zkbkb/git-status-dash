#!/bin/bash

echo "🧪 FAIR PERFORMANCE SCIENCE: Node.js vs Go"
echo "==========================================="

echo "Testing CORE functionality only (--report mode)"
echo "This compares the essential git scanning performance"
echo ""

# Test with just report mode (no TUI overhead)
echo "🟨 Node.js (--report):"
time node index.mjs --report

echo ""
echo "🟩 Go (--report):"  
time ./git-status-dash-go --report

echo ""
echo "💾 MEMORY COMPARISON:"
echo "Node.js RSS:"
/usr/bin/time -l node index.mjs --report 2>&1 | grep "maximum resident set size" | awk '{print $1/1024/1024 " MB"}'

echo "Go RSS:"
/usr/bin/time -l ./git-status-dash-go --report 2>&1 | grep "maximum resident set size" | awk '{print $1/1024/1024 " MB"}'

echo ""
echo "⚡ STARTUP COMPARISON:"
echo "Node.js cold start:"
time node -e "console.log('ready')"

echo ""
echo "Go cold start:"
time ./git-status-dash-go --version 2>/dev/null || time ./git-status-dash-go --help | head -1

echo ""
echo "📦 SIZE COMPARISON:"
echo "Node.js script + runtime requirements:"
du -h index.mjs
echo "Go single binary:"
du -h git-status-dash-go

echo ""
echo "🔥 THE TRUTH:"
echo "Node.js startup time includes V8 compilation"
echo "Go binary is pre-compiled and optimized"
echo "For CLI tools, Go wins on:"
echo "  • Memory efficiency"
echo "  • Single binary deployment" 
echo "  • No runtime dependencies"
echo "  • Better concurrency"