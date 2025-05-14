#!/bin/bash

# Change to the correct directory
cd "$(dirname "$0")"

echo "Discovering fuzz tests in pkg/assert package..."
FUZZ_TESTS=$(grep -r "^func Fuzz" go/pkg/assert/ | sed -E 's/^.*func (Fuzz[a-zA-Z0-9_]+).*/\1/')

# Display what tests were found
echo "Found the following fuzz tests:"
echo "$FUZZ_TESTS" | tr ' ' '\n'
echo ""

# Count the tests for progress display
TOTAL_TESTS=$(echo "$FUZZ_TESTS" | wc -w | tr -d ' ')
CURRENT=1

# Run each test for 1 minute
for test in $FUZZ_TESTS; do
  echo "[$CURRENT/$TOTAL_TESTS] Running $test (1 minute)..."
  echo "--------------------------------------------------------"
  cd go && go test -fuzz="$test" -fuzztime=1m ./pkg/assert && cd ..
  echo "--------------------------------------------------------"
  echo ""
  
  CURRENT=$((CURRENT + 1))
done

echo "All fuzzing tests completed!"