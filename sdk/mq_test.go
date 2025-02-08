package sdk

import (
	"fmt"
	"testing"
)

func TestMq(t *testing.T) {
	client, err := ConnectMq("127.0.0.1:1234")
	if err != nil {
		panic(err)
	}
	client.DeleteMq("test112")
	client.CreateMq("test112")
	client.Push("test112", "123")
	msgs, _ := client.Pop("test112", 1)
	fmt.Println(msgs)
}
