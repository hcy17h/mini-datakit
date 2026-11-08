// Command agent 是 mini-datakit 的采集端：读配置 → 起插件 → 按周期采集 → 输出。
//
// W1 阶段"输出"就是打到标准输出。W2 会换成批量上传到 server。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"mini-datakit/internal/inputs"
	_ "mini-datakit/internal/inputs/cpu" // 下划线导入：只为触发 cpu 包的 init() 注册
)

type config struct {
	Inputs []inputConf `json:"inputs"`
}

type inputConf struct {
	Name     string         `json:"name"`
	Interval string         `json:"interval"`
	Conf     map[string]any `json:"conf"`
}

func main() {
	confPath := flag.String("config", "configs/agent.json", "配置文件路径")
	flag.Parse()

	cfg, err := loadConfig(*confPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// Ctrl+C / kill 时 ctx 会被取消，所有采集协程优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("已注册插件: %v", inputs.Names())

	var wg sync.WaitGroup
	for _, ic := range cfg.Inputs {
		in, err := inputs.New(ic.Name, ic.Conf)
		if err != nil {
			log.Fatalf("初始化插件 %s 失败: %v", ic.Name, err)
		}
		d, err := time.ParseDuration(ic.Interval)
		if err != nil || d <= 0 {
			log.Fatalf("插件 %s 的 interval=%q 不合法（示例：2s）", ic.Name, ic.Interval)
		}

		wg.Add(1)
		go func(in inputs.Input, d time.Duration) {
			defer wg.Done()
			run(ctx, in, d)
		}(in, d)
	}

	wg.Wait()
	log.Println("已退出")
}

// run 每个插件一个协程：立刻采一次，然后按周期采。
func run(ctx context.Context, in inputs.Input, d time.Duration) {
	collectOnce(ctx, in)

	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			collectOnce(ctx, in)
		}
	}
}

func collectOnce(ctx context.Context, in inputs.Input) {
	pts, err := in.Collect(ctx)
	if err != nil {
		log.Printf("[%s] 采集失败: %v", in.Name(), err)
		return
	}
	for _, p := range pts {
		if err := p.Validate(); err != nil {
			log.Printf("[%s] 数据非法，丢弃: %v", in.Name(), err)
			continue
		}
		fmt.Println(p.Line())
	}
}

func loadConfig(path string) (*config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("%s 不是合法 JSON: %w", path, err)
	}
	if len(c.Inputs) == 0 {
		return nil, fmt.Errorf("%s 里没有配置任何 input", path)
	}
	return &c, nil
}
