// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"

	"maas-router/ent"
	"maas-router/ent/account"
)

// AccountRepository 账号数据访问仓库
type AccountRepository struct {
	client *ent.Client
}

// NewAccountRepository 创建账号仓库实例
func NewAccountRepository(client *ent.Client) *AccountRepository {
	return &AccountRepository{client: client}
}

// Account 账号实体
type Account struct {
	ID                 int64
	Name               string
	Platform           string
	AccountType        string
	Credentials        map[string]interface{}
	Status             string
	MaxConcurrency     int
	CurrentConcurrency int
	RPMLimit           int
	TotalRequests      int64
	ErrorCount         int64
	LastUsedAt         *time.Time
	LastErrorAt        *time.Time
	ProxyURL           string
	TLSFingerprint     string
	Extra              map[string]interface{}
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// AccountListFilter 账号列表筛选条件
type AccountListFilter struct {
	Platform    string
	Status      string
	AccountType string
	Keyword     string
	SortBy      string
	SortOrder   string
}

// CreateAccountInput 创建账号输入
type CreateAccountInput struct {
	Name            string
	Platform        string
	AccountType     string
	Credentials     map[string]interface{}
	MaxConcurrency  int
	RPMLimit        int
	ProxyURL        string
	TLSFingerprint  string
	Extra           map[string]interface{}
	GroupIDs        []int64
}

// UpdateAccountInput 更新账号输入
type UpdateAccountInput struct {
	Name            *string
	Credentials     map[string]interface{}
	Status          *string
	MaxConcurrency  *int
	RPMLimit        *int
	ProxyURL        *string
	TLSFingerprint  *string
	Extra           map[string]interface{}
}

// Create 创建账号
func (r *AccountRepository) Create(ctx context.Context, input *CreateAccountInput) (*Account, error) {
	create := r.client.Account.Create().
		SetName(input.Name).
		SetPlatform(account.Platform(input.Platform)).
		SetAccountType(account.AccountType(input.AccountType)).
		SetCredentials(input.Credentials).
		SetMaxConcurrency(input.MaxConcurrency).
		SetRPMLimit(input.RPMLimit)

	if input.ProxyURL != "" {
		create.SetProxyURL(input.ProxyURL)
	}
	if input.TLSFingerprint != "" {
		create.SetTLSFingerprint(input.TLSFingerprint)
	}
	if input.Extra != nil {
		create.SetExtra(input.Extra)
	}

	acc, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	// 关联分组
	if len(input.GroupIDs) > 0 {
		_, err = r.client.Account.UpdateOneID(acc.ID).
			AddGroupIDs(input.GroupIDs...).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	return r.convertToAccount(acc), nil
}

// Get 根据ID获取账号
func (r *AccountRepository) Get(ctx context.Context, id int64) (*Account, error) {
	acc, err := r.client.Account.Query().
		Where(account.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("账号不存在")
		}
		return nil, err
	}

	return r.convertToAccount(acc), nil
}

// List 获取账号列表
func (r *AccountRepository) List(ctx context.Context, filter AccountListFilter, page, pageSize int) ([]*Account, int64, error) {
	query := r.client.Account.Query()

	// 应用筛选条件
	if filter.Platform != "" {
		query.Where(account.Platform(account.Platform(filter.Platform)))
	}
	if filter.Status != "" {
		query.Where(account.Status(account.Status(filter.Status)))
	}
	if filter.AccountType != "" {
		query.Where(account.AccountType(account.AccountType(filter.AccountType)))
	}
	if filter.Keyword != "" {
		query.Where(account.NameContains(filter.Keyword))
	}

	// 应用排序
	orderBy := account.FieldCreatedAt
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
	accounts, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, r.convertToAccount(acc))
	}

	return result, int64(total), nil
}

// Update 更新账号
func (r *AccountRepository) Update(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error) {
	update := r.client.Account.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Credentials != nil {
		update.SetCredentials(input.Credentials)
	}
	if input.Status != nil {
		update.SetStatus(account.Status(*input.Status))
	}
	if input.MaxConcurrency != nil {
		update.SetMaxConcurrency(*input.MaxConcurrency)
	}
	if input.RPMLimit != nil {
		update.SetRPMLimit(*input.RPMLimit)
	}
	if input.ProxyURL != nil {
		update.SetProxyURL(*input.ProxyURL)
	}
	if input.TLSFingerprint != nil {
		update.SetTLSFingerprint(*input.TLSFingerprint)
	}
	if input.Extra != nil {
		update.SetExtra(input.Extra)
	}

	acc, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToAccount(acc), nil
}

// Delete 删除账号
func (r *AccountRepository) Delete(ctx context.Context, id int64) error {
	return r.client.Account.DeleteOneID(id).Exec(ctx)
}

// GetByPlatform 根据平台获取账号列表
func (r *AccountRepository) GetByPlatform(ctx context.Context, platform string) ([]*Account, error) {
	accounts, err := r.client.Account.Query().
		Where(
			account.Platform(account.Platform(platform)),
			account.Status(account.StatusActive),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, r.convertToAccount(acc))
	}

	return result, nil
}

