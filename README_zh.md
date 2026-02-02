# gocrc

一个轻量级、类型安全的 Go 语言并发模式处理库，使用 Generics（泛型）简化日常工作中的并发任务调度。

[English README](./README.md)

## 特性

- **泛型支持**：通过 Go 泛型提供类型安全的 Worker 和返回结果。
- **竞速模式 (Race)**：多个 Worker 处理同一个任务。第一个完成的（无论成功或失败）胜出，其他所有 Worker 会立即被取消以节省资源。
- **协同模式 (No Race)**：多个 Worker 共同完成一个大任务的不同部分。程序会等待**所有** Worker 全部完成后才继续。
- **详尽的结果**：可以访问每个 Worker 的 `Value`（返回值）、`Err`（错误）和 `Index`（原始索引）。
- **上下文感知 (Context-Aware)**：完整支持 `context.Context` 的取消和超时机制。
- **结构化错误处理**：通过 `MultiError` 收集并返回多个并发任务中发生的详细错误。

## 安装

```bash
go get github.com/scott-x/gocrc
```

## 使用示例

### 1. 竞速模式 (Race)
当你有多条路径可以达成同一个目标（如：多镜像下载、多机房查询），且只需要最快的结果时使用。

```go
func main() {
    ctx := context.Background()
    
    // 在这个例子中，Worker 返回 string 类型
    w1 := func(ctx context.Context) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "来自 A 的结果", nil
    }
    w2 := func(ctx context.Context) (string, error) {
        time.Sleep(200 * time.Millisecond)
        return "来自 B 的结果", nil
    }

    res, err := gocrc.Race(ctx, w1, w2)
    if err != nil {
        fmt.Printf("第一个完成的 Worker 报错了: %v\n", err)
    } else {
        fmt.Printf("胜出索引: %d, 返回值: %s\n", res.Index, res.Value)
    }
}
```

### 2. 协同模式 (No Race)
当你需要将大任务拆分，并确保所有部分都处理完成时使用。

```go
func main() {
    ctx := context.Background()

    results, err := gocrc.NoRace(ctx,
        func(ctx context.Context) (int, error) { return 100, nil },
        func(ctx context.Context) (int, error) { return 200, errors.New("任务失败") },
    )

    // 即使有错误，你也可以看到所有 Worker 的执行细节
    for _, r := range results {
        fmt.Printf("Worker %d: 返回值=%v, 错误=%v\n", r.Index, r.Value, r.Err)
    }
}
```

### 3. 高阶用法：多任务组同步 (A, B, C 场景)
如果您有多个任务组（A, B, C），且希望程序在**所有**任务的所有 Worker 都干完活后才退出：

```go
func main() {
    ctx := context.Background()

    // 等待所有组 (A, B, C) 完成
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
        fmt.Println("部分任务报错了，但所有 Worker 都已经运行结束或被停止。")
    }
}
```

## 开源协议
MIT
