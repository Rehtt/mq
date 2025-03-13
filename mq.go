package main

import (
	"errors"
	"time"

	"github.com/Rehtt/Kit/maps"
	"github.com/Rehtt/mq/definition"
)

type Mq struct {
	list     *maps.ConcurrentMap[*MqMsg]
	keyValue *maps.ConcurrentMap[*definition.Value]
}

var _ definition.Mq = (*Mq)(nil)

type MqMsgNode struct {
	definition.Msg            // 消息
	RetryTime      *time.Time // 重发时间

	nextNode *MqMsgNode
}
type MqMsg struct {
	index uint64

	headNode *MqMsgNode
	footNode *MqMsgNode
}

func NewMq() *Mq {
	mq := &Mq{
		list:     findAllMqToMaps(),
		keyValue: findAllMqKVToMaps(),
	}
	return mq
}

// 创建队列
func (m *Mq) CreateMq(mq string) (err error) {
	if _, ok := m.list.Get(mq); ok {
		return errors.New("mq already exists")
	}
	m.list.Set(mq, &MqMsg{})
	writeMq(WRITE_MQ_CREATE_TABLE, mq, "", nil)
	return
}

// 向队列里添加消息
func (m *Mq) Push(mq string, msg string) (id uint64, err error) {
	id = m.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		// 创建队列
		if value == nil {
			value = &MqMsg{}
			writeMq(WRITE_MQ_CREATE_TABLE, mq, "", nil)
		}

		value.index++
		node := &MqMsgNode{
			Msg: definition.Msg{
				Id:        value.index,
				Text:      msg,
				CreatedAt: time.Now(),
			},
		}
		if foot := value.footNode; foot != nil {
			foot.nextNode = node
		}
		value.footNode = node

		if value.headNode == nil {
			value.headNode = node
		}

		writeMq(WRITE_MQ_PUSH, mq, msg, nil, node.Id)
		return value
	}).index

	return
}

// 读取指定条数消息，并设置超时时间
// 在时间范围内，消息不能再次被读取
// 如果超出时间范围没有将消息删除或归档，消息会在下次被读取
func (m *Mq) Read(mq string, num int, timeout time.Duration) (msgs []definition.Msg, err error) {
	msgs = make([]definition.Msg, 0, num)
	m.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		var (
			index     = value.headNode
			retryTime = time.Now().Add(timeout)
			ids       = make([]uint64, 0, num)
		)
		for i := 0; i < num; i++ {
			if index == nil {
				break
			}
			if t := index.RetryTime; t != nil && time.Since(*t) < 0 {
				// 还在重试等待期内，跳过
				i--
				index = index.nextNode
				continue
			}
			msgs = append(msgs, index.Msg)
			ids = append(ids, index.Msg.Id)
			index.RetryTime = &retryTime
			index = index.nextNode
		}
		writeMq(WRITE_MQ_UPDATE_RETRYTIME, mq, "", &retryTime, ids...)
		return value
	})
	return
}

// 从队列中读取指定条数消息
// 并在队列中删除消息
func (m *Mq) Pop(mq string, num int) (msgs []definition.Msg, err error) {
	msgs = make([]definition.Msg, 0, num)
	m.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		index := value.headNode
		var retryNode *MqMsgNode
		ids := make([]uint64, 0, num)
		for i := 0; i < num; i++ {
			if index == nil {
				break
			}
			if t := index.RetryTime; t != nil && time.Since(*t) < 0 {
				// 还在重试等待期内，跳过
				i--
				if retryNode != nil {
					retryNode.nextNode = index
				}
				retryNode = index
				index = index.nextNode
				continue
			}
			msgs = append(msgs, index.Msg)
			ids = append(ids, index.Msg.Id)
			index = index.nextNode
		}

		if retryNode != nil {
			retryNode.nextNode = index
			index = retryNode
		}
		value.headNode = index

		writeMq(WRITE_MQ_DELETE, mq, "", nil, ids...)
		return value
	})
	return
}

// 从队列中删除指定消息
func (m *Mq) Delete(mq string, id uint64) (err error) {
	m.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		var pre *MqMsgNode
		for node := value.headNode; node != nil; node = node.nextNode {
			if node.Id != id {
				pre = node
				continue
			}
			if pre != nil {
				pre.nextNode = node.nextNode
			}
			if node == value.headNode {
				value.headNode = node.nextNode
			}
			if node == value.footNode {
				value.footNode = pre
			}
		}
		writeMq(WRITE_MQ_DELETE, mq, "", nil, id)
		return value
	})
	return
}

// 将消息存档
func (m *Mq) Active(mq string, id uint64) (err error) {
	if err = m.Delete(mq, id); err != nil {
		return
	}
	writeMq(WRITE_MQ_ACTIVE, mq, "", nil, id)
	return
}

// 删除队列
func (m *Mq) DeleteMq(mq string) (err error) {
	m.list.Delete(mq)
	writeMq(WRITE_MQ_DROP_TABLE, mq, "", nil, 0)
	return
}

// 清空队列
func (m *Mq) Drop(mq string) (err error) {
	m.list.Delete(mq)
	writeMq(WRITE_MQ_DROP_TABLE, mq, "", nil, 0)
	return
}

// 设置键值对
func (m *Mq) SetKeyValue(mq string, key string, value string, expire time.Duration) (err error) {
	var exp time.Time
	if expire > 0 {
		exp = time.Now().Add(expire)
	}

	m.keyValue.SetByFunc(genKeyValueKey(mq, key), func(oldValue *definition.Value) *definition.Value {
		if oldValue == nil {
			oldValue = &definition.Value{}
		}
		oldValue.Value = value
		oldValue.UpdatedAt = time.Now()
		oldValue.ExpireAt = exp
		writeMqKV(mq, key, oldValue)
		return oldValue
	}, expire)
	return
}

func (m *Mq) GetKeyValue(mq string, key string) (value *definition.Value, ok bool, err error) {
	value, ok = m.keyValue.Get(genKeyValueKey(mq, key))
	return
}

func (m *Mq) DeleteKeyValue(mq string, key string) (err error) {
	m.keyValue.Delete(genKeyValueKey(mq, key))
	writeMqKV(mq, key, nil)
	return
}

func genKeyValueKey(mq string, key string) string {
	return "keyvalue:" + mq + ":" + key
}

func (m *Mq) Len(mq string) (int, error) {
	value, ok := m.list.Get(mq)
	if !ok || value == nil {
		return 0, nil
	}
	var out int
	for node := value.headNode; node != nil; node = node.nextNode {
		if t := node.RetryTime; t != nil && time.Since(*t) < 0 {
			continue
		}
		out++
		if node == value.footNode {
			break
		}
	}
	return out, nil
}
