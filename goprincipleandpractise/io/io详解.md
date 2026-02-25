# Go 标准库 io/bufio 详解

`io` 包定义了 Go 语言 I/O 操作的核心抽象：`Reader` 和 `Writer` 接口。整个标准库——从文件、网络到 HTTP、JSON 编解码——都建立在这两个接口之上。`bufio` 在其上增加缓冲层，将高频小粒度读写合并为低频大块系统调用。本文从源码层面剖析这套接口体系的设计哲学、内部实现与生产实践。

---

## 1 io.Reader 和 io.Writer — Go I/O 的基石

### 1.1 为什么 Go 的 I/O 只需要两个方法

Java 的 I/O 体系有 `InputStream`、`OutputStream`、`BufferedInputStream`、`DataInputStream`... 一棵深度继承树。Go 走了完全不同的路——只定义两个方法：

```go
// io/io.go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

一个方法，一个接口。所有 I/O 相关的能力（缓冲、限流、多路复用、压缩、加密）都通过**组合**这两个接口实现，而非继承。

这意味着：
- 任何实现了 `Read` 方法的类型都是 `Reader`，无需声明
- 文件、网络连接、内存缓冲区、HTTP Body、gzip 流——都是 `Reader`
- 你可以写一个函数接受 `io.Reader` 参数，它就能处理以上所有来源

### 1.2 Read 方法的契约细节

`Read` 方法的行为规范在源码注释中有精确定义，这是大多数 I/O bug 的根源所在：

```go
// Read reads up to len(p) bytes into p. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered.
//
// Even if Read returns n < len(p), it may use all of p as scratch space.
// If some data is available but not len(p) bytes, Read conventionally
// returns what is available instead of waiting for more.
//
// Callers should always process the n > 0 bytes returned before
// considering the error err.
```

关键契约：

| 规则 | 含义 |
|------|------|
| `n` 可能小于 `len(p)` | 即使没有错误，也可能只读到部分数据 |
| `n > 0` 和 `err != nil` 可同时发生 | **必须先处理 n 字节，再检查 err** |
| `err == io.EOF` 表示读完 | EOF 不是错误，是正常的流结束信号 |
| 不得保留 `p` | 实现不能把 `p` 存起来在 Read 返回后使用 |

正确的读取循环：

```go
buf := make([]byte, 4096)
for {
    n, err := r.Read(buf)
    if n > 0 {
        // 先处理已读取的数据
        process(buf[:n])
    }
    if err != nil {
        if err == io.EOF {
            break // 正常结束
        }
        return err // 真正的错误
    }
}
```

**反面教材**——先检查 err 再处理 n：

```go
// 错误！最后一批数据可能丢失
n, err := r.Read(buf)
if err != nil {
    return err  // 即使 n > 0，也丢弃了数据
}
process(buf[:n])
```

### 1.3 Write 方法的契约细节

```go
// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
```

Write 的契约比 Read 更严格：

| 规则 | 含义 |
|------|------|
| `n < len(p)` 必须返回 error | 短写一定是错误 |
| 不得修改 `p` | 即使临时修改也不行 |
| 不得保留 `p` | 和 Read 一样 |

### 1.4 核心接口全景

Go 的 `io` 包通过小接口组合定义了完整的 I/O 能力矩阵：

| 接口 | 方法 | 用途 | 典型实现 |
|------|------|------|---------|
| `Reader` | `Read` | 字节源 | os.File, bytes.Buffer, strings.Reader, net.Conn |
| `Writer` | `Write` | 字节目标 | os.File, bytes.Buffer, net.Conn, http.ResponseWriter |
| `Closer` | `Close` | 释放资源 | os.File, net.Conn, http.Response.Body |
| `Seeker` | `Seek` | 随机定位 | os.File, bytes.Reader, strings.Reader |
| `ReadWriter` | `Read+Write` | 双向流 | os.File, bytes.Buffer, net.Conn |
| `ReadCloser` | `Read+Close` | 可关闭的读取源 | http.Response.Body, os.File |
| `WriteCloser` | `Write+Close` | 可关闭的写入目标 | os.File, gzip.Writer |
| `ReadSeeker` | `Read+Seek` | 可定位的读取源 | os.File, bytes.Reader |
| `ReaderAt` | `ReadAt` | 并发安全的随机读 | os.File, bytes.Reader |
| `WriterTo` | `WriteTo` | 优化的批量写出 | bytes.Buffer, strings.Reader |
| `ReaderFrom` | `ReadFrom` | 优化的批量读入 | bytes.Buffer, bufio.Writer |

---

## 2 接口组合的设计哲学

### 2.1 小接口组合 vs 大接口继承

Go 没有 `Stream` 基类，而是定义了约 16 个小接口。为什么？

1. **解耦**：一个函数只依赖它需要的能力。`io.Copy(dst Writer, src Reader)` 不关心 src 能不能 Seek。
2. **可测试**：测试时用 `strings.NewReader("test data")` 就能满足 `io.Reader`，不需要创建文件。
3. **可组合**：`io.LimitReader` 接受任何 `Reader`，返回一个新 `Reader`——装饰器模式，零继承。

### 2.2 编译期接口检查

Go 的接口满足是隐式的，但可以用编译期断言确保类型实现了预期接口：

```go
// 确认 *os.File 实现了这些接口
var _ io.ReadWriteCloser = (*os.File)(nil)
var _ io.ReadSeeker      = (*os.File)(nil)
var _ io.ReaderAt         = (*os.File)(nil)
var _ io.WriterAt         = (*os.File)(nil)
```

### 2.3 WriterTo 和 ReaderFrom — 优化接口

这两个接口是性能优化的关键。以 `io.Copy` 为例，它的内部实现会依次检查：

```go
// io/io.go copyBuffer() 简化版
func copyBuffer(dst Writer, src Reader, buf []byte) (int64, error) {
    // 优化路径 1：src 能直接写到 dst
    if wt, ok := src.(WriterTo); ok {
        return wt.WriteTo(dst)
    }
    // 优化路径 2：dst 能直接从 src 读
    if rt, ok := dst.(ReaderFrom); ok {
        return rt.ReadFrom(src)
    }
    // 兜底：用中间缓冲区搬运
    if buf == nil {
        buf = make([]byte, 32*1024) // 默认 32KB
    }
    // ... read/write 循环
}
```

当 `src` 是 `*bytes.Buffer`（实现了 `WriterTo`）时，`io.Copy` 直接调用 `buf.WriteTo(dst)`，跳过中间缓冲区，减少一次内存拷贝。

---

## 3 标准库中的 Reader/Writer 实现

### 3.1 strings.Reader — 只读字符串流

`strings.NewReader` 把一个字符串包装为 `Reader`，无需拷贝：

```go
// strings/reader.go
type Reader struct {
    s        string
    i        int64 // 当前读取位置
    prevRune int   // 上一个 ReadRune 的位置，用于 UnreadRune
}
```

它实现了 `Reader`、`ReaderAt`、`Seeker`、`WriterTo` 四个接口。其中 `WriteTo` 直接调用 `io.WriteString(w, r.s[r.i:])`，避免了中间缓冲区。

### 3.2 bytes.Buffer — 可增长的读写缓冲区

`bytes.Buffer` 是标准库中最常用的内存 I/O 类型，同时实现 `Reader` 和 `Writer`：

```go
// bytes/buffer.go
type Buffer struct {
    buf      []byte
    off      int    // 读取位置，&buf[off] 到 &buf[len(buf)] 是未读部分
    lastRead readOp
}
```

关键实现细节：
- **小缓冲区优化**：内部有一个 64 字节的 bootstrap 数组，小数据不触发堆分配
- **增长策略**：容量不足时调用 `grow()`，尝试通过移动数据到缓冲区头部来腾出空间，只有确实不够时才扩容（2 倍增长）
- **ReadFrom 优化**：`ReadFrom` 直接读入内部缓冲区，避免中间拷贝

### 3.3 os.File — 系统调用的薄封装

`os.File` 封装了文件描述符，其 `Read`/`Write` 直接映射到 `read(2)`/`write(2)` 系统调用。关键点：

- **ReadAt/WriteAt**：映射到 `pread(2)`/`pwrite(2)`，不影响文件偏移量，**并发安全**
- **每次 Read/Write 都是一次系统调用**：这就是为什么需要 `bufio` 缓冲

### 3.4 io.LimitReader 和 io.SectionReader

装饰器模式的经典应用：

```go
// io/io.go
type LimitedReader struct {
    R Reader // 底层 Reader
    N int64  // 剩余可读字节数
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
    if l.N <= 0 {
        return 0, EOF
    }
    if int64(len(p)) > l.N {
        p = p[0:l.N]
    }
    n, err = l.R.Read(p)
    l.N -= int64(n)
    return
}
```

`SectionReader` 更强大——在底层 `ReaderAt` 上定义一个窗口 `[off, off+n)`，实现了 `Reader`、`Seeker`、`ReaderAt`。典型用途：读取 ZIP 文件中单个文件条目。

---

## 4 io 包的工具函数

### 4.1 io.Copy / io.CopyN / io.CopyBuffer

`io.Copy` 是整个 I/O 体系的枢纽函数。它的三层优化策略在第 2.3 节已分析。补充几个实践要点：

**CopyN — 限量拷贝**：

```go
// 只从 HTTP Body 读取前 1MB，防止客户端上传超大文件
n, err := io.CopyN(file, req.Body, 1<<20)
```

**CopyBuffer — 自定义缓冲区**：

```go
// 在内存敏感的场景使用 4KB 缓冲区替代默认的 32KB
buf := make([]byte, 4096)
io.CopyBuffer(dst, src, buf)
```

### 4.2 io.TeeReader — 流式分叉

```go
// io/io.go
func TeeReader(r Reader, w Writer) Reader {
    return &teeReader{r, w}
}

