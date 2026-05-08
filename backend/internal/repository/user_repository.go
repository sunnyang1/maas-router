// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"maas-router/ent"
	"maas-router/ent/user"
)

// UserRepository 用户数据访问仓库
type UserRepository struct {
	client *ent.Client
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(client *ent.Client) *UserRepository {
	return &UserRepository{client: client}
}

// User 用户实体
type User struct {
	ID            int64
	Email         string
	PasswordHash  string
	Name          string
	Role          string
	Status        string
	Balance       float64
	Concurrency   int
	TokenVersion  int
	LastActiveAt  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserListFilter 用户列表筛选条件
type UserListFilter struct {
	Keyword   string
	Role      string
	Status    string
	SortBy    string
	SortOrder string
}

// CreateUserInput 创建用户输入
type CreateUserInput struct {
	Email       string
	Password    string
	Name        string
	Role        string
	Balance     float64
	Concurrency int
}

// UpdateUserInput 更新用户输入
type UpdateUserInput struct {
	Name        *string
	Role        *string
	Status      *string
	Concurrency *int
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, input *CreateUserInput) (*User, error) {
	// 生成密码哈希
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	u, err := r.client.User.Create().
		SetEmail(input.Email).
		SetPasswordHash(string(passwordHash)).
		SetNillableName(&input.Name).
		SetRole(user.Role(input.Role)).
		SetBalance(input.Balance).
		SetConcurrency(input.Concurrency).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToUser(u), nil
}

// Get 根据ID获取用户
func (r *UserRepository) Get(ctx context.Context, id int64) (*User, error) {
	u, err := r.client.User.Query().
		Where(user.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	return r.convertToUser(u), nil
}

// GetByEmail 根据邮箱获取用户
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u, err := r.client.User.Query().
		Where(user.Email(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	return r.convertToUser(u), nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, id int64, input *UpdateUserInput) (*User, error) {
	update := r.client.User.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Role != nil {
		update.SetRole(user.Role(*input.Role))
	}
	if input.Status != nil {
		update.SetStatus(user.Status(*input.Status))
	}
	if input.Concurrency != nil {
		update.SetConcurrency(*input.Concurrency)
	}

	u, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToUser(u), nil
}

// Delete 删除用户（软删除，设置状态为deleted）
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.client.User.UpdateOneID(id).
		SetStatus(user.StatusDeleted).
		Save(ctx)
	return err
}

// List 获取用户列表
func (r *UserRepository) List(ctx context.Context, filter UserListFilter, page, pageSize int) ([]*User, int64, error) {
	query := r.client.User.Query()

	// 应用筛选条件
	if filter.Keyword != "" {
		query.Where(user.Or(
			user.EmailContains(filter.Keyword),
			user.NameContains(filter.Keyword),
		))
	}
	if filter.Role != "" {
		query.Where(user.Role(user.Role(filter.Role)))
	}
	if filter.Status != "" {
		query.Where(user.Status(user.Status(filter.Status)))
	}

	// 应用排序
	orderBy := user.FieldCreatedAt
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}
	orderDirection := sql.OrderDesc()
	if filter.SortOrder == "asc" {
		orderDirection = sql.OrderAsc()
	}
	query.Order(ent.OrderByFunc(func(s *sql.Selector) {
		s.OrderBy(sql.OrderExpr(orderDirection, orderBy))
	}))

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (page - 1) * pageSize
	users, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*User, 0, len(users))
	for _, u := range users {
		result = append(result, r.convertToUser(u))
	}

	return result, int64(total), nil
}

// Count 统计用户总数
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.client.User.Query().Count(ctx)
	return int64(count), err
}

// CountByStatus 按状态统计用户数
func (r *UserRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	count, err := r.client.User.Query().
		Where(user.Status(user.Status(status))).
		Count(ctx)
	return int64(count), err
}

// CountByCreatedAt 按创建时间统计用户数
func (r *UserRepository) CountByCreatedAt(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	count, err := r.client.User.Query().
		Where(user.CreatedAtGTE(startTime), user.CreatedAtLTE(endTime)).
		Count(ctx)
	return int64(count), err
}

// ExistsByEmail 检查邮箱是否已存在
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.client.User.Query().
		Where(user.Email(email)).
		Exist(ctx)
}

// CountAPIKeys 统计用户的API Key数量
func (r *UserRepository) CountAPIKeys(ctx context.Context, userID int64) (int64, error) {
	u, err := r.client.User.Query().
		Where(user.ID(userID)).
		WithAPIKeys().
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(u.Edges.APIKeys)), nil
}

// AdjustBalance 调整用户余额
func (r *UserRepository) AdjustBalance(ctx context.Context, id int64, amount float64, reason string) (float64, error) {
	// 开启事务
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 获取当前用户
	u, err := tx.User.Query().Where(user.ID(id)).Only(ctx)
	if err != nil {
		return 0, err
	}

	// 计算新余额
	newBalance := u.Balance + amount
	if newBalance < 0 {
		return 0, errors.New("余额不足")
	}

	// 更新余额
	updatedUser, err := tx.User.UpdateOneID(id).
		SetBalance(newBalance).
		Save(ctx)
	if err != nil {
		return 0, err
	}

	// TODO: 记录余额变动日志

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return updatedUser.Balance, nil
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, newPassword string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = r.client.User.UpdateOneID(id).
		SetPasswordHash(string(passwordHash)).
		SetTokenVersion(uuid.New().ID()).
		Save(ctx)
	return err
}

// UpdateLastActiveAt 更新最后活跃时间
func (r *UserRepository) UpdateLastActiveAt(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.User.UpdateOneID(id).
		SetLastActiveAt(now).
		Save(ctx)
	return err
}

// convertToUser 转换Ent实体到领域实体
func (r *UserRepository) convertToUser(u *ent.User) *User {
	return &User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Role:         string(u.Role),
		Status:       string(u.Status),
		Balance:      u.Balance,
		Concurrency:  u.Concurrency,
		TokenVersion: u.TokenVersion,
		LastActiveAt: u.LastActiveAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
