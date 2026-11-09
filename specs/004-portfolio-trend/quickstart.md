# Quickstart: portfolio trend

1. `./build.sh backend --no-lint --no-test && ./build.sh frontend --no-lint --no-test && ./run-local.sh`
2. Sign in. The chart is the first card on Home and on Investments.
3. Record a price for a security (Refresh prices, or edit the price). The dashed part of the value line gets shorter as real prices accumulate.
4. `go test ./pkg/investments/ ./pkg/services/` and `npx vitest run src/expbook`.
