package main

import (
	"kCache/lru"
	"sync"
)

// cache 是一个简单的缓存结构，使用 LRU 算法管理缓存项。
type cache struct {
	mu         sync.Mutex // 互斥锁，用于保护并发访问
	lru        *lru.Cache // LRU 缓存实例
	cacheBytes int64      // 缓存的最大字节数限制
}

// add 方法向缓存中添加一个键值对。
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 加锁，确保并发安全
	defer c.mu.Unlock() // 确保在方法结束时释放锁
	if c.lru == nil {   // 如果 LRU 缓存实例尚未初始化
		c.lru = lru.New(c.cacheBytes, nil) // 根据缓存大小限制初始化 LRU 缓存
	}
	c.lru.Add(key, value) // 将键值对添加到 LRU 缓存中
}

// get 方法从缓存中获取一个键对应的值。
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 加锁，确保并发安全
	defer c.mu.Unlock() // 确保在方法结束时释放锁
	if c.lru == nil {   // 如果 LRU 缓存实例尚未初始化
		return
	}
	if v, ok := c.lru.Get(key); ok { // 尝试从 LRU 缓存中获取键对应的值
		return v.(ByteView), ok // 如果存在，将值断言为 ByteView 类型并返回
	}
	return
}
