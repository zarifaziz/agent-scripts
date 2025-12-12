# Build & Test

## Start Service
```bash
make local  # runs on port 50069
```

## Test with grpcurl

### ValidateLatexAndDelimiters
```bash
grpcurl -plaintext \
  -import-path ./proto \
  -proto v1/latex_service.proto \
  -d '{"text": "What is $7$ in $2,735$?"}' \
  localhost:50069 v1.latex.LatexService/ValidateLatexAndDelimiters
```

### ValidateLatexExpression
```bash
grpcurl -plaintext \
  -import-path ./proto \
  -proto v1/latex_service.proto \
  -d '{"latex": "x^2 + 2x + 1"}' \
  localhost:50069 v1.latex.LatexService/ValidateLatexExpression
```

### CompareLatexExpressions
```bash
grpcurl -plaintext \
  -import-path ./proto \
  -proto v1/latex_service.proto \
  -d '{"left": "x^2", "right": "x*x", "simplify": true}' \
  localhost:50069 v1.latex.LatexService/CompareLatexExpressions
```

## Proto Definition
See `proto/v1/latex_service.proto` for full API spec.
