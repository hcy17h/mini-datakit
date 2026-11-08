package inputs

import (
	"context"
	"testing"

	"mini-datakit/internal/point"
)

// fakeInput 是最小的 Input 实现，只用来测注册表本身。
type fakeInput struct{ name string }

func (f *fakeInput) Name() string { return f.name }

func (f *fakeInput) Collect(ctx context.Context) ([]point.Point, error) {
	return []point.Point{*point.New(f.name).WithField("ok", 1)}, nil
}

func TestRegisterAndNew(t *testing.T) {
	Register("fake_w1", func(conf map[string]any) (Input, error) {
		return &fakeInput{name: "fake_w1"}, nil
	})

	in, err := New("fake_w1", nil)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	if in.Name() != "fake_w1" {
		t.Fatalf("Name() = %q", in.Name())
	}

	if _, err := New("根本不存在的插件", nil); err == nil {
		t.Fatal("不存在的插件应该返回错误")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	mk := func(conf map[string]any) (Input, error) {
		return &fakeInput{name: "fake_dup"}, nil
	}
	Register("fake_dup", mk)

	defer func() {
		if recover() == nil {
			t.Fatal("重复注册同名插件应该 panic")
		}
	}()
	Register("fake_dup", mk)
}
