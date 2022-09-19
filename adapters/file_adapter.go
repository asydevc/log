// author: asydevc <asydev@163.com>
// date: 2021-02-22

package adapters

import (
	"encoding/json"
	"fmt"
	"github.com/asydevc/log/v2/interfaces"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sync"
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
				go func(line interfaces.LineInterface) {
					err := o.send(line)
					if err != nil {
						return
					}
				}(line)
			}
		}
	}()
}

// Send log.
func (o *fileAdapter) send(line interfaces.LineInterface) (err error) {
	defer func() {
		if r := recover(); r != nil {
			o.handler(line)
		}
	}()
	lines := o.body(line)
	// define log filename
	var file = line.ServiceName()
	if o.Conf.UseMonth == true {
		file = fmt.Sprintf("%.10s", line.Timeline())
	}
	// creat log directory
	if _, err := os.Stat(o.Conf.Path); os.IsNotExist(err) {
		err = os.MkdirAll(o.Conf.Path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// combine
	filePath := filepath.Join(o.Conf.Path, file+".txt")

	// 创建文件
	var fi *os.File
	fi, err = os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer func(fi *os.File) {
		err := fi.Close()
		if err != nil {
			return
		}
	}(fi)

	// 写文件
	_, err = fi.WriteString(fmt.Sprintf("[%s][%s] %s", line.Level(), line.Timeline(), lines) + "\r\n")
	if err != nil {
		return err
	}
	return nil
}

func NewFile() *fileAdapter {
	o := &fileAdapter{ch: make(chan interfaces.LineInterface), mu: new(sync.RWMutex)}
	// Parse configuration.
	// 1. base config.
	for _, file := range []string{"./tmp/log.yaml", "../tmp/log.yaml", "./config/log.yaml", "../config/log.yaml"} {
		body, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if yaml.Unmarshal(body, o) != nil {
			continue
		}
		break
	}
	// 2. default value
	if o.Conf == nil {
		o.Conf = &fileConfig{Path: "./logs"}
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
