# Go Testing Reference Card

## Testing Functions

| Function | Package | Use Case |
|----------|---------|----------|
| `testing.T` | stdlib | Basic testing |
| `testing.B` | stdlib | Benchmarking |
| `assert.Equal` | testify | Soft assertions |
| `require.Equal` | testify | Hard assertions |
| `suite.Run` | testify | Test suites |

## Common Assertions

```go
// Equality
assert.Equal(t, expected, actual)
assert.NotEqual(t, expected, actual)

// Nil/Empty
assert.Nil(t, obj)
assert.NotNil(t, obj)
assert.Empty(t, slice)

// Errors
assert.NoError(t, err)
assert.Error(t, err)
assert.ErrorIs(t, err, target)

// Slices/Maps
assert.Contains(t, slice, item)
assert.Len(t, slice, 0)

// Booleans
assert.True(t, cond)
assert.False(t, cond)
```

## Benchmarking

```go
func BenchmarkFib(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Fib(10)
    }
}

// Run with: go test -bench=BenchmarkFib -benchmem
```

## Coverage

```bash
# Get coverage
go test -cover

# Get detailed coverage
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```
