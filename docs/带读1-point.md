# 带读 1：把 `point.go` 拆成 9 段读

> 只读这一个文件，113 行。别跳，别同时读别的。
> 读完你能回答 3 个问题，就算过关。

---

## 先看清楚 Point 是什么

`Point` 就是**一条观测数据**，能直接画成一张表：

| measurement（表名） | Tags（维度） | Fields（指标） | Time（时间） |
|---|---|---|---|
| `cpu` | `host=web01, core=cpu0` | `usage=12.5, user=8.1` | 1730000000000000000 |
| `cpu` | `host=web01, core=cpu1` | `usage=18.0, user=9.3` | 1730000000000000000 |
| `mem` | `host=web01` | `used_percent=63.2` | 1730000000000000000 |

**整份代码干的事只有两件：**

1. 怎么把一条数据造出来（`New` / `WithTag` / `WithField` / `WithTime`）
2. 怎么把它变成一行文本（`Line`）

其余的（`Validate` / `sortedKeys` / `escapeName`）都是给这两件事打杂的。

---

## 读之前：6 把钥匙

Go 代码看不懂，99% 是因为缺这几把钥匙，而不是逻辑难。**只看这 6 条，不要扩展。**

### 钥匙 1：`type X struct { ... }` = 定义一张表有哪些列

```go
type Point struct {
	Measurement string
	Tags        map[string]string
}
```

= "有一条数据叫 Point，它有一个名字叫 Measurement，还有一堆标签叫 Tags"。

### 钥匙 2：`map[string]string` = 键值对的小本子

- `map[string]string`：左边是名字，右边也是名字。`{"host": "web01", "core": "cpu0"}`
- `map[string]float64`：左边名字，右边数字。`{"usage": 12.5, "user": 8.1}`

**取值写法：** `p.Tags["host"]` → 拿到 `"web01"`。**存值写法：** `p.Tags["host"] = "web01"`。

### 钥匙 3：`func (p *Point) WithTag(...)` = 绑定在这张表上的动作

```go
func (p *Point) WithTag(k, v string) *Point
     ↑           ↑
   "绑在谁身上"   "这个动作的名字"
```

`p` 就是"这条数据自己"（跟 Python 的 `self` 一个意思）。**看到 `p.` 就念成"我自己的"**：

`p.Tags[k] = v` → "把我自己的 Tags 里 k 这一项设成 v"。

`*` 表示"操作原件，不是复印件"。Go 默认传复印件，加 `*` 才是原件——所以 `p.Tags[k] = v` 才能真的改到那条数据。

### 钥匙 4：`&Point{...}` = 造一张表，把它的位置给你

```go
return &Point{ Measurement: measurement }
```

`&` 读作"取地址"。所以 `New()` 的作用是：**造一条新数据，把它的门牌号（指针）给你**。

对应地，返回类型写 `*Point` 就是"返回一个门牌号"。

### 钥匙 5：`error` 不是异常，是"可能出错的返回值"

```go
func (p *Point) Validate() error {
	if p.Measurement == "" {
		return fmt.Errorf("measurement 不能为空")   // 有错：返回一个错误
	}
	return nil                                      // 没错：返回 nil
}
```

`nil` 就是"啥也没有"，在这里表示"没有错误"。

**Go 里判断出错永远长这样**（记住这个形状，全项目到处都是）：

```go
err := something()
if err != nil {
    // 出错了
}
```

### 钥匙 6：`for _, k := range m` = 遍历

```go
for i, k := range 某个map {   // i = 第几个，k = 这一项的名字
for _, k := range 某个map {   // 只关心名字，序号用 _ 扔掉
```

另外：`:=` 是"声明并赋值"（`x := 5`），`var x 类型` 是"只声明"。

---

## 现在开始分段读

打开 VS Code，左边放 `point.go`，右边放这份文档，一段一段对照。

---

### 第 1 段（1–13 行）：这是什么包，要用什么工具

```go
package point                    // 这个文件属于 point 包（= point 这个文件夹）
import ( "fmt"; "sort"; ... )    // 我要用的 5 个标准库工具
```

**只要知道：** `package` = 文件夹名；`import` = 我要借用哪些工具。

（`fmt` 做格式化输出和造错误，`sort` 排序，`strconv` 数字转字符串，`strings` 字符串操作，`time` 时间。）

---

### 第 2 段（20–25 行）：定义表结构

```go
type Point struct {
	Measurement string                 // 表名，字符串
	Tags        map[string]string      // 维度，键值对（值也是字符串）
	Fields      map[string]float64     // 指标，键值对（值是数字）
	Time        time.Time              // 时间
}
```

