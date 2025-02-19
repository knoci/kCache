package main

import (
	"fmt"
	"kCache/singleflight"
	"log"
	"sync"
)

// Getter 是一个接口，用于加载键对应的值。
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 是一个函数类型，实现了 Getter 接口。
type GetterFunc func(key string) ([]byte, error)

// Get 实现了 Getter 接口的方法。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 是一个缓存命名空间，关联了加载数据的逻辑。
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建一个新的 Group 实例，并将其注册到全局 map 中。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup 返回之前通过 NewGroup 创建的 Group 实例，如果不存在则返回 nil。
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 从缓存中获取键对应的值，如果缓存中不存在，则通过 Getter 加载数据。
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok { // 尝试从缓存中获取数据
		log.Println("[kCache] hit")
		return v, nil
	}

	return g.load(key) // 缓存中没有命中，加载数据
}

// getLocally 从本地加载数据，并将其填充到缓存中。
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)} // 创建 ByteView 实例
	g.populateCache(key, value)             // 将数据填充到缓存
	return value, nil
}

// populateCache 将数据添加到缓存中。
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// RegisterPeers 注册一个 PeerPicker，用于选择远程对等节点。
// PeerPicker 负责根据键选择合适的对等节点。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once") // 确保只调用一次
	}
	g.peers = peers // 将 PeerPicker 实例绑定到缓存组
}

// load 方法尝试从本地或远程对等节点加载指定键的值。
// load 方法尝试从本地或远程对等节点加载指定键的值。
// 每个键的加载操作只执行一次，无论有多少并发调用者。
func (g *Group) load(key string) (value ByteView, err error) {
	// 使用 singleflight.Group 确保每个键的加载操作只执行一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 如果已注册 PeerPicker，尝试从远程对等节点获取数据
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				// 从远程对等节点获取数据
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[kCache] Failed to get from peer", err)
			}
		}

		// 如果远程获取失败或未注册 PeerPicker，则从本地加载
		return g.getLocally(key)
	})

	// 如果加载成功，返回 ByteView 类型的结果
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// getFromPeer 从指定的对等节点获取数据。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// 调用对等节点的 Get 方法获取数据
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err // 如果获取失败，返回错误
	}
	return ByteView{b: bytes}, nil // 包装数据并返回
}
