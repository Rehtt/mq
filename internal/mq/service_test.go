package mq

import (
	"testing"
	"time"

	"github.com/Rehtt/Kit/maps"
	"github.com/Rehtt/mq/definition"
	"github.com/Rehtt/mq/internal/errors"
)

// MockRepository 模拟数据库仓库
type MockRepository struct {
	createMqTableCalled   []string
	dropMqTableCalled     []string
	pushMessageCalled     []PushCall
	deleteMessagesCalled  []DeleteCall
	activeMessageCalled   []ActiveCall
	updateRetryTimeCalled []UpdateRetryTimeCall
	setKeyValueCalled     []SetKVCall
	getKeyValueCalled     []string
	deleteKeyValueCalled  []string

	// 可配置的返回值
	createMqTableError   error
	dropMqTableError     error
	pushMessageError     error
	deleteMessagesError  error
	activeMessageError   error
	updateRetryTimeError error
	setKeyValueError     error
	getKeyValueResult    *definition.Value
	getKeyValueExists    bool
	getKeyValueError     error
	deleteKeyValueError  error

	loadAllMQResult *maps.ConcurrentMap[*MqMsg]
	loadAllKVResult *maps.ConcurrentMap[*definition.Value]
}

type PushCall struct {
	Mq   string
	Text string
	Id   uint64
}

type DeleteCall struct {
	Mq  string
	Ids []uint64
}

type ActiveCall struct {
	Mq string
	Id uint64
}

type UpdateRetryTimeCall struct {
	Mq        string
	RetryTime *time.Time
	Ids       []uint64
}

type SetKVCall struct {
	Mq    string
	Key   string
	Value *definition.Value
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		loadAllMQResult: maps.NewConcurrentMap[*MqMsg](),
		loadAllKVResult: maps.NewConcurrentMap[*definition.Value](),
	}
}

func (m *MockRepository) CreateMqTable(mq string) error {
	m.createMqTableCalled = append(m.createMqTableCalled, mq)
	return m.createMqTableError
}

func (m *MockRepository) DropMqTable(mq string) error {
	m.dropMqTableCalled = append(m.dropMqTableCalled, mq)
	return m.dropMqTableError
}

func (m *MockRepository) PushMessage(mq string, text string, id uint64) error {
	m.pushMessageCalled = append(m.pushMessageCalled, PushCall{Mq: mq, Text: text, Id: id})
	return m.pushMessageError
}

func (m *MockRepository) DeleteMessages(mq string, ids ...uint64) error {
	m.deleteMessagesCalled = append(m.deleteMessagesCalled, DeleteCall{Mq: mq, Ids: ids})
	return m.deleteMessagesError
}

func (m *MockRepository) ActiveMessage(mq string, id uint64) error {
	m.activeMessageCalled = append(m.activeMessageCalled, ActiveCall{Mq: mq, Id: id})
	return m.activeMessageError
}

func (m *MockRepository) UpdateRetryTime(mq string, retryTime *time.Time, ids ...uint64) error {
	m.updateRetryTimeCalled = append(m.updateRetryTimeCalled, UpdateRetryTimeCall{
		Mq: mq, RetryTime: retryTime, Ids: ids,
	})
	return m.updateRetryTimeError
}

func (m *MockRepository) SetKeyValue(mq string, key string, value *definition.Value) error {
	m.setKeyValueCalled = append(m.setKeyValueCalled, SetKVCall{Mq: mq, Key: key, Value: value})
	return m.setKeyValueError
}

func (m *MockRepository) GetKeyValue(mq string, key string) (*definition.Value, bool, error) {
	m.getKeyValueCalled = append(m.getKeyValueCalled, genKeyValueKey(mq, key))
	return m.getKeyValueResult, m.getKeyValueExists, m.getKeyValueError
}

