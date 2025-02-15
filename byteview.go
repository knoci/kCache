package main

// ByteView 是一个不可变的字节视图结构。
type ByteView struct {
	b []byte // 存储字节数据
}

// Len 返回视图的长度。
func (v ByteView) Len() int {
	return len(v.b) // 直接返回底层字节切片的长度
}

// ByteSlice 返回数据的一个副本，作为字节切片。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b) // 调用 cloneBytes 函数复制字节数据
}

// String 将数据作为字符串返回，如果需要，会创建一个副本。
func (v ByteView) String() string {
	return string(v.b) // 将字节切片转换为字符串
}

// cloneBytes 复制一个字节切片，返回其副本。
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b)) // 创建一个与原切片等长的新切片
	copy(c, b)                // 将原切片的内容复制到新切片
	return c                  // 返回副本
}