**这一段是整个项目的"数据长什么样"。** 后面所有代码都在伺候这四个字段。

> 为什么不把 usage、user 这些直接写成字段？
> 因为不同采集器的指标完全不同（CPU 有 usage，内存有 used_percent，硬盘有 iops）。
> 用 map 才能装下任意指标 —— 这是**取舍**，面试会问。

---

### 第 3 段（28–35 行）：怎么造一条新数据

```go
func New(measurement string) *Point {
	return &Point{
		Measurement: measurement,             // 用传进来的名字
		Tags:        make(map[string]string), // 造一个空的标签本
		Fields:      make(map[string]float64),// 造一个空的指标本
		Time:        time.Now(),              // 现在的时间
	}
}
```

**逐句翻译：**

> 给我一个名字，我造一条数据：名字用它，两个本子都是空的，时间取现在。

**`make(...)` 是干什么的？** Go 里 map 不能直接用，必须先 `make` 出来，否则往里写东西会崩。所以这一步**必须**有。

**`time.Now()`** 是标准库函数，返回当前时间。

---

### 第 4 段（38–53 行）：三个"设置"动作

这三个长得一样，学一个就够：

```go
func (p *Point) WithTag(k, v string) *Point {
	p.Tags[k] = v      // 把标签本里 k 这一项设成 v
	return p           // 把自己还回去
}
```

`WithField` 换成 `p.Fields[k] = v`，`WithTime` 换成 `p.Time = t`，其他一样。

**为什么要 `return p`？** 为了能连着写：

```go
p := point.New("cpu").
    WithTag("host", "web01").      // 返回 p，所以能接着点
    WithField("usage", 12.5).
    WithTime(t)
```

这叫**链式调用**。如果不返回 `p`，你就只能一行一行写。

> `k, v string` 是简写，等于 `k string, v string`。Go 允许相邻同类型参数合并。

---

### 第 5 段（57–65 行）：检查数据合不合法

```go
func (p *Point) Validate() error {
	if p.Measurement == "" {                        // 名字是空的？
		return fmt.Errorf("point: measurement 不能为空")
	}
	if len(p.Fields) == 0 {                          // 一个指标都没有？
		return fmt.Errorf("point: measurement=%s 没有任何 field", p.Measurement)
	}
	return nil                                       // 都合格
}
```

**逐句翻译：**

> 名字空 → 报错。指标本空 → 报错。都行 → 返回 nil（没错误）。

**`len(p.Fields)`** = 这个 map 里有几项。

**为什么要有这个函数？** 一条没有指标的数据存进去就是垃圾，查的时候还得处理空行。**在入口就拦掉**比在出口修补便宜得多。

---

### 第 6 段（71–95 行）：变成一行文本（最长的，拆 4 步看）

**目标产出长这样：**

```
cpu,core=cpu0,host=web01 usage=12.5,user=8.1 1730000000000000000
└─表名─┘└────标签，逗号分隔────┘└──指标──┘ └───时间戳───┘
```

代码就是按这个顺序，一块一块往纸上写：

```go
var b strings.Builder              // ① 拿一张纸（高效拼接用）

b.WriteString(escapeName(p.Measurement))   // ② 先写表名
```

```go
for _, k := range sortedKeys(p.Tags) {     // ③ 再写标签，每个前面加逗号
	b.WriteByte(',')
	b.WriteString(escapeName(k))           //    名字
	b.WriteByte('=')                       //    等号
	b.WriteString(escapeName(p.Tags[k]))   //    值
}
```

```go
b.WriteByte(' ')                           // ④ 一个空格，分隔标签和指标
for i, k := range sortedKeys(p.Fields) {
	if i > 0 { b.WriteByte(',') }          //    第一个不加逗号，之后都加
	b.WriteString(escapeName(k))
	b.WriteByte('=')
	b.WriteString(strconv.FormatFloat(p.Fields[k], 'g', -1, 64))  // 数字转文本
}
```

```go
b.WriteByte(' ')                           // ⑤ 空格，然后写时间戳
b.WriteString(strconv.FormatInt(p.Time.UnixNano(), 10))
return b.String()                          // ⑥ 把纸上的内容交出去
```

**几个细节：**

