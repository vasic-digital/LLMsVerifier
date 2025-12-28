#!/bin/bash
set -e

echo "=== CLEAN SLATE VERIFICATION ==="
echo "Step 1: Clean up..."
cd llm-verifier
rm -f llm-verifier.db

echo "Step 2: Build binary..."
go build -o llm-verifier-bin cmd/main.go

echo "Step 3: Run verification..."
timeout 120 ./llm-verifier-bin --config config_minimal.yaml 2>&1 | tee ../verification_output.log

echo "Step 4: Check results..."
if [ -f llm-verifier.db ]; then
    echo "✅ Database created"
    sqlite3 llm-verifier.db "SELECT COUNT(*) FROM models;"
    sqlite3 llm-verifier.db "SELECT COUNT(*) FROM verification_results;"
else
    echo "❌ No database created"
fi

echo "Done!"
