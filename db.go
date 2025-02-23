package main

import (
	"path/filepath"
	"sync"
	"time"

	ss "strings"

	"github.com/Rehtt/Kit/maps"
	"github.com/Rehtt/Kit/strings"
	"github.com/Rehtt/mq/definition"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Msg struct {
	Id        uint64 `gorm:"column:id;autoIncrement:false;index"`
	Text      string `gorm:"column:text;type:text"`
	Active    bool   `gorm:"column:active"`
	CreatedAt time.Time
	RetryTime *time.Time `gorm:"column:retry_time"`
}

var msgPool = sync.Pool{
	New: func() any {
		return &Msg{}
	},
}

func NewMsg() *Msg {
	msg := msgPool.Get().(*Msg)
	msg.Id = 0
	msg.Text = ""
	msg.CreatedAt = time.Time{}
	msg.Active = false
	return msg
}

type writeMqOption int

const (
	WRITE_MQ_PUSH = writeMqOption(iota)
	WRITE_MQ_DELETE
	WRITE_MQ_ACTIVE
	WRITE_MQ_UPDATE_RETRYTIME
	WRITE_MQ_DISTINCT
	WRITE_MQ_CREATE_TABLE
	WRITE_MQ_DROP_TABLE
)

const (
	MQ_TABLE_PREFIX = "mq_"
)

var db *gorm.DB

func OpenDB(workPath string) error {
	d, err := gorm.Open(sqlite.Open(filepath.Join(workPath, "db")))
	if err != nil {
		return err
	}

	// 预写日志，避免并发读写锁库
	d.Exec("pragma journal_mode = wal")
	// 不等待完全写入磁盘
	d.Exec("pragma synchronous = normal")
	// 指定将临时数据存储在内存中
	d.Exec("pragma temp_store = memory")
	// 不立即回收空间，直到手动调用pragma incremental_vacuum才回收一次空间
	d.Exec("pragma auto_vacuum = INCREMENTAL")
	// 只有单一进程使用数据库，减少系统文件锁的使用
	// d.Exec("pragma locking_mode = EXCLUSIVE")

	// 空间回收
	d.Exec("pragma incremental_vacuum")
	db = d
	return d.AutoMigrate(
		&ConfDB{},
	)
}

func CloseDB() error {
	d, _ := db.DB()
	return d.Close()
}

type writeMqNode struct {
	option    writeMqOption
	mq        string
	text      string
	retryTime *time.Time
	ids       []uint64
}

var (
	writeMqChan     = make(chan *writeMqNode, 100)
	writeMqOnce     sync.Once
	writeMqNodePool = sync.Pool{
		New: func() any {
			return &writeMqNode{}
		},
	}
)

func writeMq(option writeMqOption, mq string, text string, retryTime *time.Time, ids ...uint64) {
	node := writeMqNodePool.Get().(*writeMqNode)
	node.option = option
	node.mq = MQ_TABLE_PREFIX + mq
	node.text = text
	node.retryTime = retryTime
	node.ids = ids
	writeMqChan <- node

	writeMqOnce.Do(func() {
		go func() {
			var deleteNum int

			for {
				node := <-writeMqChan
				handleWriteMq(node)
				writeMqNodePool.Put(node)

				if node.option == WRITE_MQ_DELETE {
					deleteNum++
					if deleteNum > 100 {
						// 执行空间回收
						db.Exec("pragma incremental_vacuum")
						deleteNum = 0
					}
				}

			}
		}()
	})
}

// 分离队列操作与数据库的写入
func handleWriteMq(node *writeMqNode) {
	id := strings.JoinToString(node.ids, ",")
	mq := node.mq
	msg := NewMsg()
	defer msgPool.Put(msg)

	switch node.option {
	case WRITE_MQ_PUSH:
		msg.Id = node.ids[0]
		msg.Text = node.text
		db.Table(mq).Create(msg)
	case WRITE_MQ_DELETE:
		db.Table(mq).Where("id in (?)", id).Delete(msg)
	case WRITE_MQ_ACTIVE:
		db.Table(mq).Where("id in (?)", id).Update("active", true)
	case WRITE_MQ_DISTINCT:
		db.Table(mq).Distinct(msg)
	case WRITE_MQ_CREATE_TABLE:
		db.Table(mq).Migrator().CreateTable(msg)
	case WRITE_MQ_DROP_TABLE:
		db.Table(mq).Migrator().DropTable(msg)
	case WRITE_MQ_UPDATE_RETRYTIME:
		db.Table(mq).Where("id in (?)", id).Update("retry_time", node.retryTime)
	}
}

func getAllMqTableNames() []string {
	var names []string
	db.Table("sqlite_master").Where("type = 'table' AND name LIKE ?", MQ_TABLE_PREFIX+"%").
		Select("name").Pluck("name", &names)
	return names
}

func findAllMqToMaps() *maps.ConcurrentMap[*MqMsg] {
	m := maps.NewConcurrentMap[*MqMsg]()
	for _, name := range getAllMqTableNames() {
		var tmp []*Msg
		db.Table(name).Order("id").Find(&tmp)

		mq := &MqMsg{}
		var indexNode *MqMsgNode
		for _, value := range tmp {
			node := &MqMsgNode{
				Msg: definition.Msg{
					Id:        value.Id,
					Text:      value.Text,
					CreatedAt: value.CreatedAt,
				},
				RetryTime: value.RetryTime,
			}
			if indexNode != nil {
				indexNode.nextNode = node
			} else {
				mq.headNode = node
			}
			mq.index = value.Id
			indexNode = node
		}
		mq.footNode = indexNode
		m.Set(ss.TrimLeft(name, MQ_TABLE_PREFIX), mq)
	}
	return m
}
