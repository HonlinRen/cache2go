package examples

import (
	"testing"
	"time"

	"github.com/muesli/cache2go/cache"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// 使用SQLite内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&User{})
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestUserCacheRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserCacheRepository(db)

	// 准备测试数据
	users := []*User{
		{Username: "user1", Email: "user1@example.com", Age: 20},
		{Username: "user2", Email: "user2@example.com", Age: 25},
		{Username: "user3", Email: "user3@example.com", Age: 30},
	}

	// 测试：批量保存用户
	t.Run("BatchSave", func(t *testing.T) {
		for _, user := range users {
			err := repo.Save(user)
			if err != nil {
				t.Errorf("Failed to save user: %v", err)
			}
		}
	})

	// 测试：查询用户（第一次从数据库）
	t.Run("GetByID_FirstTime", func(t *testing.T) {
		user, err := repo.GetByID(users[0].ID)
		if err != nil {
			t.Errorf("Failed to get user: %v", err)
		}
		if user.Username != users[0].Username {
			t.Errorf("Username mismatch: got %s, want %s", user.Username, users[0].Username)
		}
	})

	// 测试：查询用户（第二次应该从缓存）
	t.Run("GetByID_FromCache", func(t *testing.T) {
		// 先清空数据库（模拟缓存场景）
		// 注意：实际测试中我们无法清空数据库，这里仅用于演示
		user, err := repo.GetByID(users[0].ID)
		if err != nil {
			t.Errorf("Failed to get user from cache: %v", err)
		}
		if user.Username != users[0].Username {
			t.Errorf("Username mismatch: got %s, want %s", user.Username, users[0].Username)
		}
	})

	// 测试：查询不存在的用户
	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(99999)
		if err == nil {
			t.Error("Expected error for non-existent user, got nil")
		}

		// 再次查询（应该从缓存）
		_, err = repo.GetByID(99999)
		if err == nil {
			t.Error("Expected error for non-existent user from cache, got nil")
		}
	})

	// 测试：获取所有用户
	t.Run("GetAll", func(t *testing.T) {
		allUsers, err := repo.GetAll()
		if err != nil {
			t.Errorf("Failed to get all users: %v", err)
		}
		if len(allUsers) != len(users) {
			t.Errorf("User count mismatch: got %d, want %d", len(allUsers), len(users))
		}
	})

	// 测试：更新用户
	t.Run("UpdateUser", func(t *testing.T) {
		user, err := repo.GetByID(users[0].ID)
		if err != nil {
			t.Errorf("Failed to get user: %v", err)
		}

		originalAge := user.Age
		user.Age = originalAge + 10

		err = repo.Save(user)
		if err != nil {
			t.Errorf("Failed to update user: %v", err)
		}

		// 验证更新
		updatedUser, err := repo.GetByID(users[0].ID)
		if err != nil {
			t.Errorf("Failed to get updated user: %v", err)
		}
		if updatedUser.Age != originalAge+10 {
			t.Errorf("Age not updated: got %d, want %d", updatedUser.Age, originalAge+10)
		}
	})

	// 测试：删除用户
	t.Run("DeleteUser", func(t *testing.T) {
		userID := users[1].ID

		err := repo.Delete(userID)
		if err != nil {
			t.Errorf("Failed to delete user: %v", err)
		}

		// 验证删除
		_, err = repo.GetByID(userID)
		if err == nil {
			t.Error("Expected error for deleted user, got nil")
		}
	})

	// 测试：缓存过期
	t.Run("CacheExpiration", func(t *testing.T) {
		// 创建一个短期缓存的仓库
		shortConfig := cache.CacheConfig{
			TableName:        "short_cache",
			Expiration:       100 * time.Millisecond,
			UseNotFoundCache: true,
		}
		shortRepo := &UserCacheRepository{
			GormCacheRepository: cache.NewGormCacheRepository[User, uint](db, shortConfig, "id"),
		}

		// 保存并查询
		user := &User{Username: "temp", Email: "temp@example.com", Age: 99}
		err := shortRepo.Save(user)
		if err != nil {
			t.Errorf("Failed to save user: %v", err)
		}

		// 立即查询（应该命中缓存）
		_, err = shortRepo.GetByID(user.ID)
		if err != nil {
			t.Errorf("Failed to get user from cache: %v", err)
		}

		// 等待缓存过期
		time.Sleep(150 * time.Millisecond)

		// 再次查询（应该从数据库重新加载）
		_, err = shortRepo.GetByID(user.ID)
		if err != nil {
			t.Errorf("Failed to get user after cache expired: %v", err)
		}
	})
}