type teeReader struct {
    r Reader
    w Writer
}

func (t *teeReader) Read(p []byte) (n int, err error) {
    n, err = t.r.Read(p)
    if n > 0 {
        if n, err := t.w.Write(p[:n]); err != nil {
            return n, err
        }
    }
    return
}
```

极度简单，但极度实用。典型场景——边下载边计算校验和：

```go
h := sha256.New()
tee := io.TeeReader(resp.Body, h)
io.Copy(file, tee)          // 数据写入文件
checksum := h.Sum(nil)       // 同时得到哈希
```

### 4.3 io.Pipe — 同步内存管道

`io.Pipe()` 返回一对绑定的 `PipeReader` 和 `PipeWriter`。**没有内部缓冲区**——`Write` 阻塞到 `Read` 消费了数据才返回。

```go
// io/pipe.go 核心结构
type pipe struct {
    wrMu sync.Mutex // 序列化写操作
    wrCh chan []byte // 写端 -> 读端 的数据通道
    rdCh chan int    // 读端 -> 写端 的确认通道
    once sync.Once
    done chan struct{}
    rerr onceError
    werr onceError
}
```

典型用途——将 `json.Encoder`（需要 Writer）的输出传给 `http.Post`（需要 Reader）：

```go
pr, pw := io.Pipe()
go func() {
    defer pw.Close()
    json.NewEncoder(pw).Encode(data)
}()
resp, err := http.Post(url, "application/json", pr)
```

**关键**：读写必须在不同 goroutine，否则死锁。详见 `trap/pipe-deadlock/`。

### 4.4 io.MultiReader / io.MultiWriter

**MultiReader** — 串联多个 Reader，按顺序读完一个再读下一个：

```go
header := strings.NewReader("HTTP/1.1 200 OK\r\n\r\n")
body := bytes.NewReader(bodyBytes)
r := io.MultiReader(header, body) // 先读 header，再读 body
```

**MultiWriter** — 扇出写入，同时写入多个 Writer：

```go
// 同时写入文件和标准输出
f, _ := os.Create("output.log")
w := io.MultiWriter(f, os.Stdout)
fmt.Fprintln(w, "写入到两个目标")
```

`MultiWriter.Write` 依次调用每个 Writer，任一失败则立即返回错误。

---

## 5 bufio — 缓冲 I/O 的必要性

### 5.1 为什么需要缓冲

每次 `os.File.Read` 或 `os.File.Write` 都是一次系统调用。系统调用的开销包括：
1. 用户态 → 内核态上下文切换
2. 参数校验
3. 内核缓冲区拷贝
4. 内核态 → 用户态上下文切换

一个单字节写入和一个 4KB 写入的系统调用开销几乎相同。如果每次只写 1 个字节，写 1MB 数据需要 1,048,576 次系统调用。用 `bufio.Writer`（默认 4096 字节缓冲区），只需约 256 次。

性能差异见 `performance/buffered-vs-unbuffered/`。

### 5.2 bufio.Reader 内部实现

```go
// bufio/bufio.go
type Reader struct {
    buf          []byte
    rd           io.Reader // 底层 Reader
    r, w         int       // buf 中的读写位置
    err          error
    lastByte     int
    lastRuneSize int
}
```

工作原理：

```
buf: [已读区域|未读数据|空闲空间]
      0      r        w       len(buf)
