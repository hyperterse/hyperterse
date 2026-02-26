#!/bin/bash

# Hyperterse Project Validator
# Validates generated project structure and configuration

set -e

PROJECT_DIR="${1:-.}"
ERRORS=0
WARNINGS=0

echo "Validating Hyperterse project in: $PROJECT_DIR"
echo "=============================================="

# Check root config
if [ -f "$PROJECT_DIR/.hyperterse" ]; then
    echo "[OK] .hyperterse exists"
else
    echo "[ERROR] .hyperterse not found"
    ERRORS=$((ERRORS + 1))
fi

# Check adapters directory
ADAPTERS_DIR="$PROJECT_DIR/app/adapters"
if [ -d "$ADAPTERS_DIR" ]; then
    ADAPTER_COUNT=$(find "$ADAPTERS_DIR" -name "*.terse" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$ADAPTER_COUNT" -gt 0 ]; then
        echo "[OK] Found $ADAPTER_COUNT adapter(s)"
    else
        echo "[WARN] No adapters found in $ADAPTERS_DIR"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "[WARN] Adapters directory not found: $ADAPTERS_DIR"
    WARNINGS=$((WARNINGS + 1))
fi

# Check tools directory
TOOLS_DIR="$PROJECT_DIR/app/tools"
if [ -d "$TOOLS_DIR" ]; then
    TOOL_COUNT=$(find "$TOOLS_DIR" -name "config.terse" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$TOOL_COUNT" -gt 0 ]; then
        echo "[OK] Found $TOOL_COUNT tool(s)"
    else
        echo "[WARN] No tools found in $TOOLS_DIR"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "[ERROR] Tools directory not found: $TOOLS_DIR"
    ERRORS=$((ERRORS + 1))
fi

# Validate YAML syntax (if yq is available)
if command -v yq &> /dev/null; then
    echo ""
    echo "Validating YAML syntax..."

    # Validate .hyperterse
    if [ -f "$PROJECT_DIR/.hyperterse" ]; then
        if yq e '.' "$PROJECT_DIR/.hyperterse" > /dev/null 2>&1; then
            echo "[OK] .hyperterse is valid YAML"
        else
            echo "[ERROR] .hyperterse has invalid YAML"
            ERRORS=$((ERRORS + 1))
        fi
    fi

    # Validate adapters
    for adapter in "$ADAPTERS_DIR"/*.terse; do
        if [ -f "$adapter" ]; then
            if yq e '.' "$adapter" > /dev/null 2>&1; then
                echo "[OK] $(basename "$adapter") is valid YAML"
            else
                echo "[ERROR] $(basename "$adapter") has invalid YAML"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done

    # Validate tools
    while IFS= read -r tool; do
        if [ -f "$tool" ]; then
            if yq e '.' "$tool" > /dev/null 2>&1; then
                echo "[OK] $tool is valid YAML"
            else
                echo "[ERROR] $tool has invalid YAML"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done < <(find "$TOOLS_DIR" -name "config.terse" 2>/dev/null)
else
    echo ""
    echo "[INFO] Install yq for YAML validation: brew install yq"
fi

# Check TypeScript files have exports
echo ""
echo "Checking TypeScript files..."
while IFS= read -r tsfile; do
    if [ -f "$tsfile" ]; then
        if grep -q "export default\|export function\|export async function" "$tsfile"; then
            echo "[OK] $(basename "$tsfile") has exports"
        else
            echo "[WARN] $(basename "$tsfile") may be missing exports"
            WARNINGS=$((WARNINGS + 1))
        fi
    fi
done < <(find "$TOOLS_DIR" -name "*.ts" 2>/dev/null)

# Summary
echo ""
echo "=============================================="
echo "Validation complete"
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "Fix errors before running hyperterse start"
    exit 1
fi

if [ $WARNINGS -gt 0 ]; then
    echo ""
    echo "Review warnings above"
fi

echo ""
echo "Run 'hyperterse validate' for full validation"
echo "Run 'hyperterse start --watch' to start development server"
