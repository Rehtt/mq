package mq

import (
	"sync"
	"time"

	"github.com/Rehtt/Kit/maps"
	"github.com/Rehtt/mq/definition"
	"github.com/Rehtt/mq/internal/errors"
)

// MqMsgNode 消息节点
type MqMsgNode struct {
	definition.Msg            // 消息
	RetryTime      *time.Time // 重发时间
	nextNode       *MqMsgNode
}

// MqMsg 队列消息管理
type MqMsg struct {
	index    uint64
	headNode *MqMsgNode
	footNode *MqMsgNode
	mutex    sync.RWMutex // 添加读写锁保护
}

// Repository 数据存储接口
type Repository interface {
	// MQ相关操作
	CreateMqTable(mq string) error
	DropMqTable(mq string) error
	PushMessage(mq string, msg string, id uint64) error
	DeleteMessages(mq string, ids ...uint64) error
	ActiveMessage(mq string, id uint64) error
	UpdateRetryTime(mq string, retryTime *time.Time, ids ...uint64) error

	// 键值对操作
	SetKeyValue(mq string, key string, value *definition.Value) error
	GetKeyValue(mq string, key string) (*definition.Value, bool, error)
	DeleteKeyValue(mq string, key string) error

	// 数据加载
	LoadAllMQ() *maps.ConcurrentMap[*MqMsg]
	LoadAllKV() *maps.ConcurrentMap[*definition.Value]
}

// Service MQ服务
type Service struct {
	list     *maps.ConcurrentMap[*MqMsg]
	keyValue *maps.ConcurrentMap[*definition.Value]
	repo     Repository
}

var _ definition.Mq = (*Service)(nil)

// NewService 创建MQ服务
func NewService(repo Repository) *Service {
	return &Service{
		list:     repo.LoadAllMQ(),
		keyValue: repo.LoadAllKV(),
		repo:     repo,
	}
}

// CreateMq 创建队列
func (s *Service) CreateMq(mq string) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}

	if _, ok := s.list.Get(mq); ok {
		return errors.AlreadyExists("queue already exists")
	}

	s.list.Set(mq, &MqMsg{})

	if err := s.repo.CreateMqTable(mq); err != nil {
		// 回滚内存操作
		s.list.Delete(mq)
		return errors.Wrap(err, "failed to create queue table")
	}

	return nil
}

// DeleteMq 删除队列
func (s *Service) DeleteMq(mq string) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}

	s.list.Delete(mq)

	if err := s.repo.DropMqTable(mq); err != nil {
		return errors.Wrap(err, "failed to drop queue table")
	}

	return nil
}

// Push 向队列里添加消息
func (s *Service) Push(mq string, msg string) (uint64, error) {
	if mq == "" {
		return 0, errors.InvalidInput("queue name cannot be empty")
	}
	if msg == "" {
		return 0, errors.InvalidInput("message cannot be empty")
	}

	var messageId uint64
	id := s.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		// 创建队列（如果不存在）
		if value == nil {
			value = &MqMsg{}
			s.repo.CreateMqTable(mq) // 最佳努力，忽略错误
		}

		value.mutex.Lock()
		defer value.mutex.Unlock()

		value.index++
		messageId = value.index

		node := &MqMsgNode{
			Msg: definition.Msg{
				Id:        messageId,
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

		return value
	}).index

	// 异步持久化
	go func() {
		if err := s.repo.PushMessage(mq, msg, messageId); err != nil {
			// 记录日志，但不影响返回结果
			// TODO: 添加日志记录
		}
	}()

	return id, nil
}

// Read 读取指定条数消息，并设置超时时间
func (s *Service) Read(mq string, num int, timeout time.Duration) ([]definition.Msg, error) {
	if mq == "" {
		return nil, errors.InvalidInput("queue name cannot be empty")
	}
	if num <= 0 {
		return nil, errors.InvalidInput("number must be positive")
	}

	msgs := make([]definition.Msg, 0, num)

	s.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		value.mutex.Lock()
		defer value.mutex.Unlock()

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

		// 异步更新重试时间
		if len(ids) > 0 {
			go s.repo.UpdateRetryTime(mq, &retryTime, ids...)
		}

		return value
	})

	return msgs, nil
}