// GetAvailable 获取可用账号列表（状态为active且并发未满）
func (r *AccountRepository) GetAvailable(ctx context.Context, platform string) ([]*Account, error) {
	accounts, err := r.client.Account.Query().
		Where(
			account.Platform(account.Platform(platform)),
			account.Status(account.StatusActive),
		).
		Where(func(s *sql.Predicate) {
			// 当前并发数小于最大并发数
			s.Append(sql.ExprP("current_concurrency < max_concurrency"))
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, r.convertToAccount(acc))
	}

	return result, nil
}

// ListActive 获取所有活跃账号
func (r *AccountRepository) ListActive(ctx context.Context) ([]*Account, error) {
	accounts, err := r.client.Account.Query().
		Where(account.Status(account.StatusActive)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, r.convertToAccount(acc))
	}

	return result, nil
}

// Count 统计账号总数
func (r *AccountRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.client.Account.Query().Count(ctx)
	return int64(count), err
}

// CountByStatus 按状态统计账号数
func (r *AccountRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	count, err := r.client.Account.Query().
		Where(account.Status(account.Status(status))).
		Count(ctx)
	return int64(count), err
}

// CountGroupByPlatform 按平台分组统计账号数
func (r *AccountRepository) CountGroupByPlatform(ctx context.Context) (map[string]int64, error) {
	type platformCount struct {
		Platform string `json:"platform"`
		Count    int    `json:"count"`
	}

	var results []platformCount
	err := r.client.Account.Query().
		Modify(func(s *sql.Selector) {
			s.Select(
				sql.As(sql.Expr("platform"), "platform"),
				sql.As(sql.Count(sql.Expr("*")), "count"),
			).
				GroupBy(sql.Expr("platform"))
		}).
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, r := range results {
		result[r.Platform] = int64(r.Count)
	}

	return result, nil
}

// GetGroupIDs 获取账号所属分组ID列表
func (r *AccountRepository) GetGroupIDs(ctx context.Context, accountID int64) ([]int64, error) {
	groups, err := r.client.Account.Query().
		Where(account.ID(accountID)).
		QueryGroups().
		IDs(ctx)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// IncrementConcurrency 增加当前并发数
func (r *AccountRepository) IncrementConcurrency(ctx context.Context, id int64) error {
	_, err := r.client.Account.UpdateOneID(id).
		SetCurrentConcurrency(
			sql.ExprFunc(func(b *sql.Builder) {
				b.WriteString("current_concurrency + 1")
			}),
		).
		Save(ctx)
	return err
}

// DecrementConcurrency 减少当前并发数
func (r *AccountRepository) DecrementConcurrency(ctx context.Context, id int64) error {
	_, err := r.client.Account.UpdateOneID(id).
		SetCurrentConcurrency(
			sql.ExprFunc(func(b *sql.Builder) {
				b.WriteString("GREATEST(current_concurrency - 1, 0)")
			}),
		).
		Save(ctx)
	return err
}

// IncrementRequestCount 增加请求计数
func (r *AccountRepository) IncrementRequestCount(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.Account.UpdateOneID(id).
		SetTotalRequests(
			sql.ExprFunc(func(b *sql.Builder) {
				b.WriteString("total_requests + 1")
			}),
		).
		SetLastUsedAt(now).
		Save(ctx)
	return err
}

// IncrementErrorCount 增加错误计数
func (r *AccountRepository) IncrementErrorCount(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.Account.UpdateOneID(id).
		SetErrorCount(
			sql.ExprFunc(func(b *sql.Builder) {
				b.WriteString("error_count + 1")
			}),
		).
		SetLastErrorAt(now).
		Save(ctx)
	return err
}

// SumCurrentConcurrency 汇总当前并发数
func (r *AccountRepository) SumCurrentConcurrency(ctx context.Context) (int64, error) {
	var result struct {
		Sum int64 `json:"sum"`
	}

	err := r.client.Account.Query().
		Modify(func(s *sql.Selector) {
			s.Select(sql.As(sql.Sum(sql.Expr("current_concurrency")), "sum"))
		}).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}

	return result.Sum, nil
}

// convertToAccount 转换Ent实体到领域实体
func (r *AccountRepository) convertToAccount(acc *ent.Account) *Account {
	return &Account{
		ID:                 acc.ID,
		Name:               acc.Name,
		Platform:           string(acc.Platform),
		AccountType:        string(acc.AccountType),
		Credentials:        acc.Credentials,
		Status:             string(acc.Status),
		MaxConcurrency:     acc.MaxConcurrency,
		CurrentConcurrency: acc.CurrentConcurrency,
		RPMLimit:           acc.RPMLimit,
		TotalRequests:      acc.TotalRequests,
		ErrorCount:         acc.ErrorCount,
		LastUsedAt:         acc.LastUsedAt,
		LastErrorAt:        acc.LastErrorAt,
		ProxyURL:           acc.ProxyURL,
		TLSFingerprint:     acc.TLSFingerprint,
		Extra:              acc.Extra,
		CreatedAt:          acc.CreatedAt,
		UpdatedAt:          acc.UpdatedAt,
	}
}
