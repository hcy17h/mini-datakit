// Package cpu 采集 CPU 使用率。
//
// 关键认知：/proc/stat 里的数字是"自开机以来的累加计数器"，不是使用率。
// 所以必须记住上一次的采样值，两次相减得到这一段时间的增量，才能算占用百分比。
// 这就是监控系统里 counter（计数器）和 gauge（瞬时值）的区别。
package cpu

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"mini-datakit/internal/inputs"
	"mini-datakit/internal/point"
)

// 插件通过 init() 把自己登记进注册表，主程序用下划线导入触发这里执行。
func init() {
	inputs.Register("cpu", newInput)
}

// raw 是 /proc/stat 里一行 cpu 数据的原始计数器，单位 jiffies。
type raw struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (r raw) total() uint64 {
	return r.user + r.nice + r.system + r.idle + r.iowait + r.irq + r.softirq + r.steal
}

// busy 是"非空闲"的 jiffies。iowait 按 Linux 的习惯算进空闲。
func (r raw) busy() uint64 { return r.total() - r.idle - r.iowait }

// diff 计算两次采样的增量。调用前必须保证 r 的每个计数都不小于 o。
func (r raw) diff(o raw) raw {
	return raw{
		user:    r.user - o.user,
		nice:    r.nice - o.nice,
		system:  r.system - o.system,
		idle:    r.idle - o.idle,
		iowait:  r.iowait - o.iowait,
		irq:     r.irq - o.irq,
		softirq: r.softirq - o.softirq,
		steal:   r.steal - o.steal,
	}
}

// Input 是 cpu 采集插件。
type Input struct {
	path string
	host string

	mu   sync.Mutex
	prev map[string]raw // nil 表示还没有基线
}

func newInput(conf map[string]any) (inputs.Input, error) {
	path := "/proc/stat"
	if v, ok := conf["path"].(string); ok && v != "" {
		path = v
	}

	host, _ := os.Hostname()
	if v, ok := conf["host"].(string); ok && v != "" {
		host = v
	}

	return &Input{path: path, host: host}, nil
}

func (i *Input) Name() string { return "cpu" }

// Collect 读一次 /proc/stat（或配置里指定的同类文件）。
//
// 第一次调用只记录基线并返回 nil：没有上一次的值，算不出增量。
// 这是所有 counter 型指标的通用行为，不是 bug。
func (i *Input) Collect(ctx context.Context) ([]point.Point, error) {
	f, err := os.Open(i.path)
	if err != nil {
		return nil, fmt.Errorf("cpu: 打开 %s 失败（Windows 上没有 /proc，请在 WSL 或 Linux 里运行）: %w", i.path, err)
	}
	defer f.Close()

	cur, err := parseStat(f)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	i.mu.Lock()
	prev := i.prev
	i.prev = cur
	i.mu.Unlock()

	if prev == nil {
		return nil, nil // 基线轮，这一轮没有数据可算
	}

	pts := make([]point.Point, 0, len(cur))
	for name, c := range cur {
		p0, ok := prev[name]
		if !ok {
			continue // 新出现的核，下一轮再算
		}
		if c.total() < p0.total() {
			continue // 计数器变小 = 回绕或重启，这一轮丢弃
		}
		d := c.diff(p0)
		dt := d.total()
		if dt == 0 {
			continue
		}

		p := point.New("cpu").
			WithTag("host", i.host).
			WithTag("core", name).
			WithTime(now)
		p.WithField("usage", pct(d.busy(), dt))
		p.WithField("user", pct(d.user+d.nice, dt))
		p.WithField("system", pct(d.system, dt))
		p.WithField("iowait", pct(d.iowait, dt))
		pts = append(pts, *p)
	}
	return pts, nil
}

func pct(v, total uint64) float64 {
	return float64(v) * 100 / float64(total)
}

// parseStat 从 /proc/stat 的内容里取出所有 cpu 行（cpu、cpu0、cpu1……）。
// 这里把"解析"和"采集"分开，是为了能拿一段固定文本做单元测试。
func parseStat(r io.Reader) (map[string]raw, error) {
	out := make(map[string]raw, 9)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 { // 至少要有 user nice system idle
			continue
		}

		var vals [8]uint64
		for idx := 1; idx < len(fields) && idx <= 8; idx++ {
			v, err := strconv.ParseUint(fields[idx], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cpu: 解析 %q 第 %d 列失败: %w", line, idx, err)
			}
			vals[idx-1] = v
		}

		out[fields[0]] = raw{
			user:    vals[0],
			nice:    vals[1],
			system:  vals[2],
			idle:    vals[3],
			iowait:  vals[4],
			irq:     vals[5],
			softirq: vals[6],
			steal:   vals[7],
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("cpu: 读取失败: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("cpu: 文件里没有任何 cpu 行")
	}
	return out, nil
}