func (m *MockRepository) DeleteKeyValue(mq string, key string) error {
	m.deleteKeyValueCalled = append(m.deleteKeyValueCalled, genKeyValueKey(mq, key))
	return m.deleteKeyValueError
}

func (m *MockRepository) LoadAllMQ() *maps.ConcurrentMap[*MqMsg] {
	return m.loadAllMQResult
}

func (m *MockRepository) LoadAllKV() *maps.ConcurrentMap[*definition.Value] {
	return m.loadAllKVResult
}

func TestService_CreateMq(t *testing.T) {
	tests := []struct {
		name      string
		queueName string
		repoError error
		wantError bool
		errorType errors.ErrorType
	}{
		{
			name:      "successful creation",
			queueName: "test-queue",
			wantError: false,
		},
		{
			name:      "empty queue name",
			queueName: "",
			wantError: true,
			errorType: errors.ErrorTypeInvalidInput,
		},
		{
			name:      "queue already exists",
			queueName: "existing-queue",
			wantError: true,
			errorType: errors.ErrorTypeAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository()
			repo.createMqTableError = tt.repoError
			service := NewService(repo)

			// 为"已存在队列"测试预先添加队列
			if tt.name == "queue already exists" {
				service.list.Set("existing-queue", &MqMsg{})
			}

			err := service.CreateMq(tt.queueName)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				if be, ok := err.(*errors.BusinessError); ok {
					if be.Type != tt.errorType {
						t.Errorf("expected error type %v, got %v", tt.errorType, be.Type)
					}
				} else {
					t.Errorf("expected BusinessError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// 验证队列已创建
				if _, ok := service.list.Get(tt.queueName); !ok {
					t.Errorf("queue not created in memory")
				}

				// 验证仓库方法被调用
				if len(repo.createMqTableCalled) != 1 || repo.createMqTableCalled[0] != tt.queueName {
					t.Errorf("CreateMqTable not called correctly")
				}
			}
		})
	}
}

func TestService_Push(t *testing.T) {
	tests := []struct {
		name      string
		queueName string
		message   string
		wantError bool
		errorType errors.ErrorType
	}{
		{
			name:      "successful push",
			queueName: "test-queue",
			message:   "test message",
			wantError: false,
		},
		{
			name:      "empty queue name",
			queueName: "",
			message:   "test message",
			wantError: true,
			errorType: errors.ErrorTypeInvalidInput,
		},
		{
			name:      "empty message",
			queueName: "test-queue",
			message:   "",
			wantError: true,
			errorType: errors.ErrorTypeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository()
			service := NewService(repo)

			id, err := service.Push(tt.queueName, tt.message)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				if be, ok := err.(*errors.BusinessError); ok {
					if be.Type != tt.errorType {
						t.Errorf("expected error type %v, got %v", tt.errorType, be.Type)
					}
				} else {
					t.Errorf("expected BusinessError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if id == 0 {
					t.Errorf("expected non-zero message id")
				}
			}
		})
	}
}

func TestService_Pop(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo)

	// 准备测试数据
	queueName := "test-queue"

	// Push一些消息
	id1, _ := service.Push(queueName, "message1")
	id2, _ := service.Push(queueName, "message2")
	id3, _ := service.Push(queueName, "message3")

	t.Run("pop messages", func(t *testing.T) {
		msgs, err := service.Pop(queueName, 2)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(msgs) != 2 {
			t.Errorf("expected 2 messages, got %d", len(msgs))
		}

		if msgs[0].Id != id1 {
			t.Errorf("expected first message id %d, got %d", id1, msgs[0].Id)
		}

		if msgs[1].Id != id2 {
			t.Errorf("expected second message id %d, got %d", id2, msgs[1].Id)
		}
	})

	t.Run("pop remaining messages", func(t *testing.T) {
		msgs, err := service.Pop(queueName, 5) // 请求比剩余多的数量
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(msgs) != 1 {
			t.Errorf("expected 1 message, got %d", len(msgs))
		}

		if msgs[0].Id != id3 {
			t.Errorf("expected message id %d, got %d", id3, msgs[0].Id)
		}
	})

	t.Run("pop from empty queue", func(t *testing.T) {
		msgs, err := service.Pop(queueName, 1)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(msgs) != 0 {
			t.Errorf("expected 0 messages, got %d", len(msgs))
		}
	})
}

