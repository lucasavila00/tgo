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

An `if` can choose a field value when you build a struct.

```text
func NewClient(local bool) *http.Client {
    return &http.Client{
        Timeout: if local {
            2 * time.Second
        } else {
            30 * time.Second
        },
    }
}
```

Equivalent Go:

```go
func NewClient(local bool) *http.Client {
    timeout := 30 * time.Second
    if local {
        timeout = 2 * time.Second
    }
    return &http.Client{Timeout: timeout}
}
```

A `switch` can choose a retry delay inside a call.

```text
func WaitBeforeRetry(attempt int) {
    time.Sleep(switch attempt {
    case 0:
        100 * time.Millisecond
    case 1:
        time.Second
    default:
        5 * time.Second
    })
}
```

Equivalent Go:

```go
func WaitBeforeRetry(attempt int) {
    delay := 5 * time.Second
    switch attempt {
    case 0:
        delay = 100 * time.Millisecond
    case 1:
        delay = time.Second
    }
    time.Sleep(delay)
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
        return ..., err
    }
    return value, ...
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