// Pop 从队列中读取指定条数消息并删除
func (s *Service) Pop(mq string, num int) ([]definition.Msg, error) {
	if mq == "" {
		return nil, errors.InvalidInput("queue name cannot be empty")
	}
	if num <= 0 {
		return nil, errors.InvalidInput("number must be positive")
	}

	msgs := make([]definition.Msg, 0, num)

	s.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		value.mutex.Lock()
		defer value.mutex.Unlock()

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

		// 异步删除消息
		if len(ids) > 0 {
			go s.repo.DeleteMessages(mq, ids...)
		}

		return value
	})

	return msgs, nil
}

// Delete 从队列中删除指定消息
func (s *Service) Delete(mq string, id uint64) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}
	if id == 0 {
		return errors.InvalidInput("message id cannot be zero")
	}

	found := false
	s.list.SetByFunc(mq, func(value *MqMsg) *MqMsg {
		if value == nil {
			return value
		}

		value.mutex.Lock()
		defer value.mutex.Unlock()

		var pre *MqMsgNode
		for node := value.headNode; node != nil; node = node.nextNode {
			if node.Id != id {
				pre = node
				continue
			}

			found = true
			if pre != nil {
				pre.nextNode = node.nextNode
			}
			if node == value.headNode {
				value.headNode = node.nextNode
			}
			if node == value.footNode {
				value.footNode = pre
			}
			break
		}

		return value
	})

	if !found {
		return errors.NotFound("message not found")
	}

	// 异步删除
	go s.repo.DeleteMessages(mq, id)

	return nil
}

// Drop 清空队列
func (s *Service) Drop(mq string) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}

	s.list.Delete(mq)

	if err := s.repo.DropMqTable(mq); err != nil {
		return errors.Wrap(err, "failed to drop queue")
	}

	return nil
}

// Active 将消息存档
func (s *Service) Active(mq string, id uint64) error {
	if err := s.Delete(mq, id); err != nil {
		return err
	}

	// 异步存档
	go s.repo.ActiveMessage(mq, id)

	return nil
}

// Len 获取队列长度
func (s *Service) Len(mq string) (int, error) {
	if mq == "" {
		return 0, errors.InvalidInput("queue name cannot be empty")
	}

	value, ok := s.list.Get(mq)
	if !ok || value == nil {
		return 0, nil
	}

	value.mutex.RLock()
	defer value.mutex.RUnlock()

	var count int
	for node := value.headNode; node != nil; node = node.nextNode {
		if t := node.RetryTime; t != nil && time.Since(*t) < 0 {
			continue
		}
		count++
		if node == value.footNode {
			break
		}
	}

	return count, nil
}

// SetKeyValue 设置键值对
func (s *Service) SetKeyValue(mq string, key string, value string, expire time.Duration) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}
	if key == "" {
		return errors.InvalidInput("key cannot be empty")
	}

	var exp time.Time
	if expire > 0 {
		exp = time.Now().Add(expire)
	}

	kvKey := genKeyValueKey(mq, key)
	kv := &definition.Value{
		Value:     value,
		UpdatedAt: time.Now(),
		ExpireAt:  exp,
	}

	s.keyValue.SetByFunc(kvKey, func(oldValue *definition.Value) *definition.Value {
		return kv
	}, expire)

	// 异步持久化
	go s.repo.SetKeyValue(mq, key, kv)

	return nil
}

func (s *Service) GetKeyValue(mq string, key string) (*definition.Value, bool, error) {
	if mq == "" {
		return nil, false, errors.InvalidInput("queue name cannot be empty")
	}
	if key == "" {
		return nil, false, errors.InvalidInput("key cannot be empty")
	}

	kvKey := genKeyValueKey(mq, key)
	value, ok := s.keyValue.Get(kvKey)
	if !ok {
		return nil, false, nil
	}

	// 检查是否过期
	if !value.ExpireAt.IsZero() && time.Since(value.ExpireAt) > 0 {
		s.keyValue.Delete(kvKey)
		go s.repo.DeleteKeyValue(mq, key)
		return nil, false, nil
	}

	return value, true, nil
}

func (s *Service) DeleteKeyValue(mq string, key string) error {
	if mq == "" {
		return errors.InvalidInput("queue name cannot be empty")
	}
	if key == "" {
		return errors.InvalidInput("key cannot be empty")
	}

	kvKey := genKeyValueKey(mq, key)
	s.keyValue.Delete(kvKey)

	// 异步删除
	go s.repo.DeleteKeyValue(mq, key)

	return nil
}

func genKeyValueKey(mq string, key string) string {
	return "keyvalue:" + mq + ":" + key
}