func TestService_Read(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo)

	queueName := "test-queue"

	// Push一些消息
	id1, _ := service.Push(queueName, "message1")
	id2, _ := service.Push(queueName, "message2")

	t.Run("read messages with timeout", func(t *testing.T) {
		msgs, err := service.Read(queueName, 2, time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(msgs) != 2 {
			t.Errorf("expected 2 messages, got %d", len(msgs))
		}

		if msgs[0].Id != id1 {
			t.Errorf("expected first message id %d, got %d", id1, msgs[0].Id)
		}

		if msgs[1].Id != id2 {
			t.Errorf("expected second message id %d, got %d", id2, msgs[1].Id)
		}
	})

	t.Run("read same messages again immediately", func(t *testing.T) {
		// 应该返回空，因为消息还在重试期内
		msgs, err := service.Read(queueName, 2, time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(msgs) != 0 {
			t.Errorf("expected 0 messages (in retry period), got %d", len(msgs))
		}
	})
}

func TestService_Delete(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo)

	queueName := "test-queue"

	// Push消息
	id, _ := service.Push(queueName, "message to delete")

	t.Run("delete existing message", func(t *testing.T) {
		err := service.Delete(queueName, id)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("delete non-existing message", func(t *testing.T) {
		err := service.Delete(queueName, 999)
		if err == nil {
			t.Errorf("expected error for non-existing message")
			return
		}

		if be, ok := err.(*errors.BusinessError); ok {
			if be.Type != errors.ErrorTypeNotFound {
				t.Errorf("expected NotFound error, got %v", be.Type)
			}
		} else {
			t.Errorf("expected BusinessError, got %T", err)
		}
	})
}

func TestService_Len(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo)

	queueName := "test-queue"

	t.Run("empty queue length", func(t *testing.T) {
		length, err := service.Len(queueName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if length != 0 {
			t.Errorf("expected length 0, got %d", length)
		}
	})

	t.Run("queue with messages", func(t *testing.T) {
		// Push一些消息
		service.Push(queueName, "message1")
		service.Push(queueName, "message2")
		service.Push(queueName, "message3")

		length, err := service.Len(queueName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if length != 3 {
			t.Errorf("expected length 3, got %d", length)
		}
	})
}

func TestService_KeyValue(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo)

	queueName := "test-queue"
	key := "test-key"
	value := "test-value"

	t.Run("set and get key-value", func(t *testing.T) {
		// Set
		err := service.SetKeyValue(queueName, key, value, 0)
		if err != nil {
			t.Errorf("unexpected error setting key-value: %v", err)
		}

		// Get
		retrievedValue, ok, err := service.GetKeyValue(queueName, key)
		if err != nil {
			t.Errorf("unexpected error getting key-value: %v", err)
		}

		if !ok {
			t.Errorf("expected key to exist")
		}

		if retrievedValue.Value != value {
			t.Errorf("expected value %s, got %s", value, retrievedValue.Value)
		}
	})

	t.Run("get non-existing key", func(t *testing.T) {
		_, ok, err := service.GetKeyValue(queueName, "non-existing-key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if ok {
			t.Errorf("expected key to not exist")
		}
	})

	t.Run("delete key-value", func(t *testing.T) {
		err := service.DeleteKeyValue(queueName, key)
		if err != nil {
			t.Errorf("unexpected error deleting key-value: %v", err)
		}

		// Verify deletion
		_, ok, err := service.GetKeyValue(queueName, key)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if ok {
			t.Errorf("expected key to not exist after deletion")
		}
	})
}
