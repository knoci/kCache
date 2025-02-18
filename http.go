package main

import (
	"fmt"
	"io/ioutil"
	"kCache/consistenthash"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 默认的基础路径和副本数量。
const (
	defaultBasePath = "/_geecache/" // 默认的基础路径
	defaultReplicas = 50            // 默认的副本数量
)

// httpGetter 是一个用于从远程 HTTP 服务器获取数据的结构体。
type httpGetter struct {
	baseURL string // 基础 URL，用于构建完整的请求地址
}

// HTTPPool 是一个实现 PeerPicker 接口的结构体，用于管理一组 HTTP 对等节点。
type HTTPPool struct {
	self        string                 // 当前节点的基础 URL
	basePath    string                 // 基础路径
	mu          sync.Mutex             // 互斥锁，保护 peers 和 httpGetters
	peers       *consistenthash.Map    // 一致性哈希映射
	httpGetters map[string]*httpGetter // 存储每个对等节点的 httpGetter 实例
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

// Get 方法通过 HTTP GET 请求从远程服务器获取指定键的值。
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 构建完整的请求 URL，包括对 group 和 key 的 URL 编码。
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 发起 HTTP GET 请求。
	res, err := http.Get(u)
	if err != nil {
		return nil, err // 如果请求失败，返回错误
	}
	defer res.Body.Close() // 确保响应体在函数返回时关闭

	// 检查 HTTP 响应状态码。
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status) // 如果状态码不是 200 OK，返回错误
	}

	// 读取响应体内容。
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err) // 如果读取失败，返回错误
	}

	return bytes, nil // 返回读取到的数据
}

// 确保 httpGetter 实现了 PeerGetter 接口。
var _ PeerGetter = (*httpGetter)(nil)

// Set 方法更新节点池中的对等节点列表。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)       // 创建一致性哈希映射
	p.peers.Add(peers...)                                    // 添加对等节点
	p.httpGetters = make(map[string]*httpGetter, len(peers)) // 初始化 httpGetters 映射
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath} // 为每个对等节点创建 httpGetter 实例
	}
}

// PickPeer 方法根据键选择一个对等节点。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self { // 根据键选择对等节点
		p.Log("Pick peer %s", peer)      // 日志记录
		return p.httpGetters[peer], true // 返回对等节点的 httpGetter 实例
	}
	return nil, false // 如果没有找到合适的对等节点，返回 nil
}

// 确保 HTTPPool 实现了 PeerPicker 接口。
var _ PeerPicker = (*HTTPPool)(nil)
