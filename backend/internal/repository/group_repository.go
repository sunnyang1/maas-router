// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"

	"maas-router/ent"
	"maas-router/ent/group"
)

// GroupRepository 分组数据访问仓库
type GroupRepository struct {
	client *ent.Client
}

// NewGroupRepository 创建分组仓库实例
func NewGroupRepository(client *ent.Client) *GroupRepository {
	return &GroupRepository{client: client}
}

// Group 分组实体
type Group struct {
	ID             int64
	Name           string
	Description    string
	Platform       string
	BillingMode    string
	RateMultiplier float64
	RPMOverride    *int
	ModelMapping   map[string]string
	Priority       int
	Weight         int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GroupListFilter 分组列表筛选条件
type GroupListFilter struct {
	Platform  string
	Status    string
	Keyword   string
	SortBy    string
	SortOrder string
}

// CreateGroupInput 创建分组输入
type CreateGroupInput struct {
	Name           string
	Description    string
	Platform       string
	BillingMode    string
	RateMultiplier float64
	RPMOverride    *int
	ModelMapping   map[string]string
	Priority       int
	Weight         int
	Status         string
	AccountIDs     []int64
}

// UpdateGroupInput 更新分组输入
type UpdateGroupInput struct {
	Name           *string
	Description    *string
	BillingMode    *string
	RateMultiplier *float64
	RPMOverride    *int
	ModelMapping   map[string]string
	Priority       *int
	Weight         *int
	Status         *string
}

// Create 创建分组
func (r *GroupRepository) Create(ctx context.Context, input *CreateGroupInput) (*Group, error) {
	create := r.client.Group.Create().
		SetName(input.Name).
		SetPlatform(group.Platform(input.Platform)).
		SetBillingMode(group.BillingMode(input.BillingMode)).
		SetRateMultiplier(input.RateMultiplier).
		SetPriority(input.Priority).
		SetWeight(input.Weight).
		SetStatus(group.Status(input.Status))

	if input.Description != "" {
		create.SetDescription(input.Description)
	}
	if input.RPMOverride != nil {
		create.SetRpmOverride(*input.RPMOverride)
	}
	if input.ModelMapping != nil {
		create.SetModelMapping(input.ModelMapping)
	}

	g, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	// 关联账号
	if len(input.AccountIDs) > 0 {
		_, err = r.client.Group.UpdateOneID(g.ID).
			AddAccountIDs(input.AccountIDs...).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	return r.convertToGroup(g), nil
}

// Get 根据ID获取分组
func (r *GroupRepository) Get(ctx context.Context, id int64) (*Group, error) {
	g, err := r.client.Group.Query().
		Where(group.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("分组不存在")
		}
		return nil, err
	}

	return r.convertToGroup(g), nil
}

// List 获取分组列表
func (r *GroupRepository) List(ctx context.Context, filter GroupListFilter, page, pageSize int) ([]*Group, int64, error) {
	query := r.client.Group.Query()

	// 应用筛选条件
	if filter.Platform != "" {
		query.Where(group.Platform(group.Platform(filter.Platform)))
	}
	if filter.Status != "" {
		query.Where(group.Status(group.Status(filter.Status)))
	}
	if filter.Keyword != "" {
		query.Where(group.Or(
			group.NameContains(filter.Keyword),
			group.DescriptionContains(filter.Keyword),
		))
	}

	// 应用排序
	orderBy := group.FieldCreatedAt
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
	groups, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*Group, 0, len(groups))
	for _, g := range groups {
		result = append(result, r.convertToGroup(g))
	}

	return result, int64(total), nil
}

// Update 更新分组
func (r *GroupRepository) Update(ctx context.Context, id int64, input *UpdateGroupInput) (*Group, error) {
	update := r.client.Group.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.BillingMode != nil {
		update.SetBillingMode(group.BillingMode(*input.BillingMode))
	}
	if input.RateMultiplier != nil {
		update.SetRateMultiplier(*input.RateMultiplier)
	}
	if input.RPMOverride != nil {
		update.SetRpmOverride(*input.RPMOverride)
	} else {
		update.ClearRpmOverride()
	}
	if input.ModelMapping != nil {
		update.SetModelMapping(input.ModelMapping)
	}
	if input.Priority != nil {
		update.SetPriority(*input.Priority)
	}
	if input.Weight != nil {
		update.SetWeight(*input.Weight)
	}
	if input.Status != nil {
		update.SetStatus(group.Status(*input.Status))
	}

	g, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToGroup(g), nil
}

