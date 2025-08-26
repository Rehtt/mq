package mq

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkService_Push 测试推送消息的性能
func BenchmarkService_Push(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Push(queueName, fmt.Sprintf("message-%d", i))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
	}
}

// BenchmarkService_Push_Concurrent 测试并发推送消息的性能
func BenchmarkService_Push_Concurrent(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 使用CPU核数的goroutine
	numGoroutines := runtime.NumCPU()
	messagesPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	var wg sync.WaitGroup
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < messagesPerGoroutine; i++ {
				_, err := service.Push(queueName, fmt.Sprintf("message-%d-%d", goroutineID, i))
				if err != nil {
					b.Errorf("Push failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

// BenchmarkService_Pop 测试弹出消息的性能
func BenchmarkService_Pop(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 预先填充消息
	for i := 0; i < b.N; i++ {
		service.Push(queueName, fmt.Sprintf("message-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msgs, err := service.Pop(queueName, 1)
		if err != nil {
			b.Fatalf("Pop failed: %v", err)
		}
		if len(msgs) != 1 {
			b.Fatalf("Expected 1 message, got %d", len(msgs))
		}
	}
}

// BenchmarkService_Read 测试读取消息的性能
func BenchmarkService_Read(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 预先填充消息
	numMessages := 1000
	for i := 0; i < numMessages; i++ {
		service.Push(queueName, fmt.Sprintf("message-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Read(queueName, 10, time.Second)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

// BenchmarkService_Len 测试获取队列长度的性能
func BenchmarkService_Len(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 预先填充消息
	numMessages := 1000
	for i := 0; i < numMessages; i++ {
		service.Push(queueName, fmt.Sprintf("message-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Len(queueName)
		if err != nil {
			b.Fatalf("Len failed: %v", err)
		}
	}
}

// BenchmarkService_SetKeyValue 测试设置键值对的性能
func BenchmarkService_SetKeyValue(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := service.SetKeyValue(queueName, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), 0)
		if err != nil {
			b.Fatalf("SetKeyValue failed: %v", err)
		}
	}
}

// BenchmarkService_GetKeyValue 测试获取键值对的性能
func BenchmarkService_GetKeyValue(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 预先填充键值对
	numKVPairs := 1000
	for i := 0; i < numKVPairs; i++ {
		service.SetKeyValue(queueName, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), 0)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := service.GetKeyValue(queueName, fmt.Sprintf("key-%d", i%numKVPairs))
		if err != nil {
			b.Fatalf("GetKeyValue failed: %v", err)
		}
	}
}

// BenchmarkService_Mixed 测试混合操作的性能
func BenchmarkService_Mixed(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 推送消息
		if i%4 == 0 {
			service.Push(queueName, fmt.Sprintf("message-%d", i))
		}

		// 弹出消息
		if i%4 == 1 {
			service.Pop(queueName, 1)
		}

		// 设置键值对
		if i%4 == 2 {
			service.SetKeyValue(queueName, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), 0)
		}

		// 获取键值对
		if i%4 == 3 {
			service.GetKeyValue(queueName, fmt.Sprintf("key-%d", i-1))
		}
	}
}

// BenchmarkService_ManyQueues 测试多队列操作的性能
func BenchmarkService_ManyQueues(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	numQueues := 100

	// 创建多个队列
	for i := 0; i < numQueues; i++ {
		service.CreateMq(fmt.Sprintf("queue-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queueName := fmt.Sprintf("queue-%d", i%numQueues)
		_, err := service.Push(queueName, fmt.Sprintf("message-%d", i))
		if err != nil {
			b.Fatalf("Push to queue %s failed: %v", queueName, err)
		}
	}
}

// BenchmarkService_LargeMessages 测试大消息的性能
func BenchmarkService_LargeMessages(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	// 创建大消息 (1KB)
	largeMessage := string(make([]byte, 1024))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Push(queueName, largeMessage)
		if err != nil {
			b.Fatalf("Push large message failed: %v", err)
		}
	}
}

// BenchmarkService_Memory 内存使用基准测试
func BenchmarkService_Memory(b *testing.B) {
	repo := NewMockRepository()
	service := NewService(repo)
	queueName := "bench-queue"

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.Push(queueName, fmt.Sprintf("message-%d", i))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
	}
}
