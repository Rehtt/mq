package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/Rehtt/mq/definition"
)

// TestMq 测试gRPC MQ客户端
func TestMq(t *testing.T) {
	client, err := ConnectMq(context.Background(), "127.0.0.1:1234", false, "")
	if err != nil {
		t.Skip("gRPC server not available:", err)
		return
	}
	defer client.Close()

	runMqTests(t, client)
}

// runMqTests 运行完整的MQ功能测试
func runMqTests(t *testing.T, client definition.Mq) {
	mq := "test-mq"

	if err := client.DeleteMq(mq); err != nil {
		t.Fatal("DeleteMq", err)
	}
	if err := client.CreateMq(mq); err != nil {
		t.Fatal("CreateMq", err)
	}
	if id, err := client.Push(mq, "123"); err != nil {
		t.Fatal("Push", err)
	} else if id != 1 {
		t.Fatal("Push", "id", id)
	}

	if id, err := client.Push(mq, "234"); err != nil {
		t.Fatal("Push", err)
	} else if id != 2 {
		t.Fatal("Push", "id", id)
	}

	if msgs, err := client.Pop(mq, 1); err != nil {
		t.Fatal("Pop", err)
	} else {
		if len(msgs) != 1 {
			t.Fatal("Pop", "len", len(msgs))
		}
		if msgs[0].Text != "123" {
			t.Fatal("Pop", "text", msgs[0].Text)
		}
	}

	if msgs, err := client.Read(mq, 1, time.Second); err != nil {
		t.Fatal("Read", err)
	} else {
		if len(msgs) != 1 {
			t.Fatal("Read", "len", len(msgs))
		}
		if msgs[0].Text != "234" {
			t.Fatal("Read", "text", msgs[0].Text)
		}
	}
	time.Sleep(time.Second)
	if msgs, err := client.Read(mq, 1, 2*time.Second); err != nil {
		t.Fatal("Read", err)
	} else {
		if len(msgs) != 1 {
			t.Fatal("Read", "len", len(msgs))
		}
		if msgs[0].Text != "234" {
			t.Fatal("Read", "text", msgs[0].Text)
		}
	}
	client.Delete(mq, 2)
	time.Sleep(2 * time.Second)

	if msgs, err := client.Pop(mq, 1); err != nil {
		t.Fatal("Pop", err)
	} else {
		if len(msgs) != 0 {
			t.Fatal("Pop", "len", len(msgs))
		}
	}

	if id, err := client.Push(mq, "qwe"); err != nil {
		t.Fatal("Push", err)
	} else if id != 3 {
		t.Fatal("Push", "id", id)
	}

	if err := client.Active(mq, 3); err != nil {
		t.Fatal("Active", err)
	}

	if msgs, err := client.Pop(mq, 1); err != nil {
		t.Fatal("Pop", err)
	} else {
		if len(msgs) != 0 {
			t.Fatal("Pop", "len", len(msgs))
		}
	}

	client.Push(mq, "test len1")
	client.Push(mq, "test len2")
	client.Push(mq, "test len3")
	client.Pop(mq, 1)
	client.Push(mq, "test len4")
	client.Read(mq, 2, 2*time.Second)
	if l, err := client.Len(mq); err != nil {
		t.Fatal("Len", err)
	} else if l != 1 {
		t.Fatal("Len", "len1", l)
	}
	time.Sleep(3 * time.Second)
	if l, err := client.Len(mq); err != nil {
		t.Fatal("Len", err)
	} else if l != 3 {
		t.Fatal("Len", "len2", l)
	}

	if err := client.SetKeyValue(mq, "key", "value", 5*time.Second); err != nil {
		t.Fatal("SetKeyValue", err)
	}

	if value, ok, err := client.GetKeyValue(mq, "key"); err != nil {
		t.Fatal("GetKeyValue", err)
	} else if !ok {
		t.Fatal("GetKeyValue", "1ok", ok)
	} else if value.Value != "value" {
		t.Fatal("GetKeyValue", "value", value.Value)
	}
	time.Sleep(6 * time.Second)

	if _, ok, err := client.GetKeyValue(mq, "key"); err != nil {
		t.Fatal("GetKeyValue", err)
	} else if ok {
		t.Fatal("GetKeyValue", "2ok", ok)
	}

	if err := client.SetKeyValue(mq, "key", "value", 5*time.Second); err != nil {
		t.Fatal("SetKeyValue", err)
	}
	if err := client.DeleteMq(mq); err != nil {
		t.Fatal("DeleteMq", err)
	}
}

// TestMqStream 测试流式读取功能
func TestMqStream(t *testing.T) {
	client, err := ConnectMq(context.Background(), "127.0.0.1:1234", false, "")
	if err != nil {
		t.Skip("gRPC server not available:", err)
		return
	}
	defer client.Close()

	mq := "test-stream-mq"

	// 清理和准备数据
	client.DeleteMq(mq)
	if err := client.CreateMq(mq); err != nil {
		t.Fatal("CreateMq", err)
	}

	// 推送测试消息
	id1, _ := client.Push(mq, "stream-message-1")
	id2, _ := client.Push(mq, "stream-message-2")
	id3, _ := client.Push(mq, "stream-message-3")

	// 测试流式读取
	msgChan, err := client.ReadByStream(mq, 3, 5*time.Second)
	if err != nil {
		t.Fatal("ReadByStream", err)
	}

	// 收集从流中读取的消息
	var receivedMessages []definition.Msg
	for msg := range msgChan {
		receivedMessages = append(receivedMessages, msg)
	}

	// 验证消息数量
	if len(receivedMessages) != 3 {
		t.Fatalf("expected 3 messages from stream, got %d", len(receivedMessages))
	}

	// 验证消息内容
	expectedIds := []uint64{id1, id2, id3}
	expectedTexts := []string{"stream-message-1", "stream-message-2", "stream-message-3"}

	for i, msg := range receivedMessages {
		if msg.Id != expectedIds[i] {
			t.Errorf("expected message %d id %d, got %d", i, expectedIds[i], msg.Id)
		}
		if msg.Text != expectedTexts[i] {
			t.Errorf("expected message %d text '%s', got '%s'", i, expectedTexts[i], msg.Text)
		}
	}

	// 清理
	if err := client.DeleteMq(mq); err != nil {
		t.Fatal("DeleteMq", err)
	}
}
