package store

import (
	"log"  // 导入 log 包，用于日志输出，记录对话存储操作
	"sync" // 导入 sync 包，用于处理并发安全，通过读写互斥锁保护 map 访问

	"github.com/google/uuid" // 导入 uuid 包，用于生成唯一标识符 (UUID) 作为对话 ID
)

// ConversationStore 定义对话 ID 存储的接口
// 该接口抽象了对话 ID 的存取、生成和删除操作，允许不同的实现（如内存、数据库等）。
type ConversationStore interface {
	// GetConversationID 根据用户 ID 获取其对应的对话 ID，并返回一个布尔值指示是否存在。
	// userID: 用户的唯一标识符。
	GetConversationID(userID string) (string, bool)
	// SaveConversationID 保存或更新用户 ID 对应的对话 ID。
	// 如果 userID 已存在，则更新其 conversationID；如果不存在，则添加新的映射。
	// userID: 用户的唯一标识符。
	// conversationID: 要保存或更新的对话 ID。
	SaveConversationID(userID, conversationID string)
	// NewConversationID 为指定用户生成并保存一个新的对话 ID。
	// 该方法会生成一个全新的 UUID 作为对话 ID，并将其与用户 ID 关联起来。
	// userID: 用户的唯一标识符。
	NewConversationID(userID string) string
	// DeleteConversationID 删除用户 ID 对应的对话 ID。
	// userID: 用户的唯一标识符。
	DeleteConversationID(userID string)
}

// InMemoryConversationStore 是 ConversationStore 接口的内存实现
// 它将对话 ID 存储在内存中的一个 map 中，适用于不需要持久化存储的场景。
type InMemoryConversationStore struct {
	store map[string]string // 存储用户 ID (string) 到对话 ID (string) 的映射
	mu    sync.RWMutex      // 读写互斥锁，用于保证在并发访问 map 时的线程安全
}

// NewInMemoryConversationStore 创建并返回一个新的 InMemoryConversationStore 实例
// 这是 InMemoryConversationStore 的构造函数，负责初始化内部的 map。
func NewInMemoryConversationStore() *InMemoryConversationStore {
	return &InMemoryConversationStore{
		store: make(map[string]string), // 初始化存储 map，准备接收数据
	}
}

// GetConversationID 根据用户 ID 获取对话 ID，并指示是否存在
// 该方法是并发安全的，通过获取读锁来保护对 map 的读取操作。
func (s *InMemoryConversationStore) GetConversationID(userID string) (string, bool) {
	s.mu.RLock()         // 获取读锁，允许多个读取者同时访问
	defer s.mu.RUnlock() // 确保在函数返回时释放读锁

	conversationID, ok := s.store[userID] // 从 map 中查找对话 ID
	if ok {
		log.Printf("[ConversationStore] 获取对话ID成功，用户: '%s', 对话ID: '%s'", userID, conversationID)
	} else {
		log.Printf("[ConversationStore] 未找到用户 '%s' 的对话ID", userID)
	}
	return conversationID, ok // 返回对话 ID 和一个布尔值，指示是否找到
}

// SaveConversationID 保存或更新用户 ID 对应的对话 ID
// 该方法是并发安全的，通过获取写锁来保护对 map 的写入操作。
func (s *InMemoryConversationStore) SaveConversationID(userID, conversationID string) {
	s.mu.Lock()         // 获取写锁，独占访问，防止其他读写操作
	defer s.mu.Unlock() // 确保在函数返回时释放写锁

	s.store[userID] = conversationID // 设置或更新用户 ID 对应的对话 ID
	log.Printf("[ConversationStore] 保存对话ID成功，用户: '%s', 对话ID: '%s'", userID, conversationID)
}

// NewConversationID 为指定用户生成并保存一个新的对话 ID
// 该方法是并发安全的，通过获取写锁来保护对 map 的写入操作。
func (s *InMemoryConversationStore) NewConversationID(userID string) string {
	s.mu.Lock()         // 获取写锁
	defer s.mu.Unlock() // 确保在函数返回时释放写锁

	// 使用 UUID 包生成一个全局唯一的对话 ID
	conversationID := uuid.New().String()
	s.store[userID] = conversationID // 将新生成的对话 ID 保存到 map 中
	log.Printf("[ConversationStore] 为用户 '%s' 生成并保存新的对话ID: '%s'", userID, conversationID)
	return conversationID // 返回新生成的对话 ID
}

// DeleteConversationID 删除用户 ID 对应的对话 ID
// 该方法是并发安全的，通过获取写锁来保护对 map 的删除操作。
func (s *InMemoryConversationStore) DeleteConversationID(userID string) {
	s.mu.Lock()         // 获取写锁
	defer s.mu.Unlock() // 确保在函数返回时释放写锁

	delete(s.store, userID) // 从 map 中删除指定用户 ID 的对话 ID
	log.Printf("[ConversationStore] 删除用户 '%s' 的对话ID成功", userID)
}
