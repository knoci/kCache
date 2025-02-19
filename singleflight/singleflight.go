package singleflight

import "sync"

// call 用于存储函数调用的结果。
type call struct {
	wg  sync.WaitGroup // 用于同步等待函数执行完成
	val interface{}    // 函数返回的值
	err error          // 函数执行过程中可能发生的错误
}

// Group 是一个并发控制结构体，确保同一个 key 的函数只执行一次。
type Group struct {
	mu sync.Mutex       // 保护 map 的互斥锁
	m  map[string]*call // 存储 key 和对应的 call 实例
}

// Do 方法确保同一个 key 的函数 fn 只会被执行一次。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call) // 初始化 map
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()       // 如果 key 已存在，释放锁
		c.wg.Wait()         // 等待函数执行完成
		return c.val, c.err // 返回已缓存的结果
	}
	c := new(call) // 创建一个新的 call 实例
	c.wg.Add(1)    // 增加 WaitGroup 的计数
	g.m[key] = c   // 将 call 实例存储到 map 中
	g.mu.Unlock()  // 释放锁

	// 执行函数 fn 并存储结果
	c.val, c.err = fn()
	c.wg.Done() // 函数执行完成，减少 WaitGroup 的计数

	g.mu.Lock()      // 再次加锁
	delete(g.m, key) // 删除 map 中的 key
	g.mu.Unlock()    // 释放锁

	return c.val, c.err // 返回函数的结果
}
