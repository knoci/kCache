package main

import pb "kCache/proto"

// PeerPicker 是一个接口，用于定位拥有特定键的对等节点（peer）。
// 实现该接口的类型需要提供一个方法来选择对等节点。
type PeerPicker interface {
	// PickPeer 方法根据给定的键选择一个对等节点。
	// 如果找到合适的对等节点，则返回该节点的 PeerGetter 接口实例和 true；
	// 否则返回 nil 和 false。
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是一个接口，表示对等节点（peer）的功能。
// 实现该接口的类型需要提供一个方法来从对等节点获取数据。
type PeerGetter interface {
	// Get 方法从对等节点获取指定分组和键的值。
	// 如果成功获取数据，返回字节切片；否则返回错误。
	Get(in *pb.Request, out *pb.Response) ([]byte, error)
}
