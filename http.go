package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_kcache/" // 默认的HTTP路径前缀，用于标识geecache服务的路径

// HTTPPool 实现了 PeerPicker 接口，用于管理一组通过 HTTP 通信的对等节点。
type HTTPPool struct {
	// 当前节点的基地址，例如 "https://example.net:8000"
	self     string
	basePath string // HTTP服务的路径前缀，默认为 defaultBasePath
}

// NewHTTPPool 初始化一个 HTTPPool 实例，指定当前节点的地址。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 在日志中记录信息，并带上当前服务器的名称
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有 HTTP 请求，是 HTTPPool 的核心请求处理函数。
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查请求路径是否以 basePath 开头
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path) // 记录请求方法和路径

	// 解析请求路径，格式应为 /<basePath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest) // 如果路径格式不正确，返回400错误
		return
	}

	groupName := parts[0] // 缓存组名
	key := parts[1]       // 缓存键

	group := GetGroup(groupName) // 根据组名获取缓存组
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound) // 如果组不存在，返回404错误
		return
	}

	view, err := group.Get(key) // 从缓存组中获取键对应的值
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 如果获取失败，返回500错误
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream") // 设置响应头，表示返回二进制数据
	w.Write(view.ByteSlice())                                  // 将缓存值写入响应体
}
