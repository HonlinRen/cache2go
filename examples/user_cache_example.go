package examples

import (
	"fmt"
	"time"

	"github.com/muesli/cache2go/cache"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// UserCacheRepository 用户缓存仓库（针对User特化）
type UserCacheRepository struct {
	*cache.GormCacheRepository[User, uint]
}

// NewUserCacheRepository 创建用户缓存仓库
func NewUserCacheRepository(db *gorm.DB) *UserCacheRepository {
	config := cache.CacheConfig{
		TableName:        "users_cache",    // cache2go 缓存表名
		Expiration:       10 * time.Minute, // 缓存10分钟过期
		UseNotFoundCache: true,             // 缓存不存在的用户
	}

	return &UserCacheRepository{
		GormCacheRepository: cache.NewGormCacheRepository[User, uint](db, config, "id"),
	}
}

// InitDatabase 初始化数据库连接
func InitDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// ExampleUsage 使用示例
func ExampleUsage() {
	// 1. 初始化数据库连接
	dsn := "root:password@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := InitDatabase(dsn)
	if err != nil {
		panic(err)
	}

	// 2. 自动迁移（创建表）
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic(err)
	}

	// 3. 创建缓存仓库
	userRepo := NewUserCacheRepository(db)

	// ========== 示例1：创建用户 ==========
	user := &User{
		Username: "john_doe",
		Email:    "john@example.com",
		Age:      30,
		Active:   true,
	}
	err = userRepo.Save(user)
	if err != nil {
		panic(err)
	}
	fmt.Printf("创建用户成功，ID: %d\n", user.ID)

	// ========== 示例2：根据ID查询用户（第一次从数据库，第二次从缓存） ==========
	queriedUser, err := userRepo.GetByID(user.ID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("查询用户: %+v\n", queriedUser)

	// 再次查询（从缓存）
	queriedUser2, err := userRepo.GetByID(user.ID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("从缓存查询用户: %+v\n", queriedUser2)

	// ========== 示例3：查询不存在的用户（会被缓存） ==========
	_, err = userRepo.GetByID(99999)
	if err != nil {
		fmt.Println("查询不存在的用户:", err)
	}

	// 再次查询不存在的用户（从缓存，不会访问数据库）
	_, err = userRepo.GetByID(99999)
	if err != nil {
		fmt.Println("从缓存查询不存在的用户:", err)
	}

	// ========== 示例4：更新用户 ==========
	queriedUser.Age = 31
	err = userRepo.Save(queriedUser)
	if err != nil {
		panic(err)
	}
	fmt.Println("更新用户成功")

	// ========== 示例5：获取所有用户 ==========
	allUsers, err := userRepo.GetAll()
	if err != nil {
		panic(err)
	}
	fmt.Printf("所有用户数量: %d\n", len(allUsers))

	// ========== 示例6：删除用户并清除缓存 ==========
	err = userRepo.Delete(user.ID)
	if err != nil {
		panic(err)
	}
	fmt.Println("删除用户成功")

	// ========== 示例7：批量查询 ==========
	ids := []uint{1, 2, 3, 4, 5}
	userMap, err := userRepo.BatchGet(ids)
	if err != nil {
		panic(err)
	}
	fmt.Printf("批量查询到 %d 个用户\n", len(userMap))

	// ========== 示例8：清除指定缓存 ==========
	userRepo.ClearCache(123)
	fmt.Println("清除指定用户缓存")

	// ========== 示例9：清除所有缓存 ==========
	userRepo.ClearAllCache()
	fmt.Println("清除所有缓存")
}
