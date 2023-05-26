package store

import "time"

type Key struct {
	ID             uint      `gorm:"primarykey" json:"id,omitempty"`
	Key            string    `gorm:"unique;not null" json:"key,omitempty"`
	Name           string    `gorm:"unique;not null" json:"name,omitempty"`
	UserId         string    `json:"-,omitempty"`
	ApiType        string    `gorm:"column:api_type"`
	EndPoint       string    `gorm:"column:endpoint"`
	DeploymentName string    `gorm:"column:deployment_name"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

func GetKeyrByName(name string) (*Key, error) {
	var key Key
	result := db.First(&key, "name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}
	return &key, nil
}

func GetAllKeys() ([]Key, error) {
	var keys []Key
	if err := db.Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// 添加记录
func AddKey(apikey, name string) error {
	key := Key{
		Key:  apikey,
		Name: name,
	}
	if err := db.Create(&key).Error; err != nil {
		return err
	}
	LoadKeysCache()
	return nil
}

// 删除记录
func DeleteKey(id uint) error {
	if err := db.Delete(&Key{}, id).Error; err != nil {
		return err
	}
	LoadKeysCache()
	return nil
}

// 更新记录
func UpdateKey(id uint, apikey string, userId string) error {
	key := Key{
		Key:    apikey,
		UserId: userId,
	}
	if err := db.Model(&Key{}).Where("id = ?", id).Updates(key).Error; err != nil {
		return err
	}
	LoadKeysCache()
	return nil
}
