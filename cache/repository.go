package cache

import "time"

// CacheRepository 缓存仓库接口
// 提供带缓存的数据访问能力
type CacheRepository[T any, ID comparable] interface {
	// GetByID 根据ID获取实体（带缓存）
	GetByID(id ID) (*T, error)

	// GetAll 获取所有实体（带缓存）
	GetAll() ([]*T, error)

	// Save 保存实体并更新缓存
	Save(entity *T) error

	// Delete 删除实体并清除缓存
	Delete(id ID) error

	// ClearCache 清除指定实体的缓存
	ClearCache(id ID) error

	// ClearAllCache 清除所有缓存
	ClearAllCache() error
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// TableName 缓存表名
	TableName string

	// Expiration 缓存过期时间，0表示永不过期
	Expiration time.Duration

	// UseNotFoundCache 是否缓存不存在的key
	UseNotFoundCache bool
}
