package sdk

import (
	"context"
	"testing"
	"time"
)

func TestMq(t *testing.T) {
	client, err := ConnectMq(context.Background(), "127.0.0.1:1234")
	if err != nil {
		panic(err)
	}
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
}
