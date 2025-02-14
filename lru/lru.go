package lru

import (
	"container/list"
	"sync"
)

// Cache 是一个线程安全的LRU缓存。
type Cache struct {
	maxBytes  int64                         // 缓存的最大字节数限制
	nbytes    int64                         // 当前缓存占用的字节数
	ll        *list.List                    // 使用双向链表维护最近最少使用的顺序
	cache     map[string]*list.Element      // 将键映射到链表中的元素
	mu        sync.Mutex                    // 互斥锁，用于线程安全
	OnEvicted func(key string, value Value) // 可选的回调函数，当缓存项被移除时调用
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// removeOldest 内部方法，调用前必须持有锁
func (c *Cache) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) RemoveOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeOldest()
}

func (c *Cache) Add(key string, value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.removeOldest()
	}
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.ll.Len() > 0 {
		c.removeOldest()
	}
}
