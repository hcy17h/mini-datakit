package point

import (
	"testing"
	"time"
)

func TestLine(t *testing.T) {
	p := New("cpu").
		WithTag("host", "web01").
		WithTag("core", "cpu0").
		WithField("usage", 12.5).
		WithField("user", 8.1).
		WithTime(time.Unix(0, 1730000000000000000))

	want := `cpu,core=cpu0,host=web01 usage=12.5,user=8.1 1730000000000000000`
	if got := p.Line(); got != want {
		t.Fatalf("Line() = %q, 期望 %q", got, want)
	}
}

// TestLineIsStable 验证 map 随机顺序不会导致输出抖动。
// 这个测试在你把 sortedKeys 去掉之后会挂——这就是它的价值。
func TestLineIsStable(t *testing.T) {
	p := New("mem").
		WithField("a", 1).
		WithField("b", 2).
		WithField("c", 3)

	first := p.Line()
	for i := 0; i < 100; i++ {
		if got := p.Line(); got != first {
			t.Fatalf("第 %d 次输出不一致: %q != %q", i, got, first)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := New("cpu").Validate(); err == nil {
		t.Fatal("没有 field 的 point 应该报错")
	}
	if err := New("").WithField("usage", 1).Validate(); err == nil {
		t.Fatal("measurement 为空应该报错")
	}
	if err := New("cpu").WithField("usage", 1).Validate(); err != nil {
		t.Fatalf("合法的 point 不该报错: %v", err)
	}
}
