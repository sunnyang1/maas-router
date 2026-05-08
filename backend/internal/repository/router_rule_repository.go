// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"errors"

	"entgo.io/ent/dialect/sql"

	"maas-router/ent"
	"maas-router/ent/routerrule"
)

// RouterRuleRepository 路由规则数据访问仓库
type RouterRuleRepository struct {
	client *ent.Client
}

// NewRouterRuleRepository 创建路由规则仓库实例
func NewRouterRuleRepository(client *ent.Client) *RouterRuleRepository {
	return &RouterRuleRepository{client: client}
}

// RouterRule 路由规则实体
type RouterRule struct {
	ID          int64
	Name        string
	Description string
	Priority    int
	Condition   map[string]interface{}
	Action      map[string]interface{}
	IsActive    bool
	CreatedAt   string
	UpdatedAt   string
}

// RouterRuleListFilter 路由规则列表筛选条件
type RouterRuleListFilter struct {
	ActiveOnly bool
	SortBy     string
	SortOrder  string
}

// CreateRouterRuleInput 创建路由规则输入
type CreateRouterRuleInput struct {
	Name        string
	Description string
	Priority    int
	Condition   map[string]interface{}
	Action      map[string]interface{}
	IsActive    bool
}

// UpdateRouterRuleInput 更新路由规则输入
type UpdateRouterRuleInput struct {
	Name        *string
	Description *string
	Priority    *int
	Condition   map[string]interface{}
	Action      map[string]interface{}
	IsActive    *bool
}

// Create 创建路由规则
func (r *RouterRuleRepository) Create(ctx context.Context, input *CreateRouterRuleInput) (*RouterRule, error) {
	create := r.client.RouterRule.Create().
		SetName(input.Name).
		SetPriority(input.Priority).
		SetCondition(input.Condition).
		SetAction(input.Action).
		SetIsActive(input.IsActive)

	if input.Description != "" {
		create.SetDescription(input.Description)
	}

	rule, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToRouterRule(rule), nil
}

// Get 根据ID获取路由规则
func (r *RouterRuleRepository) Get(ctx context.Context, id int64) (*RouterRule, error) {
	rule, err := r.client.RouterRule.Query().
		Where(routerrule.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("路由规则不存在")
		}
		return nil, err
	}

	return r.convertToRouterRule(rule), nil
}

// List 获取路由规则列表
func (r *RouterRuleRepository) List(ctx context.Context, filter RouterRuleListFilter, page, pageSize int) ([]*RouterRule, int64, error) {
	query := r.client.RouterRule.Query()

	// 应用筛选条件
	if filter.ActiveOnly {
		query.Where(routerrule.IsActive(true))
	}

	// 应用排序（默认按优先级降序）
	orderBy := routerrule.FieldPriority
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
	rules, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*RouterRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.convertToRouterRule(rule))
	}

	return result, int64(total), nil
}

// Update 更新路由规则
func (r *RouterRuleRepository) Update(ctx context.Context, id int64, input *UpdateRouterRuleInput) (*RouterRule, error) {
	update := r.client.RouterRule.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.Priority != nil {
		update.SetPriority(*input.Priority)
	}
	if input.Condition != nil {
		update.SetCondition(input.Condition)
	}
	if input.Action != nil {
		update.SetAction(input.Action)
	}
	if input.IsActive != nil {
		update.SetIsActive(*input.IsActive)
	}

	rule, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToRouterRule(rule), nil
}

// Delete 删除路由规则
func (r *RouterRuleRepository) Delete(ctx context.Context, id int64) error {
	return r.client.RouterRule.DeleteOneID(id).Exec(ctx)
}

// GetActive 获取所有启用的路由规则（按优先级降序）
func (r *RouterRuleRepository) GetActive(ctx context.Context) ([]*RouterRule, error) {
	rules, err := r.client.RouterRule.Query().
		Where(routerrule.IsActive(true)).
		Order(ent.Desc(routerrule.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*RouterRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, r.convertToRouterRule(rule))
	}

	return result, nil
}

// Count 统计路由规则总数
func (r *RouterRuleRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.client.RouterRule.Query().Count(ctx)
	return int64(count), err
}

// CountActive 统计启用的路由规则数
func (r *RouterRuleRepository) CountActive(ctx context.Context) (int64, error) {
	count, err := r.client.RouterRule.Query().
		Where(routerrule.IsActive(true)).
		Count(ctx)
	return int64(count), err
}

// SetActive 设置规则启用状态
func (r *RouterRuleRepository) SetActive(ctx context.Context, id int64, isActive bool) error {
	_, err := r.client.RouterRule.UpdateOneID(id).
		SetIsActive(isActive).
		Save(ctx)
	return err
}

// BatchSetActive 批量设置规则启用状态
func (r *RouterRuleRepository) BatchSetActive(ctx context.Context, ids []int64, isActive bool) error {
	_, err := r.client.RouterRule.Update().
		Where(routerrule.IDIn(ids...)).
		SetIsActive(isActive).
		Save(ctx)
	return err
}

// Reorder 重新排序规则优先级
func (r *RouterRuleRepository) Reorder(ctx context.Context, idPriorities map[int64]int) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for id, priority := range idPriorities {
		_, err := tx.RouterRule.UpdateOneID(id).
			SetPriority(priority).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// convertToRouterRule 转换Ent实体到领域实体
func (r *RouterRuleRepository) convertToRouterRule(rule *ent.RouterRule) *RouterRule {
	return &RouterRule{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Priority:    rule.Priority,
		Condition:   rule.Condition,
		Action:      rule.Action,
		IsActive:    rule.IsActive,
		CreatedAt:   rule.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   rule.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