- `' '` 用**单引号** = 一个字符；`","` 用**双引号** = 一串字符串。写错了编译不过。
- `strings.Builder` 是"高效拼接"工具。用 `+` 也能拼，但数据量大时慢很多。**这是性能取舍，可以讲。**
- `strconv.FormatFloat(12.5, 'g', -1, 64)` → `"12.5"`（`'g'` = 最短表示，否则会变成 `12.500000`）
- `p.Time.UnixNano()` → 时间转成纳秒整数 `1730000000000000000`

---

### 第 7 段（98 行）：一行的小工具

```go
func (p *Point) String() string { return p.Line() }
```

`String()` 是 Go 的一个**约定名字**：只要你的类型有这个动作，`fmt.Println(p)` 打印出来就是它返回的内容。调试时很有用。

---

### 第 8 段（101–108 行）：把 map 的 key 拿出来排序

```go
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))   // 准备一个空列表（预估能装 len(m) 个）
	for k := range m {                  // 把 map 里每个名字
		keys = append(keys, k)          // 挨个塞进列表
	}
	sort.Strings(keys)                  // 排序
	return keys                         // 交出去
}
```

**`[V any]` 是什么？** 泛型。意思是"这个函数对任何类型的 map 都能用"——传 `map[string]string` 也行，传 `map[string]float64` 也行，不用写两遍。`V` 就是个占位符。

**`[]string`** 读作"字符串列表"（切片）。

**为什么要排序？** Go 的 map 遍历顺序**每次都随机**（语言规定，故意打乱，防止你依赖顺序）。不排序的话，同一条数据每次拼出来顺序都不一样，测试就随机失败。

> 这是第 8 段的全部价值。**面试题："为什么你的序列化要先排序？"答案就在这。**

---

### 第 9 段（111–113 行）：转义特殊字符

```go
func escapeName(s string) string {
	return strings.NewReplacer(" ", `\ `, ",", `\,`, "=", `\=`).Replace(s)
}
```

**翻译：** 把字符串里的空格、逗号、等号前面各加一个反斜杠。

**为什么？** 因为 line protocol 用逗号分标签、用等号分键值。如果你的 host 名字叫 `web 01`（带空格），不打转义就会把一行拆成两半，服务端解析崩掉。

**注意反引号 `` ` ``**：Go 里反引号包起来的字符串不做转义处理，所以 `\ ` 就是两个字符（反斜杠 + 空格），正好是要输出的内容。用双引号写 `"\\ "` 也行，但难看。

---

## 两个 3 分钟实验（必须做）

**看不懂就靠"破坏它"来懂。** 这是最快的办法。

### 实验 A：让测试挂一次

打开 [`point_test.go`](mini-datakit/internal/point/point_test.go)，找到这一行：

```go
want := `cpu,core=cpu0,host=web01 usage=12.5,user=8.1 1730000000000000000`
```

把 `12.5` 改成 `99`，然后跑：

```powershell
go test ./internal/point/ -v
```

**你会看到测试红了**，并打印出"实际得到什么 / 期望什么"。看完把 `12.5` 改回来。

> 目的：让你亲眼看到"测试"到底在断言什么。它不是玄学，就是**拿实际输出跟期望文本逐字符比**。

### 实验 B：去掉排序，看它怎么挂

把第 75 行的 `sortedKeys(p.Tags)` 改成直接遍历 `p.Tags`：

```go
for _, k := range p.Tags {        // 去掉了 sortedKeys
```

跑测试：

```powershell
go test ./internal/point/ -v
```

**多跑几次**（跑 5 次），你会看到它**时好时坏**——这就是"随机失败"。看完改回来。

> 目的：让你亲手体验"map 顺序随机"这件事有多坑。以后面试被问"你遇到过最难查的 bug"，这就是一个。

---

## 过关问题（3 个，很容易）

用自己的话答，两三句就行：

**Q1** 一个 `Point` 里装了哪四样东西？各举个例子。

**Q2** `New()` 里为什么要 `make(map[string]string)`？如果删掉这一行，会怎样？

**Q3** `WithTag` 为什么最后要 `return p`？去掉会有什么后果？

答完这三个，我们读 `registry.go`（只有 60 行，讲"插件怎么自己挂上去"）。

---

## 卡住了怎么办

| 症状 | 做法 |
|---|---|
| 某一行完全看不懂 | 把那行**原样**贴给我，我逐字解释 |
| 不知道 `:=` 和 `=` 区别 | 记住：`:=` 是"第一次出现时用"，`=` 是"已经有的改值" |
| 分不清 `*` 和 `&` | `&x` = 取门牌号；`*T` = "这是个门牌号类型" |
| 读不进去 | 一次只读一段，读完合上文件，用嘴复述一遍那段在干嘛 |
