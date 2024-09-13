package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"opencatd-open/pkg/vertexai"
	"os"
	"time"

	"gorm.io/gorm"
)

func init() {
	// check vertex
	if os.Getenv("Vertex") != "" {
		vertex_auth := os.Getenv("Vertex")
		var Vertex vertexai.VertexSecretKey
		if err := json.Unmarshal([]byte(vertex_auth), &Vertex); err != nil {
			log.Fatalln(fmt.Errorf("import vertex_auth json error: %w", err))
			return
		}
		key := Key{
			ApiType:   "vertex",
			Name:      Vertex.ProjectID,
			Key:       vertex_auth,
			ApiSecret: vertex_auth,
		}
		if err := db.Table("keys").Where("name = ?", Vertex.ProjectID).Find(&key).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&key).Error; err != nil {
					log.Fatalln(fmt.Errorf("import vertex_auth json error: %w", err))
				}
			} else {
				log.Fatalln(fmt.Errorf("import vertex_auth json error: %w", err))
				return
			}
		}
	}
	LoadKeysCache()
}

type Key struct {
	ID             uint      `gorm:"primarykey" json:"id,omitempty"`
	Key            string    `gorm:"unique;not null" json:"key,omitempty"`
	Name           string    `gorm:"unique;not null" json:"name,omitempty"`
	UserId         string    `json:"-,omitempty"`
	ApiType        string    `gorm:"column:api_type"`
	EndPoint       string    `gorm:"column:endpoint"`
	ResourceNmae   string    `gorm:"column:resource_name"`
	DeploymentName string    `gorm:"column:deployment_name"`
	ApiSecret      string    `gorm:"column:api_secret"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

func (k Key) ToString() string {
	bdate, _ := json.Marshal(k)
	return string(bdate)
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
func AddKey(apitype, apikey, name string) error {
	key := Key{
		ApiType: apitype,
		Key:     apikey,
		Name:    name,
	}
	if err := db.Create(&key).Error; err != nil {
		return err
	}
	LoadKeysCache()
	return nil
}

func CreateKey(k *Key) error {
	if err := db.Create(&k).Error; err != nil {
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
