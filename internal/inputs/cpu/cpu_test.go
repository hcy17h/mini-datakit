package cpu

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 两段伪造的 /proc/stat 内容，用来验证"增量 → 百分比"的算法。
// 总 jiffies：A = 1000，B = 2000，增量 1000。
// 其中 user 增 100、system 增 50、idle 增 850 →
// usage = (1000-850-0)/1000 = 15%，user = 10%，system = 5%。
const (
	sampleA = `cpu  100 0 50 850 0 0 0 0 0 0
cpu0 100 0 50 850 0 0 0 0 0 0
intr 12345
ctxt 6789
btime 1730000000
`
	sampleB = `cpu  200 0 100 1700 0 0 0 0 0 0
cpu0 200 0 100 1700 0 0 0 0 0 0
intr 12346
ctxt 6790
btime 1730000000
`
)

func TestParseStat(t *testing.T) {
	m, err := parseStat(strings.NewReader(sampleA))
	if err != nil {
		t.Fatalf("parseStat 失败: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("期望 2 个 cpu（cpu + cpu0），实际 %d", len(m))
	}
	if got := m["cpu"]; got.user != 100 || got.idle != 850 || got.total() != 1000 {
		t.Fatalf("解析结果不对: %+v", got)
	}
}

func TestParseStatRejectsGarbage(t *testing.T) {
	if _, err := parseStat(strings.NewReader("intr 1\nctxt 2\n")); err == nil {
		t.Fatal("没有 cpu 行时应该报错")
	}
}

func TestCollectFirstCallIsBaselineOnly(t *testing.T) {
	path := writeStat(t, sampleA)
	in, err := newInput(map[string]any{"path": path, "host": "test"})
	if err != nil {
		t.Fatal(err)
	}

	pts, err := in.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect 失败: %v", err)
	}
	if len(pts) != 0 {
		t.Fatalf("第一轮只建基线，应该返回 0 条，实际 %d 条", len(pts))
	}
}

func TestCollectComputesDelta(t *testing.T) {
	path := writeStat(t, sampleA)
	in, err := newInput(map[string]any{"path": path, "host": "test"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := in.Collect(ctx); err != nil { // 第一轮建立基线
		t.Fatalf("Collect 失败: %v", err)
	}

	if err := os.WriteFile(path, []byte(sampleB), 0o644); err != nil {
		t.Fatal(err)
	}
	pts, err := in.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect 失败: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("期望 2 条数据，实际 %d", len(pts))
	}

	var got *struct {
		usage, user, system, iowait float64
	}
	for _, p := range pts {
		if p.Tags["core"] != "cpu0" {
			continue
		}
		if p.Tags["host"] != "test" {
			t.Fatalf("host tag 不对: %v", p.Tags)
		}
		if p.Measurement != "cpu" {
			t.Fatalf("measurement 不对: %q", p.Measurement)
		}
		got = &struct{ usage, user, system, iowait float64 }{
			p.Fields["usage"], p.Fields["user"], p.Fields["system"], p.Fields["iowait"],
		}
	}
	if got == nil {
		t.Fatal("没有找到 core=cpu0 的数据")
	}

	check := func(name string, v, want float64) {
		if math.Abs(v-want) > 1e-9 {
			t.Errorf("%s = %v，期望 %v", name, v, want)
		}
	}
	check("usage", got.usage, 15)
	check("user", got.user, 10)
	check("system", got.system, 5)
	check("iowait", got.iowait, 0)
}

func TestCollectMissingFile(t *testing.T) {
	in, err := newInput(map[string]any{"path": filepath.Join(t.TempDir(), "不存在")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := in.Collect(context.Background()); err == nil {
		t.Fatal("文件不存在时应该返回错误")
	}
}

func writeStat(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stat")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
