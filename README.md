# gocrc

A lightweight, type-safe Go library for handling common concurrency patterns in daily workflows using Generics.

[中文文档](./README_zh.md)

## Features

- **Generic Support**: Type-safe workers and results using Go Generics.
- **Race**: Multiple workers compete for the same task. The first one to finish (success or failure) wins, and all others are cancelled immediately.
- **No Race**: Multiple workers work together on different parts of a large task. The program waits for ALL workers to complete.
- **Detailed Results**: Access the `Value`, `Err`, and `Index` of every worker.
- **Context-Aware**: Full support for `context.Context` cancellation and timeouts.
- **Structured Error Handling**: Collects multiple errors into a `MultiError`.

## Installation

```bash
go get github.com/scott-x/gocrc
```

## Usage

### 1. Race Pattern
Use `Race` when you have multiple ways to achieve the same result (e.g., multi-source downloads) and only need the first one.

```go
func main() {
    ctx := context.Background()
    
    // Workers return a string in this example
    w1 := func(ctx context.Context) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Result from A", nil
    }
    w2 := func(ctx context.Context) (string, error) {
        time.Sleep(200 * time.Millisecond)
        return "Result from B", nil
    }

    res, err := gocrc.Race(ctx, w1, w2)
    if err != nil {
        fmt.Printf("First worker failed: %v\n", err)
    } else {
        fmt.Printf("Winner Index: %d, Value: %s\n", res.Index, res.Value)
    }
}
```

### 2. No Race Pattern
Use `NoRace` when you need to partition a task and wait for all parts to finish.

```go
func main() {
    ctx := context.Background()

    results, err := gocrc.NoRace(ctx,
        func(ctx context.Context) (int, error) { return 100, nil },
        func(ctx context.Context) (int, error) { return 200, errors.New("failed") },
    )

    for _, r := range results {
        fmt.Printf("Worker %d: Value=%v, Error=%v\n", r.Index, r.Value, r.Err)
    }
}
```

### 3. Advanced: Task Grouping (A, B, C Scenario)
If you have multiple groups of tasks (A, B, C) and want to ensure the program only exits after *everything* is finished:

```go
func main() {
    ctx := context.Background()

    // Wait for all groups to finish
    _, err := gocrc.NoRace(ctx,
        func(ctx context.Context) (any, error) {
            return gocrc.NoRace(ctx, workersA...)
        },
        func(ctx context.Context) (any, error) {
            return gocrc.NoRace(ctx, workersB...)
        },
        func(ctx context.Context) (any, error) {
            return gocrc.NoRace(ctx, workersC...)
        },
    )
    
    if err != nil {
        fmt.Println("Some tasks failed, but all workers have finished or stopped.")
    }
}
```

## License
MIT
