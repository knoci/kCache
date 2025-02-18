package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 是一个哈希函数接口，将字节数组映射为 uint32。
type Hash func(data []byte) uint32

// Map 包含所有哈希后的键，并支持一致性哈希。
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个键的副本数量
	keys     []int          // 排序后的哈希值列表
	hashMap  map[int]string // 哈希值到键的映射
}

// New 创建一个新的 Map 实例。
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,             // 设置副本数量
		hash:     fn,                   // 设置哈希函数
		hashMap:  make(map[int]string), // 初始化哈希映射
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE // 默认使用 CRC32 哈希函数
	}
	return m
}

// Add 将多个键添加到一致性哈希环中。
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) // 为每个副本生成哈希值
			m.keys = append(m.keys, hash)                      // 将哈希值加入列表
			m.hashMap[hash] = key                              // 将哈希值与键关联
		}
	}
	sort.Ints(m.keys) // 对哈希值进行排序
}

// Get 获取与给定键最接近的哈希值对应的键。
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return "" // 如果哈希环为空，返回空字符串
	}

	hash := int(m.hash([]byte(key))) // 计算目标键的哈希值
	// 使用二分查找找到合适的副本位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 返回最接近的键
	return m.hashMap[m.keys[idx%len(m.keys)]] // 如果超出范围，循环回到列表开头
}
