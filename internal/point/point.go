// Package point 定义 mini-datakit 的统一数据模型。
//
// 约定：所有采集插件（inputs）只产出 Point，所有下游（writer、storage、query）
// 只消费 Point。这样一来"新增一个采集器"不需要动任何下游代码。
package point

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Point 是一条观测数据，等价于 InfluxDB line protocol 的一行：
//
//	cpu,core=cpu0,host=web01 usage=12.5,user=8.1 1730000000000000000
//
//	Measurement 是"表名"，Tags 是索引维度，Fields 是指标值，Time 是采样时刻。
type Point struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]float64
	Time        time.Time
}

// New 创建一个 Point 并初始化 Tags/Fields，避免使用方到处判空。
func New(measurement string) *Point {
	return &Point{
		Measurement: measurement,
		Tags:        make(map[string]string), // map[string]string{} 也可一样初始化，但 make() 更常用
		Fields:      make(map[string]float64),
		Time:        time.Now(),
	}
}

// WithTag 设置一个 tag（维度），返回自身以便链式调用。
func (p *Point) WithTag(k, v string) *Point {
	p.Tags[k] = v
	return p
}

// WithField 设置一个 field（指标值）。
func (p *Point) WithField(k string, v float64) *Point {
	p.Fields[k] = v
	return p
}

// WithTime 设置采样时间。
func (p *Point) WithTime(t time.Time) *Point {
	p.Time = t
	return p
}

// Validate 检查数据是否合法。不合格的数据应该在写入存储之前就被拒绝，
// 而不是等到查询的时候才发现有一堆空行。
func (p *Point) Validate() error {
	if p.Measurement == "" {
		return fmt.Errorf("point: measurement 不能为空")
	}
	if len(p.Fields) == 0 {
		return fmt.Errorf("point: measurement=%s 没有任何 field", p.Measurement)
	}
	return nil
}

// Line 序列化成 line protocol 文本。
//
// 注意：Go 里 map 的遍历顺序是随机的，所以这里必须先排序再拼接。
// 否则同一份数据每次输出都不一样，测试会随机失败。
func (p *Point) Line() string {
	var b strings.Builder

	b.WriteString(escapeName(p.Measurement))
	for _, k := range sortedKeys(p.Tags) {
		b.WriteByte(',')
		b.WriteString(escapeName(k))
		b.WriteByte('=')
		b.WriteString(escapeName(p.Tags[k]))
	}

	b.WriteByte(' ')
	for i, k := range sortedKeys(p.Fields) {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(escapeName(k))
		b.WriteByte('=')
		b.WriteString(strconv.FormatFloat(p.Fields[k], 'g', -1, 64))
	}

	b.WriteByte(' ')
	b.WriteString(strconv.FormatInt(p.Time.UnixNano(), 10))
	return b.String()
}

// String 让 fmt 打印出来就是 line protocol，方便调试。
func (p *Point) String() string { return p.Line() }

// sortedKeys 取出 map 的 key 并排序，保证输出稳定。
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// escapeName 转义 line protocol 里的特殊字符（空格、逗号、等号）。
func escapeName(s string) string {
	return strings.NewReplacer(" ", `\ `, ",", `\,`, "=", `\=`).Replace(s)
}
