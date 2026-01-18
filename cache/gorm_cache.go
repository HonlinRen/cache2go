package cache

import (
	"errors"
	"fmt"
	"sync"

	"github.com/muesli/cache2go"
	"gorm.io/gorm"
)

// NotFound 缓存不存在的key的标记
type NotFound struct{}

// GormCacheRepository GORM缓存仓库实现
type GormCacheRepository[T any, ID comparable] struct {
	db         *gorm.DB
	cacheTable *cache2go.CacheTable
	config     CacheConfig
	primaryKey string // 主键字段名
	instance   sync.Pool
}

// NewGormCacheRepository 创建GORM缓存仓库
// db: GORM数据库连接
// config: 缓存配置
// primaryKey: 主键字段名（如 "id"）
func NewGormCacheRepository[T any, ID comparable](db *gorm.DB, config CacheConfig, primaryKey string) *GormCacheRepository[T, ID] {
	repo := &GormCacheRepository[T, ID]{
		db:         db,
		cacheTable: cache2go.Cache(config.TableName),
		config:     config,
		primaryKey: primaryKey,
		instance: sync.Pool{
			New: func() interface{} {
				return new(T)
			},
		},
	}

	return repo
}

// GetByID 根据ID获取实体（带缓存）
func (r *GormCacheRepository[T, ID]) GetByID(id ID) (*T, error) {
	cacheKey := r.buildCacheKey(id)

	// 尝试从缓存获取
	cachedData, err := r.cacheTable.Value(cacheKey)
	if err == nil {
		// 缓存命中
		if _, ok := cachedData.Data().(NotFound); ok {
			// 缓存中标记为不存在
			return nil, gorm.ErrRecordNotFound
		}
		return cachedData.Data().(*T), nil
	}

	// 缓存未命中，查询数据库
	entity := new(T)
	query := r.db.Where(fmt.Sprintf("%s = ?", r.primaryKey), id).First(entity)

	if query.Error != nil {
		// 如果启用了不存在的key缓存
		if r.config.UseNotFoundCache && errors.Is(query.Error, gorm.ErrRecordNotFound) {
			r.cacheTable.Add(cacheKey, r.config.Expiration, NotFound{})
		}
		return nil, query.Error
	}

	// 将查询结果写入缓存
	r.cacheTable.Add(cacheKey, r.config.Expiration, entity)

	return entity, nil
}

// GetAll 获取所有实体（带缓存）
func (r *GormCacheRepository[T, ID]) GetAll() ([]*T, error) {
	cacheKey := "all"

	// 尝试从缓存获取
	cachedData, err := r.cacheTable.Value(cacheKey)
	if err == nil {
		return cachedData.Data().([]*T), nil
	}

	// 缓存未命中，查询数据库
	var entities []*T
	query := r.db.Find(&entities)
	if query.Error != nil {
		return nil, query.Error
	}

	// 将查询结果写入缓存
	r.cacheTable.Add(cacheKey, r.config.Expiration, entities)

	return entities, nil
}

// Save 保存实体并更新缓存
func (r *GormCacheRepository[T, ID]) Save(entity *T) error {
	// 先保存到数据库
	query := r.db.Save(entity)
	if query.Error != nil {
		return query.Error
	}

	// 更新缓存
	cacheKey := r.buildCacheKeyFromEntity(entity)
	r.cacheTable.Add(cacheKey, r.config.Expiration, entity)

	// 清除全量缓存（因为全量缓存可能已过时）
	r.cacheTable.Delete("all")

	return nil
}

// Delete 删除实体并清除缓存
func (r *GormCacheRepository[T, ID]) Delete(id ID) error {
	// 先从数据库删除
	query := r.db.Where(fmt.Sprintf("%s = ?", r.primaryKey), id).Delete(new(T))
	if query.Error != nil {
		return query.Error
	}

	// 清除缓存
	cacheKey := r.buildCacheKey(id)
	r.cacheTable.Delete(cacheKey)

	// 清除全量缓存
	r.cacheTable.Delete("all")

	return nil
}

// ClearCache 清除指定实体的缓存
func (r *GormCacheRepository[T, ID]) ClearCache(id ID) error {
	cacheKey := r.buildCacheKey(id)
	r.cacheTable.Delete(cacheKey)
	return nil
}

// ClearAllCache 清除所有缓存
func (r *GormCacheRepository[T, ID]) ClearAllCache() error {
	r.cacheTable.Flush()
	return nil
}

// buildCacheKey 构建缓存key
func (r *GormCacheRepository[T, ID]) buildCacheKey(id ID) string {
	return fmt.Sprintf("%v", id)
}

// buildCacheKeyFromEntity 从实体构建缓存key
// 这里需要根据实际实体类型实现
func (r *GormCacheRepository[T, ID]) buildCacheKeyFromEntity(entity *T) string {
	// 注意：这里需要使用反射获取主键值
	// 为了简化，这里返回空字符串，实际使用时需要实现
	// 或者让实现者传入一个key生成函数
	return ""
}

// BatchGet 批量获取实体（带缓存）
func (r *GormCacheRepository[T, ID]) BatchGet(ids []ID) (map[ID]*T, error) {
	result := make(map[ID]*T)
	var uncachedIDs []ID

	// 先尝试从缓存获取
	for _, id := range ids {
		cacheKey := r.buildCacheKey(id)
		cachedData, err := r.cacheTable.Value(cacheKey)
		if err == nil {
			if _, ok := cachedData.Data().(NotFound); !ok {
				result[id] = cachedData.Data().(*T)
			}
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	// 批量查询未缓存的数据
	if len(uncachedIDs) > 0 {
		var entities []*T
		query := r.db.Where(fmt.Sprintf("%s IN ?", r.primaryKey), uncachedIDs).Find(&entities)
		if query.Error != nil {
			return nil, query.Error
		}

		// 将查询结果加入缓存和返回值
		for _, entity := range entities {
			cacheKey := r.buildCacheKeyFromEntity(entity)
			// 注意：这里需要正确获取ID
			// 简化处理，实际需要反射
			r.cacheTable.Add(cacheKey, r.config.Expiration, entity)
		}
	}

	return result, nil
}
