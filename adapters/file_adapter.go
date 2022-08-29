// author: asydevc <asydev@163.com>
// date: 2021-02-22

package adapters

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/asydevc/log/v2/interfaces"
)

// 文件配置.
type fileConfig struct {
	Path     string `yaml:"path"`
	UseMonth bool   `yaml:"use-month"`
}

type fileAdapter struct {
	Conf    *fileConfig `yaml:"file"`
	ch      chan interfaces.LineInterface
	mu      *sync.RWMutex
	handler interfaces.Handler
}

func (o *fileAdapter) Run(line interfaces.LineInterface) {
	go func() {
		o.ch <- line
	}()
}

// Listen channel.
func (o *fileAdapter) listen() {
	go func() {
		defer o.listen()
		for {
			select {
			case line := <-o.ch:
				go o.send(line)
			}
		}
	}()
}

// Send log.
func (o *fileAdapter) send(line interfaces.LineInterface) {
	defer func() {
		if r := recover(); r != nil {
			o.handler(line)
		}
	}()
	lines := o.body(line)
	var file = line.ServiceName()
	if o.Conf.UseMonth {
		file = time.Now().Format("2006-01-02")
	}
	// 判断目录是否存在
	if _, err := os.Stat(o.Conf.Path); os.IsNotExist(err) {
		_ = os.MkdirAll(o.Conf.Path, 0666)
	}
	// 判断文件是否存在
	filePath := filepath.Join(o.Conf.Path+".txt", file)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fi, _ := os.Create(filePath)
		defer fi.Close()
		fi.Write([]byte(lines))
	}
}

func NewFile() *fileAdapter {
	o := &fileAdapter{ch: make(chan interfaces.LineInterface), mu: new(sync.RWMutex)}
	// Parse configuration.
	// 1. base config.
	for _, file := range []string{"./tmp/log.yaml", "../tmp/log.yaml", "./config/log.yaml", "../config/log.yaml"} {
		body, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		if yaml.Unmarshal(body, o) != nil {
			continue
		}
		break
	}
	o.listen()
	return o
}

func (o *fileAdapter) body(line interfaces.LineInterface) string {
	// Init
	data := make(map[string]interface{})
	// Basic.
	data["content"] = line.Content()
	data["duration"] = line.Duration()
	data["level"] = line.Level()
	data["time"] = line.Timeline()
	// Tracing.
	data["action"] = ""
	if line.Tracing() {
		data["parentSpanId"] = line.ParentSpanId()
		data["requestId"] = line.TraceId()
		data["requestMethod"], data["requestUrl"] = line.RequestInfo()
		data["spanId"] = line.SpanId()
		data["traceId"] = line.TraceId()
		data["version"] = line.SpanVersion()
	} else {
		data["parentSpanId"] = ""
		data["requestId"] = ""
		data["requestMethod"] = ""
		data["requestUrl"] = ""
		data["spanId"] = ""
		data["traceId"] = ""
		data["version"] = ""
	}
	// Server.
	data["module"] = line.ServiceName()
	data["pid"] = line.Pid()
	data["serverAddr"] = line.ServiceAddr()
	data["taskId"] = 0
	data["taskName"] = ""
	// JSON string.
	if body, err := json.Marshal(data); err == nil {
		return string(body)
	}
	return ""
}