// Delete 删除分组
func (r *GroupRepository) Delete(ctx context.Context, id int64) error {
	return r.client.Group.DeleteOneID(id).Exec(ctx)
}

// AddAccount 添加账号到分组
func (r *GroupRepository) AddAccount(ctx context.Context, groupID, accountID int64) error {
	_, err := r.client.Group.UpdateOneID(groupID).
		AddAccountIDs(accountID).
		Save(ctx)
	return err
}

// AddAccounts 批量添加账号到分组
func (r *GroupRepository) AddAccounts(ctx context.Context, groupID int64, accountIDs []int64) error {
	_, err := r.client.Group.UpdateOneID(groupID).
		AddAccountIDs(accountIDs...).
		Save(ctx)
	return err
}

// RemoveAccount 从分组移除账号
func (r *GroupRepository) RemoveAccount(ctx context.Context, groupID, accountID int64) error {
	_, err := r.client.Group.UpdateOneID(groupID).
		RemoveAccountIDs(accountID).
		Save(ctx)
	return err
}

// RemoveAccounts 批量从分组移除账号
func (r *GroupRepository) RemoveAccounts(ctx context.Context, groupID int64, accountIDs []int64) error {
	_, err := r.client.Group.UpdateOneID(groupID).
		RemoveAccountIDs(accountIDs...).
		Save(ctx)
	return err
}

// GetAccounts 获取分组关联的账号列表
func (r *GroupRepository) GetAccounts(ctx context.Context, groupID int64) ([]*Account, error) {
	accounts, err := r.client.Group.Query().
		Where(group.ID(groupID)).
		QueryAccounts().
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, &Account{
			ID:                 acc.ID,
			Name:               acc.Name,
			Platform:           string(acc.Platform),
			Status:             string(acc.Status),
			MaxConcurrency:     acc.MaxConcurrency,
			CurrentConcurrency: acc.CurrentConcurrency,
		})
	}

	return result, nil
}

// CountAccounts 统计分组中的账号数量
func (r *GroupRepository) CountAccounts(ctx context.Context, groupID int64) (int64, error) {
	count, err := r.client.Group.Query().
		Where(group.ID(groupID)).
		QueryAccounts().
		Count(ctx)
	return int64(count), err
}

// Count 统计分组总数
func (r *GroupRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.client.Group.Query().Count(ctx)
	return int64(count), err
}

// CountByStatus 按状态统计分组数
func (r *GroupRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	count, err := r.client.Group.Query().
		Where(group.Status(group.Status(status))).
		Count(ctx)
	return int64(count), err
}

// CountGroupByPlatform 按平台分组统计分组数
func (r *GroupRepository) CountGroupByPlatform(ctx context.Context) (map[string]int64, error) {
	type platformCount struct {
		Platform string `json:"platform"`
		Count    int    `json:"count"`
	}

	var results []platformCount
	err := r.client.Group.Query().
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

// GetActiveByPlatform 根据平台获取活跃分组列表
func (r *GroupRepository) GetActiveByPlatform(ctx context.Context, platform string) ([]*Group, error) {
	groups, err := r.client.Group.Query().
		Where(
			group.Platform(group.Platform(platform)),
			group.Status(group.StatusActive),
		).
		Order(ent.Desc(group.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Group, 0, len(groups))
	for _, g := range groups {
		result = append(result, r.convertToGroup(g))
	}

	return result, nil
}

// convertToGroup 转换Ent实体到领域实体
func (r *GroupRepository) convertToGroup(g *ent.Group) *Group {
	return &Group{
		ID:             g.ID,
		Name:           g.Name,
		Description:    g.Description,
		Platform:       string(g.Platform),
		BillingMode:    string(g.BillingMode),
		RateMultiplier: g.RateMultiplier,
		RPMOverride:    g.RpmOverride,
		ModelMapping:   g.ModelMapping,
		Priority:       g.Priority,
		Weight:         g.Weight,
		Status:         string(g.Status),
		CreatedAt:      g.CreatedAt,
		UpdatedAt:      g.UpdatedAt,
	}
}
