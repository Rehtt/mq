package main

import (
	"encoding/json"
)

type ConfDB struct {
	Key   string `gorm:"type:varchar(255);primaryKey;column:key;not null"`
	Value string `gorm:"type:text;column:value;not null"`
}

func (ConfDB) TableName() string {
	return "s_conf"
}

type Conf struct {
	TlsPublic  string `json:"tls_public"`
	TlsPrivate string `json:"tls_private"`
}

var conf = new(Conf)

func (Conf) TableName() string {
	return ""
}

func (c *Conf) ReadDB() error {
	var tmp []*ConfDB
	if err := db.Find(&tmp).Error; err != nil {
		return err
	}
	tmpMap := make(map[string]string, len(tmp))
	for _, v := range tmp {
		tmpMap[v.Key] = v.Value
	}

	raw, _ := json.Marshal(tmpMap)

	return json.Unmarshal(raw, c)
}

func (c *Conf) WriteDB() error {
	raw, _ := json.Marshal(c)
	var tmpMap map[string]string
	err := json.Unmarshal(raw, &tmpMap)
	if err != nil {
		return err
	}
	var tmp ConfDB
	for k, v := range tmpMap {
		tmp.Key = k
		tmp.Value = v
		if err := db.Save(&tmp).Error; err != nil {
			return err
		}
	}

	return nil
}
