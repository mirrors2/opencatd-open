package store

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	IsDelete  bool      `gorm:"default:false" json:"IsDelete"`
	ID        uint      `gorm:"primarykey autoIncrement;" json:"id,omitempty"`
	Name      string    `gorm:"unique;not null" json:"name,omitempty"`
	Token     string    `gorm:"unique;not null" json:"token,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func CreateUser(u *User) error {
	result := db.Create(u)
	if result.Error != nil {
		return result.Error
	}
	LoadAuthCache()
	return nil
}

// 添加用户
func AddUser(name, token string) error {
	user := &User{Name: name, Token: token}
	result := db.Create(&user)
	if result.Error != nil {
		return result.Error
	}
	LoadAuthCache()
	return nil
}

// 删除用户
func DeleteUser(id uint) error {
	result := db.Delete(&User{}, id)
	if result.Error != nil {
		return result.Error
	}
	LoadAuthCache()
	return nil
}

// 修改用户
func UpdateUser(id uint, token string) error {
	user := &User{Token: token}
	result := db.Model(&User{}).Where("id = ?", id).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	LoadAuthCache()
	return nil
}

func GetUserByID(id uint) (*User, error) {
	var user User
	result := db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func GetUserByName(name string) (*User, error) {
	var user User
	result := db.Where(&User{Name: name}).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func GetUserByToken(token string) (*User, error) {
	var user User
	result := db.Where("token = ?", token).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func GetUserID(authkey string) (int, error) {
	var user User
	result := db.Where(&User{Token: authkey}).First(&user)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(user.ID), nil
}

func GetAllUsers() ([]*User, error) {
	var users []*User
	result := db.Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}
