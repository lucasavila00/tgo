# TGo

`!` returns the same error if a call fails.

```text
func Parse(text string) (int, error) {
    value := strconv.Atoi(text)!
    return value, nil
}
```

Equivalent Go:

```go
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return 0, err
    }
    return value, nil
}
```

`!!` adds the call name to the error. `errors.Is` and `errors.As` can still
find the original error.

```text
func Parse(text string) (int, error) {
    value := strconv.Atoi(text)!!
    return value, nil
}
```

Equivalent Go:

```go
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return 0, fmt.Errorf("strconv.Atoi: %w", err)
    }
    return value, nil
}
```

An `if` can supply a value. A branch can run statements before its final value.

```text
func Label(active bool) string {
    return if active {
        fmt.Println("active")
        "on"
    } else {
        "off"
    }
}
```

Equivalent Go:

```go
func Label(active bool) string {
    if active {
        fmt.Println("active")
        return "on"
    } else {
        return "off"
    }
}
```

A `switch` can supply a value.

```text
func Label(code int) string {
    return switch code {
    case 200:
        "ok"
    case 404:
        "missing"
    default:
        "error"
    }
}
```

Equivalent Go:

```go
func Label(code int) string {
    switch code {
    case 200:
        return "ok"
    case 404:
        return "missing"
    default:
        return "error"
    }
}
```

A list comprehension builds a slice. An `if` filters the input.

```text
func PositiveSquares(numbers []int) []int {
    return []int{n * n for _, n := range numbers if n > 0}
}
```

Equivalent Go:

```go
func PositiveSquares(numbers []int) []int {
    result := []int{}
    for _, n := range numbers {
        if n > 0 {
            result = append(result, n * n)
        }
    }
    return result
}
```

A map comprehension supplies a key and a value for each input.

```text
func ByID(users []User) map[int]User {
    return map[int]User{user.ID: user for _, user := range users}
}
```

Equivalent Go:

```go
func ByID(users []User) map[int]User {
    result := map[int]User{}
    for _, user := range users {
        result[user.ID] = user
    }
    return result
}
```

Short returns omit zero values on failure and the nil error on success.

```text
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return .. err
    }
    return value ..
}
```

Equivalent Go:

```go
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return 0, err
    }
    return value, nil
}
```
