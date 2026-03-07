
---
iota详解
---

# 1 核心规则

从编译器的角度看iota，其取值规则只有一条：
**iota代表了const声明块的行索引（下标从0开始）。**

const声明还有一个特点，即如果为常量指定了一个表达式，但后续的常量没有表达式，则继承上面的表达式。

根据这个规则来分析一个复杂的常量声明：

```go
const (
  bit0, mask0 = 1 << iota, 1<<iota - 1
  bit1, mask1
  _, _
  bit3, mask3
)
```
第0行表达式展开即`bit0, mask0 = 1<<0, 1<<0-1`，所以bit0=1，mask0=0；
第1行没有指定表达式，继承第一行，即`bit1, mask1 = 1<<1, 1<<1-1`，所以bit1=2，mask1=1；
第2行没有定义常量，但iota代表const声明块的行索引还是会+1，此时iota=2；
第3行没有指定表达式，继承第一行，即`bit3, mask3 = 1<<3, 1<<3-1`，所以bit3=8，mask3=7。

**总结：**
- 单个const声明块中iota从0开始取值
- 每增加一行声明，iota的取值增1，即便声明中没有使用iota也是如此
- 单行声明语句中，即便出现多个iota，iota的取值也保持不变
- **不同const块中的iota互相独立，各自从0开始**

```go
const (
    a = iota // 0
    b        // 1
)

const (
    c = iota // 0  ← 重新从0开始，与上面的const块无关
    d        // 1
)
```


# 2 典型用法

## 2.1 枚举定义

iota最常见的用途是定义枚举。Go没有enum关键字，惯用做法是通过自定义类型+iota实现：

```go
type Weekday int

const (
    Sunday    Weekday = iota // 0
    Monday                   // 1
    Tuesday                  // 2
    Wednesday                // 3
    Thursday                 // 4
    Friday                   // 5
    Saturday                 // 6
)
```

## 2.2 位掩码（Bitflag）

利用`iota`配合位移运算，可以优雅地定义位掩码，这在标准库中广泛使用：

```go
type Permission int

const (
    Read    Permission = 1 << iota // 1  (001)
    Write                          // 2  (010)
    Execute                        // 4  (100)
)

// 组合权限
const ReadWrite = Read | Write // 3 (011)

func HasPermission(p, flag Permission) bool {
    return p&flag != 0
}

func main() {
    userPerm := Read | Write
    fmt.Println(HasPermission(userPerm, Read))    // true
    fmt.Println(HasPermission(userPerm, Execute)) // false
}
```

标准库`sync.Mutex`中的状态标志就采用了这种模式（见trap/01-sync-iota）：

```go
const (
    mutexLocked      = 1 << iota // 1
    mutexWoken                   // 2
    mutexStarving                // 4
    mutexWaiterShift = iota      // 3（注意这里切换回了iota本身的值）
)
```

## 2.3 跳值与从非零开始

使用`_`跳过不需要的值：

```go
type Season int

const (
    _      Season = iota // 跳过0
    Spring               // 1
    Summer               // 2
    Autumn               // 3
    Winter               // 4
)
```

使用表达式偏移起始值：

```go
type HTTPStatus int

const (
    StatusOK          HTTPStatus = iota + 200 // 200
    StatusCreated                              // 201
    StatusAccepted                             // 202
)
```

## 2.4 字符串化枚举

裸枚举值在调试和日志中只是数字，可读性差。为枚举实现`String()`方法可以获得可读的输出：

```go
type Color int

const (
    Red Color = iota
    Green
    Blue
)

func (c Color) String() string {
    switch c {
    case Red:
        return "Red"
    case Green:
        return "Green"
    case Blue:
        return "Blue"
    default:
        return fmt.Sprintf("Color(%d)", c)
    }
}

func main() {
    fmt.Println(Red)   // Red（而不是0）
    fmt.Println(Green) // Green
}
```

当枚举值较多时，手写`String()`方法容易遗漏。可以使用官方工具`stringer`自动生成：

```bash
go install golang.org/x/tools/cmd/stringer@latest

# 为Color类型自动生成String()方法
//go:generate stringer -type=Color
```


# 3 工程实践建议

## 3.1 定义哨兵值用于边界检查

在枚举的首尾定义哨兵值，方便进行合法性校验：

```go
type Role int

const (
    roleBegin Role = iota
    Admin
    Editor
    Viewer
    roleEnd
)

func (r Role) IsValid() bool {
    return r > roleBegin && r < roleEnd
}
```

## 3.2 避免序列化陷阱

iota枚举的值取决于声明顺序。如果将iota值持久化到数据库或用于API响应，后续在中间插入新值会导致已有数据含义错乱：

```go
// 版本1
const (
    StatusPending Status = iota // 0
    StatusActive                // 1
    StatusClosed                // 2
)

// 版本2：在中间插入了StatusSuspended
const (
    StatusPending   Status = iota // 0
    StatusActive                  // 1
    StatusSuspended               // 2 ← 新增
    StatusClosed                  // 3 ← 值从2变成了3！
)
```

**解决方案：** 需要持久化或跨服务传输的枚举，应显式指定值而非依赖iota：

```go
const (
    StatusPending   Status = 0
    StatusActive    Status = 1
    StatusClosed    Status = 2
    StatusSuspended Status = 3 // 新增值放末尾，或显式指定
)
```

## 3.3 iota与类型安全

始终为iota枚举定义自定义类型，而不是使用裸int。这样编译器能帮助发现类型误用：

```go
// 好：有类型保护
type Direction int
const (
    North Direction = iota
    South
    East
    West
)

func Move(d Direction) { /* ... */ }
Move(North) // 合法
Move(42)    // 编译通过但语义错误，Go不阻止int到Direction的隐式转换

// 更好：函数签名约束了参数类型，至少在代码审查时能发现裸数字
```

需要注意的是，Go的自定义类型（如`type Direction int`）底层仍然是int，编译器**不会**阻止
`Move(42)`这样的调用。但使用自定义类型至少在代码可读性和审查层面提供了保护。