```

当调用 `Read` 时：
1. 如果 `buf[r:w]` 中有数据，直接返回（**零系统调用**）
2. 如果 `buf` 为空，调用 `fill()`：把 `buf[r:w]` 移到 `buf[0:]`，然后调用底层 `rd.Read` 填充剩余空间

关键方法对比：

| 方法 | 返回类型 | 是否分配内存 | 跨缓冲区边界 |
|------|---------|-------------|-------------|
| `ReadByte` | `byte` | 否 | 自动 fill |
| `ReadSlice(delim)` | `[]byte` (buf 的切片) | **否，引用内部 buf** | 不跨，超长返回 ErrBufferFull |
| `ReadBytes(delim)` | `[]byte` (新分配) | 是 | 自动跨界 |
| `ReadString(delim)` | `string` (新分配) | 是 | 自动跨界 |
| `Peek(n)` | `[]byte` (buf 的切片) | **否，引用内部 buf** | 不跨，超长返回 ErrBufferFull |

**注意**：`ReadSlice` 和 `Peek` 返回的是内部缓冲区的切片，下次读操作会覆盖内容。如果需要保留数据，用 `ReadBytes` 或手动 copy。

### 5.3 bufio.Writer 内部实现

```go
// bufio/bufio.go
type Writer struct {
    err error
    buf []byte
    n   int        // buf 中已缓冲的字节数
    wr  io.Writer  // 底层 Writer
}
```

`Write` 的逻辑：
1. 如果数据能放进 `buf[n:]`，直接拷贝，返回（**零系统调用**）
2. 如果缓冲区已有数据且新数据放不下，先 `Flush` 再处理
3. 如果新数据超过整个缓冲区大小，直接写入底层 Writer（跳过缓冲）

**Flush 是命脉**：`bufio.Writer` 不会在 Close 时自动 Flush（因为它没有 `Close` 方法！）。如果底层 Writer 被关闭时缓冲区还有数据，数据丢失。详见 `trap/writer-not-flushed/`。

```go
f, _ := os.Create("output.txt")
bw := bufio.NewWriter(f)
defer bw.Flush() // 必须！且必须在 f.Close() 之前
defer f.Close()
bw.WriteString("重要数据")
```

注意 `defer` 的执行顺序是 LIFO（后进先出），上面的代码会先 Flush 再 Close，这是正确的。

### 5.4 bufio.ReadWriter

简单的 Reader + Writer 组合：

```go
type ReadWriter struct {
    *Reader
    *Writer
}
```

典型用途——给网络连接加缓冲：

```go
conn, _ := net.Dial("tcp", "localhost:6379")
rw := bufio.NewReadWriter(
    bufio.NewReader(conn),
    bufio.NewWriter(conn),
)
// 写入时自动缓冲，读取时自动预读
```

---

## 6 bufio.Scanner — 结构化文本读取

### 6.1 Scanner 的设计

`bufio.Scanner` 是按"token"（词元）读取文本的高级抽象：

```go
// bufio/scan.go
type Scanner struct {
    r            io.Reader
    split        SplitFunc  // 切分函数
    maxTokenSize int        // 最大 token 大小，默认 MaxScanTokenSize (64KB)
    token        []byte     // 最近一次扫描到的 token
    buf          []byte     // 内部缓冲区
    start        int        // buf 中未处理数据的起始位置
    end          int        // buf 中未处理数据的结束位置
    err          error
    ...
}

