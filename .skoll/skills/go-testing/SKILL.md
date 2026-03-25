---
id: skill-go-testing
name: Go Testing Expert
version: 1.0.0
description: Expert in writing comprehensive tests for Go applications using standard library and testify
trigger: go test|testing|testify|assert|require|suite
allowed_tools:
  - read
  - write
  - edit
  - glob
  - grep
  - bash
tags:
  - go
  - testing
  - tdd
  - go-testing
categories:
  - backend
  - testing
progressive_disclosure:
  level: 1
  summary: Go testing with standard library and testify
---

# Go Testing Expert

You are an expert in writing comprehensive tests for Go applications.

## Core Testing Patterns

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name    string
        a, b   int
        expect int
    }{
        {"simple addition", 1, 2, 3},
        {"with zero", 0, 5, 5},
        {"negative numbers", -1, -1, -2},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expect {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expect)
            }
        })
    }
}
```

### Using testify/assert

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWithAssert(t *testing.T) {
    result := Add(2, 3)
    assert.Equal(t, 5, result, "Add should return correct sum")
}

func TestWithRequire(t *testing.T) {
    result, err := Divide(10, 2)
    require.NoError(t, err, "Divide should not error")
    assert.Equal(t, 5, result)
}
```

### Test Suites with suite

```go
import "github.com/stretchr/testify/suite"

type MathSuite struct {
    suite.Suite
    Calculator *Calculator
}

func (s *MathSuite) SetupSuite() {
    s.Calculator = NewCalculator()
}

func (s *MathSuite) TestAdd() {
    assert.Equal(s.T(), 5, s.Calculator.Add(2, 3))
}

func TestMathSuite(t *testing.T) {
    suite.Run(t, new(MathSuite))
}
```

### Mocking Interfaces

```go
type UserRepository interface {
    FindByID(id int) (*User, error)
}

type MockUserRepository struct {
    users map[int]*User
}

func (m *MockUserRepository) FindByID(id int) (*User, error) {
    if user, ok := m.users[id]; ok {
        return user, nil
    }
    return nil, ErrUserNotFound
}

func TestGetUser(t *testing.T) {
    mockRepo := &MockUserRepository{
        users: map[int]*User{1: {ID: 1, Name: "John"}},
    }
    
    svc := NewUserService(mockRepo)
    user, err := svc.GetUser(1)
    
    require.NoError(t, err)
    assert.Equal(t, "John", user.Name)
}
```

### Testing HTTP Handlers

```go
import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()
    
    HealthHandler(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var resp map[string]string
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "healthy", resp["status"])
}
```

## Best Practices

1. **Name tests descriptively**: `TestAdd_WithPositiveNumbers_ReturnsSum`
2. **Use subtests**: Group related tests with `t.Run`
3. **Test edge cases**: Zero, negative, empty, nil
4. **Use table-driven tests**: Efficient for multiple test cases
5. **Prefer `require` over `assert`**: Fails fast on setup errors
6. **Keep tests independent**: No shared mutable state
7. **Test errors explicitly**: `assert.ErrorIs(t, err, ErrNotFound)`
