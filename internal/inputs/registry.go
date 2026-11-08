// Package inputs 提供采集插件的注册表。
//
// 设计参照观测云 DataKit 的插件机制：每个插件在自己的 init() 里调用 Register
// 把自己登记进来，主程序只 import 一次（下划线导入），不关心具体有哪些插件。
// 好处是"加采集器"= 加一个目录 + 加一行 import，主干代码一行不改。
package inputs

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"mini-datakit/internal/point"
)

// Input 是所有采集插件都必须实现的接口。
//
// 只有两个方法：上报名字，交出这一轮的采样点。
// 插件内部是读 /proc、发 HTTP 还是查数据库，调用方完全不关心。
type Input interface {
	Name() string
	Collect(ctx context.Context) ([]point.Point, error)
}

// Factory 根据一段配置造出一个 Input。配置文件里每个 inputs 项都交给它。
type Factory func(conf map[string]any) (Input, error)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register 登记一个插件。名字重复说明代码写错了，直接 panic，
// 让问题在启动时就暴露，而不是运行时偷偷用错插件。
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := factories[name]; dup {
		panic(fmt.Sprintf("inputs: 插件 %q 重复注册", name))
	}
	factories[name] = f
}

// New 按名字创建插件实例。
func New(name string, conf map[string]any) (Input, error) {
	mu.RLock()
	f, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("inputs: 没有名为 %q 的插件（已注册：%v）", name, Names())
	}
	return f(conf)
}

// Names 返回所有已注册的插件名（已排序，方便打印）。
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(factories))
	for n := range factories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