type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)
```

标准用法：

```go
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text() // 当前行（去掉换行符）
}
if err := scanner.Err(); err != nil {
    log.Fatal(err)
}
```

### 6.2 内置 SplitFunc

| 函数 | 分隔方式 | 保留分隔符 |
|------|---------|-----------|
| `ScanLines`（默认）| `\n` 或 `\r\n` | 否 |
| `ScanWords` | 连续空白字符 | 否 |
| `ScanRunes` | UTF-8 码点 | — |
| `ScanBytes` | 单字节 | — |

`ScanLines` 的核心逻辑：

```go
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
    if i := bytes.IndexByte(data, '\n'); i >= 0 {
        // 找到换行符，返回该行（去掉 \r\n 或 \n）
        return i + 1, dropCR(data[0:i]), nil
    }
    if atEOF && len(data) > 0 {
        // 最后一行没有换行符
        return len(data), dropCR(data), nil
    }
    // 需要更多数据
    return 0, nil, nil
}
```

### 6.3 自定义 SplitFunc

按空字节分隔（处理 `find -print0` 输出）：

```go
func ScanNullTerminated(data []byte, atEOF bool) (advance int, token []byte, err error) {
    if i := bytes.IndexByte(data, 0); i >= 0 {
        return i + 1, data[:i], nil
    }
    if atEOF && len(data) > 0 {
        return len(data), data, nil
    }
    return 0, nil, nil
}

