package mq

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

const (
	MQ_TABLE_PREFIX        = "mq_"
	KEY_VALUE_TABLE_PREFIX = "kv_"
)

// DBMsg 数据库消息模型
type DBMsg struct {
	Id        uint64 `gorm:"column:id;autoIncrement:false;index"`
	Text      string `gorm:"column:text;type:text"`
	Active    bool   `gorm:"column:active"`
	CreatedAt time.Time
	RetryTime *time.Time `gorm:"column:retry_time"`
}

// DBKV 数据库键值对模型
type DBKV struct {
	Key       string    `gorm:"column:key;type:varchar(255);primaryKey;not null"`
	Value     string    `gorm:"column:value;type:text;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	ExpireAt  time.Time `gorm:"column:expire_at"`
}

// ConfDB 配置数据库模型
type ConfDB struct {
	Key   string `gorm:"type:varchar(255);primaryKey;column:key;not null"`
	Value string `gorm:"type:text;column:value;not null"`
}

func (ConfDB) TableName() string {
	return "s_conf"
}

type writeMqOption int

const (
	WRITE_MQ_PUSH = writeMqOption(iota)
	WRITE_MQ_DELETE
	WRITE_MQ_ACTIVE
	WRITE_MQ_UPDATE_RETRYTIME
	WRITE_MQ_CREATE_TABLE
	WRITE_MQ_DROP_TABLE
)

// SQLiteRepository SQLite数据库仓库实现
type SQLiteRepository struct {
	db              *gorm.DB
	writeMqChan     chan *writeMqNode
	writeMqOnce     sync.Once
	writeMqNodePool sync.Pool
	msgPool         sync.Pool
	kvPool          sync.Pool
}

type writeMqNode struct {
	option    writeMqOption
	mq        string
	text      string
	retryTime *time.Time
	ids       []uint64
}

// NewSQLiteRepository 创建SQLite仓库
func NewSQLiteRepository(workPath string) (*SQLiteRepository, error) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(workPath, "db")))
	if err != nil {
		return nil, err
	}

	// SQLite优化配置
	db.Exec("pragma journal_mode = wal")
	db.Exec("pragma synchronous = normal")
	db.Exec("pragma temp_store = memory")
	db.Exec("pragma auto_vacuum = INCREMENTAL")
	db.Exec("pragma incremental_vacuum")

	repo := &SQLiteRepository{
		db:              db,
		writeMqChan:     make(chan *writeMqNode, 100),
		writeMqNodePool: sync.Pool{New: func() any { return &writeMqNode{} }},
		msgPool:         sync.Pool{New: func() any { return &DBMsg{} }},
		kvPool:          sync.Pool{New: func() any { return &DBKV{} }},
	}

	// 自动迁移配置表
	if err := db.AutoMigrate(&ConfDB{}); err != nil {
		return nil, err
	}

	return repo, nil
}

// CreateMqTable 创建消息队列表
func (r *SQLiteRepository) CreateMqTable(mq string) error {
	tableName := MQ_TABLE_PREFIX + mq
	return r.db.Table(tableName).Migrator().CreateTable(&DBMsg{})
}

// DropMqTable 删除消息队列表
func (r *SQLiteRepository) DropMqTable(mq string) error {
	tableName := MQ_TABLE_PREFIX + mq
	return r.db.Table(tableName).Migrator().DropTable(&DBMsg{})
}

// PushMessage 推送消息到数据库
func (r *SQLiteRepository) PushMessage(mq string, text string, id uint64) error {
	r.writeMq(WRITE_MQ_PUSH, mq, text, nil, id)
	return nil
}

// DeleteMessages 删除消息
func (r *SQLiteRepository) DeleteMessages(mq string, ids ...uint64) error {
	r.writeMq(WRITE_MQ_DELETE, mq, "", nil, ids...)
	return nil
}

// ActiveMessage 归档消息
func (r *SQLiteRepository) ActiveMessage(mq string, id uint64) error {
	r.writeMq(WRITE_MQ_ACTIVE, mq, "", nil, id)
	return nil
}

// UpdateRetryTime 更新重试时间
func (r *SQLiteRepository) UpdateRetryTime(mq string, retryTime *time.Time, ids ...uint64) error {
	r.writeMq(WRITE_MQ_UPDATE_RETRYTIME, mq, "", retryTime, ids...)
	return nil
}

// SetKeyValue 设置键值对
func (r *SQLiteRepository) SetKeyValue(mq string, key string, value *definition.Value) error {
	// TODO: 实现KV存储
	return nil
}

// GetKeyValue 获取键值对
func (r *SQLiteRepository) GetKeyValue(mq string, key string) (*definition.Value, bool, error) {
	// TODO: 实现KV获取
	return nil, false, nil
}

// DeleteKeyValue 删除键值对
func (r *SQLiteRepository) DeleteKeyValue(mq string, key string) error {
	// TODO: 实现KV删除
	return nil
}

// LoadAllMQ 加载所有队列数据
func (r *SQLiteRepository) LoadAllMQ() *maps.ConcurrentMap[*MqMsg] {
	m := maps.NewConcurrentMap[*MqMsg]()

	tableNames := r.getAllMqTableNames()
	for _, name := range tableNames {
		var tmp []*DBMsg
		r.db.Table(name).Order("id").Find(&tmp)

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

		queueName := ss.TrimPrefix(name, MQ_TABLE_PREFIX)
		m.Set(queueName, mq)
	}

	return m
}

// LoadAllKV 加载所有键值对数据
func (r *SQLiteRepository) LoadAllKV() *maps.ConcurrentMap[*definition.Value] {
	m := maps.NewConcurrentMap[*definition.Value]()

	// TODO: 实现KV数据加载

	return m
}

// Close 关闭数据库连接
func (r *SQLiteRepository) Close() error {
	db, _ := r.db.DB()
	return db.Close()
}

// 私有方法
func (r *SQLiteRepository) writeMq(option writeMqOption, mq string, text string, retryTime *time.Time, ids ...uint64) {
	node := r.writeMqNodePool.Get().(*writeMqNode)
	node.option = option
	node.mq = MQ_TABLE_PREFIX + mq
	node.text = text
	node.retryTime = retryTime
	node.ids = ids
	r.writeMqChan <- node

	r.writeMqOnce.Do(func() {
		go func() {
			var deleteNum int
			for {
				node := <-r.writeMqChan
				r.handleWriteMq(node)
				r.writeMqNodePool.Put(node)

				if node.option == WRITE_MQ_DELETE {
					deleteNum++
					if deleteNum > 100 {
						r.db.Exec("pragma incremental_vacuum")
						deleteNum = 0
					}
				}
			}
		}()
	})
}

func (r *SQLiteRepository) handleWriteMq(node *writeMqNode) {
	id := strings.JoinToString(node.ids, ",")
	mq := node.mq
	msg := r.msgPool.Get().(*DBMsg)
	defer r.msgPool.Put(msg)

	// 重置消息对象
	*msg = DBMsg{}

	switch node.option {
	case WRITE_MQ_PUSH:
		msg.Id = node.ids[0]
		msg.Text = node.text
		msg.CreatedAt = time.Now()
		r.db.Table(mq).Create(msg)
	case WRITE_MQ_DELETE:
		r.db.Table(mq).Where("id in (?)", id).Delete(msg)
	case WRITE_MQ_ACTIVE:
		r.db.Table(mq).Where("id in (?)", id).Update("active", true)
	case WRITE_MQ_CREATE_TABLE:
		r.db.Table(mq).Migrator().CreateTable(msg)
	case WRITE_MQ_DROP_TABLE:
		r.db.Table(mq).Migrator().DropTable(msg)
	case WRITE_MQ_UPDATE_RETRYTIME:
		r.db.Table(mq).Where("id in (?)", id).Update("retry_time", node.retryTime)
	}
}

func (r *SQLiteRepository) getAllMqTableNames() []string {
	var names []string
	r.db.Table("sqlite_master").Where("type = 'table' AND name LIKE ?", MQ_TABLE_PREFIX+"%").
		Select("name").Pluck("name", &names)
	return names
}

// resetMsg 重置消息对象
func (r *SQLiteRepository) resetMsg(msg *DBMsg) {
	msg.Id = 0
	msg.Text = ""
	msg.CreatedAt = time.Time{}
	msg.Active = false
	msg.RetryTime = nil
}

// resetKV 重置KV对象
func (r *SQLiteRepository) resetKV(kv *DBKV) {
	kv.Key = ""
	kv.Value = ""
	kv.UpdatedAt = time.Time{}
	kv.ExpireAt = time.Time{}
}
