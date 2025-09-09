#!/bin/bash
# scripts/principle-check.sh - Validates architectural principles for FlowGraph

set -e

echo "🔍 Checking Architectural Principles for FlowGraph..."

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track violations
VIOLATIONS=0

# KISS: Check function complexity
echo -e "${YELLOW}Checking KISS (complexity)...${NC}"
if command -v gocyclo >/dev/null 2>&1; then
    complexity=$(gocyclo -over 10 . 2>/dev/null | wc -l | tr -d ' ')
    if [ "$complexity" -gt 0 ]; then
        echo -e "${RED}❌ KISS violation: $complexity functions with complexity >10 found${NC}"
        gocyclo -over 10 .
        VIOLATIONS=$((VIOLATIONS + 1))
    else
        echo -e "${GREEN}✅ KISS: All functions have complexity ≤10${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  gocyclo not installed, skipping complexity check${NC}"
fi

# KISS: Check function length
echo -e "${YELLOW}Checking KISS (function length)...${NC}"
long_functions=$(grep -r "^func " --include="*.go" . | while read -r line; do
    file=$(echo "$line" | cut -d: -f1)
    if [ -f "$file" ]; then
        func_line=$(echo "$line" | cut -d: -f2)
        # Count lines until next function or end of file
        lines=$(awk -v start="$func_line" 'NR >= start && /^func / && NR > start {print NR-start; exit} END {if (NR >= start) print NR-start+1}' "$file")
        if [ "${lines:-0}" -gt 50 ]; then
            echo "$file:$func_line - Function has $lines lines (>50)"
        fi
    fi
done)

if [ -n "$long_functions" ]; then
    echo -e "${RED}❌ KISS violation: Functions with >50 lines found:${NC}"
    echo "$long_functions"
    VIOLATIONS=$((VIOLATIONS + 1))
else
    echo -e "${GREEN}✅ KISS: All functions have ≤50 lines${NC}"
fi

# YAGNI: Check for unused code
echo -e "${YELLOW}Checking YAGNI (unused code)...${NC}"
if command -v staticcheck >/dev/null 2>&1; then
    unused=$(staticcheck -checks="U*" ./... 2>&1 | grep -v "no Go files" || true)
    if [ -n "$unused" ]; then
        echo -e "${RED}❌ YAGNI violation: Unused code found:${NC}"
        echo "$unused"
        VIOLATIONS=$((VIOLATIONS + 1))
    else
        echo -e "${GREEN}✅ YAGNI: No unused code found${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  staticcheck not installed, skipping unused code check${NC}"
fi

# SOLID: Check interface size (ISP - Interface Segregation Principle)
echo -e "${YELLOW}Checking SOLID (interface segregation)...${NC}"
large_interfaces=$(grep -r "type .* interface" --include="*.go" . | while read -r line; do
    file=$(echo "$line" | cut -d: -f1)
    interface_line=$(echo "$line" | cut -d: -f2)
    if [ -f "$file" ]; then
        # Count methods in interface (simple heuristic)
        methods=$(awk -v start="$interface_line" 'NR >= start && /^}/ {exit} NR > start && /^[[:space:]]*[A-Z].*\(.*\)/ {count++} END {print count+0}' "$file")
        if [ "${methods:-0}" -gt 5 ]; then
            echo "$file:$interface_line - Interface has $methods methods (>5)"
        fi
    fi
done)

if [ -n "$large_interfaces" ]; then
    echo -e "${RED}❌ ISP violation: Interfaces with >5 methods found:${NC}"
    echo "$large_interfaces"
    VIOLATIONS=$((VIOLATIONS + 1))
else
    echo -e "${GREEN}✅ SOLID: All interfaces have ≤5 methods${NC}"
fi

# DRY: Check for duplicate code
echo -e "${YELLOW}Checking DRY (code duplication)...${NC}"
if command -v dupl >/dev/null 2>&1; then
    dupl -threshold 50 . 2>/dev/null > dupl.txt || true
    if [ -s dupl.txt ]; then
        echo -e "${RED}❌ DRY violation: Duplicate code found:${NC}"
        cat dupl.txt
        VIOLATIONS=$((VIOLATIONS + 1))
    else
        echo -e "${GREEN}✅ DRY: No significant code duplication found${NC}"
    fi
    rm -f dupl.txt
else
    echo -e "${YELLOW}⚠️  dupl not installed, skipping duplication check${NC}"
fi

# SOLID: Check for dependency inversion violations in core domain
echo -e "${YELLOW}Checking SOLID (dependency inversion)...${NC}"
core_violations=$(find internal/core -name "*.go" -exec grep -l "import.*github.com" {} \; 2>/dev/null || true)
if [ -n "$core_violations" ]; then
    echo -e "${RED}❌ DIP violation: Core domain has external dependencies:${NC}"
    echo "$core_violations"
    VIOLATIONS=$((VIOLATIONS + 1))
else
    echo -e "${GREEN}✅ SOLID: Core domain has no external dependencies${NC}"
fi

# Summary
echo ""
echo "📊 Architectural Principles Summary:"
if [ $VIOLATIONS -eq 0 ]; then
    echo -e "${GREEN}✅ All architectural principles validated successfully!${NC}"
    echo -e "${GREEN}   - KISS: Simple functions and low complexity${NC}"
    echo -e "${GREEN}   - YAGNI: No unused code${NC}"
    echo -e "${GREEN}   - SOLID: Proper interfaces and dependencies${NC}"
    echo -e "${GREEN}   - DRY: No code duplication${NC}"
    exit 0
else
    echo -e "${RED}❌ Found $VIOLATIONS architectural principle violations${NC}"
    echo -e "${RED}   Please fix the violations above before proceeding${NC}"
    exit 1
fi