scanner := bufio.NewScanner(r)
scanner.Split(ScanNullTerminated)
```

### 6.4 缓冲区限制与调整

默认最大 token 大小是 `MaxScanTokenSize = 64 * 1024`（64KB）。当一行超过这个大小：

```go
scanner := bufio.NewScanner(r)
// 默认 64KB 限制
for scanner.Scan() { /* ... */ }
fmt.Println(scanner.Err()) // bufio.Scanner: token too long
```

解决方案——用 `Buffer()` 方法调大限制：

```go
scanner := bufio.NewScanner(r)
scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 初始 0，最大 10MB
```

**注意**：第一个参数是初始缓冲区，第二个参数是最大 token 大小。详见 `trap/scanner-buffer-too-small/`。

---

## 7 io/fs — 文件系统抽象 (Go 1.16+)

### 7.1 fs.FS 接口

Go 1.16 引入了 `io/fs` 包，将文件系统操作也纳入接口抽象：

```go
// io/fs/fs.go
type FS interface {
    Open(name string) (File, error)
}

type File interface {
    Stat() (FileInfo, error)
    Read([]byte) (int, error)
    Close() error
}
```

注意 `fs.File` 也是一个 `io.Reader` + `io.Closer`——I/O 接口体系的延伸。

### 7.2 标准实现

| 实现 | 来源 | 用途 |
|------|------|------|
| `os.DirFS(dir)` | 操作系统目录 | 将目录树包装为 FS |
| `embed.FS` | 编译期嵌入 | 静态资源打包到二进制文件 |
| `zip.Reader` | ZIP 文件 | 读取 ZIP 内容 |
| `fstest.MapFS` | 测试用 | 内存模拟文件系统 |

### 7.3 实战：用 fs.FS 实现可测试的文件操作

**生产代码**——接口依赖，不绑定 os：

```go
func LoadConfig(fsys fs.FS, name string) (*Config, error) {
    f, err := fsys.Open(name)
    if err != nil {
        return nil, fmt.Errorf("open config: %w", err)
    }
    defer f.Close()

    var cfg Config
    if err := json.NewDecoder(f).Decode(&cfg); err != nil {
        return nil, fmt.Errorf("decode config: %w", err)
    }
    return &cfg, nil
}
```

**测试代码**——用 MapFS 模拟，无需真实文件：

```go
func TestLoadConfig(t *testing.T) {
    fsys := fstest.MapFS{
        "config.json": &fstest.MapFile{
            Data: []byte(`{"port": 8080, "debug": true}`),
        },
    }
    cfg, err := LoadConfig(fsys, "config.json")
    if err != nil {
        t.Fatal(err)
    }
    if cfg.Port != 8080 {
        t.Errorf("expected port 8080, got %d", cfg.Port)
    }
}
```

这正是"accept interfaces, return structs"在文件系统层面的应用。

---

## 8 常见陷阱总结

### 8.1 bufio.Writer 未 Flush 导致数据丢失

**现象**：数据写入后文件为空或部分缺失。

**原因**：`bufio.Writer` 将数据缓存在内存中，`Flush()` 才会写入底层 Writer。如果在 Flush 前关闭了底层文件，缓冲区数据丢失。

**修复**：始终 `defer bw.Flush()`，且确保 Flush 在底层 Writer 的 Close 之前执行。

```go
f, _ := os.Create("file.txt")
bw := bufio.NewWriter(f)
defer bw.Flush()  // 先 Flush（LIFO，后注册先执行）
defer f.Close()   // 再 Close
```

详见 [trap/writer-not-flushed](trap/writer-not-flushed)。

### 8.2 Scanner 缓冲区过小导致静默截断

**现象**：用 `bufio.Scanner` 按行读取文件，部分行丢失，`Err()` 返回 `bufio.ErrTooLong`。

**原因**：默认最大 token 大小为 64KB。超长行导致 Scanner 停止扫描。

**修复**：使用 `scanner.Buffer()` 设置足够大的缓冲区。

详见 [trap/scanner-buffer-too-small](trap/scanner-buffer-too-small)。

### 8.3 io.Reader 被消费后无法重读

**现象**：第一次 `io.ReadAll` 成功，第二次返回空数据。

**原因**：大多数 Reader 是一次性的，读取游标不会自动回退。

**修复**：使用 `Seek(0, io.SeekStart)` 回退（仅限 Seeker），或将数据缓存到 `[]byte` 后用 `bytes.NewReader` 重建。


详见 [trap/reader-consumed-twice](trap/reader-consumed-twice)。

### 8.4 底层资源提前关闭导致读取失败

**现象**：在 Scanner/Decoder 还在读取时关闭了底层文件，后续读取失败。

**原因**：装饰器 Reader（Scanner、json.Decoder 等）不拥有底层资源，它们在每次读取时才调用底层 Read。

**修复**：确保底层资源的生命周期覆盖所有读取操作。用 `defer f.Close()` 在函数开头。

详见 [trap/close-before-read-complete](trap/close-before-read-complete)。

### 8.5 io.Pipe 同步误用导致死锁

**现象**：`pw.Write(data)` 永远阻塞。

**原因**：`io.Pipe` 没有内部缓冲区，Write 阻塞到 Read 消费数据。在同一个 goroutine 中先 Write 后 Read 会死锁。

**修复**：Write 和 Read 必须在不同的 goroutine 中。

详见 [trap/pipe-deadlock](trap/pipe-deadlock)。


### 8.6 strings.Reader 重用时忘记 Reset

**现象**：第二次从同一个 `*strings.Reader` 读取，得到空数据。

**原因**：Reader 内部游标已经移到末尾。

**修复**：调用 `r.Reset(s)` 或 `r.Seek(0, io.SeekStart)` 重置位置。

详见 [trap/strings-reader-reset-forgotten](trap/strings-reader-reset-forgotten)。
