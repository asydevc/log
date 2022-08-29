// author: asydevc <asydev@163.com>
// date: 2021-02-23

package interfaces

import (
	"net/http"
)

type TraceInterface interface {
	GenCurrentVersion() string
	GenPreviewVersion() string
	GenVersion(i int32) string
	GetParentSpanId() string
	GetSpanId() string
	GetSpanVersion() string
	GetTraceId() string
	IncrOffset() (before int32, after int32)
	RequestInfo() (method string, url string)
	Use(traceId, spanVersion string) TraceInterface
	UseDefault() TraceInterface
	UseRequest(req *http.Request) TraceInterface
}
