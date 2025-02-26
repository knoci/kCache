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

// Vlue使用Len来计算它需要多少字节
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

// Get方法从缓存中获取指定键对应的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	c.mu.Lock()         // 加锁，保证线程安全
	defer c.mu.Unlock() // 确保在函数返回时解锁

	// 查找键对应的双向链表节点
	if ele, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ele) // 将最近访问的节点移动到队首
		kv := ele.Value.(*entry)
		return kv.value, true // 返回值并标记存在
	}
	return nil, false
}

// removeOldest是内部方法，用于移除最近最少访问的节点（队尾节点）
func (c *Cache) removeOldest() {
	ele := c.ll.Back() // 获取队尾节点
	if ele != nil {
		c.ll.Remove(ele) // 从链表中移除节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                // 从缓存字典中删除键
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新已使用字节数

		// 如果有淘汰回调，则调用
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// RemoveOldest是公开方法，用于移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	c.mu.Lock() // 加锁
	defer c.mu.Unlock()
	c.removeOldest() // 调用内部方法
}

// Add方法将键值对添加到缓存中
func (c *Cache) Add(key string, value Value) {
	c.mu.Lock() // 加锁
	defer c.mu.Unlock()

	// 如果键已存在，则更新值
	if ele, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ele) // 将节点移动到队首
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // 更新已使用字节数
		kv.value = value                                       // 更新值
	} else {
		// 键不存在，插入新节点
		ele := c.ll.PushFront(&entry{key, value})        // 将新节点插入队首
		c.cache[key] = ele                               // 将节点存入缓存字典
		c.nbytes += int64(len(key)) + int64(value.Len()) // 更新已使用字节数
	}

	// 如果设置了最大字节数并且超过限制，则移除最老的节点
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
