// Package service 邀请返利服务
// 提供邀请码生成、返利计算、返利转账等功能
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// AffiliateService 邀请返利服务接口
type AffiliateService interface {
	// GenerateInviteCode 生成邀请码
	GenerateInviteCode(ctx context.Context, userID int64) (string, error)
	// GetInviteInfo 获取邀请信息
	GetInviteInfo(ctx context.Context, userID int64) (*InviteInfo, error)
	// ProcessInvite 处理邀请关系（用户注册时调用）
	ProcessInvite(ctx context.Context, newUserID int64, inviteCode string) error
	// CalculateRechargeRebate 计算充值返利
	CalculateRechargeRebate(ctx context.Context, userID int64, amount float64) (float64, error)
	// RecordRebate 记录返利
	RecordRebate(ctx context.Context, userID int64, fromUserID int64, rebateType string, amount float64, sourceAmount float64, rate float64, description string) error
	// TransferRebateToBalance 将返利余额转入账户余额
	TransferRebateToBalance(ctx context.Context, userID int64, amount float64) error
	// GetRebateRecords 获取返利记录
	GetRebateRecords(ctx context.Context, userID int64, page, pageSize int) ([]*RebateRecord, int, error)
	// GetInviteRecords 获取邀请记录
	GetInviteRecords(ctx context.Context, userID int64, page, pageSize int) ([]*InviteRecord, int, error)
	// GetAffiliateStats 获取返利统计
	GetAffiliateStats(ctx context.Context, userID int64) (*AffiliateStats, error)
}

// InviteInfo 邀请信息
type InviteInfo struct {
	InviteCode           string  `json:"invite_code"`
	InviteLink           string  `json:"invite_link"`
	InviteCount          int     `json:"invite_count"`
	AffiliateBalance     float64 `json:"affiliate_balance"`
	TotalEarnings        float64 `json:"total_earnings"`
	PendingWithdrawal    float64 `json:"pending_withdrawal"`
	MinWithdrawal        float64 `json:"min_withdrawal"`
	RechargeRate         float64 `json:"recharge_rate"`
	ConsumptionRate      float64 `json:"consumption_rate"`
	RegisterReward       float64 `json:"register_reward"`
}

// RebateRecord 返利记录
type RebateRecord struct {
	ID           int64     `json:"id"`
	FromUserID   int64     `json:"from_user_id"`
	FromUserName string    `json:"from_user_name"`
	Type         string    `json:"type"`
	Amount       float64   `json:"amount"`
	SourceAmount float64   `json:"source_amount"`
	Rate         float64   `json:"rate"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

// InviteRecord 邀请记录
type InviteRecord struct {
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// AffiliateStats 返利统计
type AffiliateStats struct {
	TotalInvites      int                `json:"total_invites"`
	TotalRebate       float64            `json:"total_rebate"`
	AvailableBalance  float64            `json:"available_balance"`
	PendingRebate     float64            `json:"pending_rebate"`
	MonthlyStats      []MonthlyStat      `json:"monthly_stats"`
}

// MonthlyStat 月度统计
type MonthlyStat struct {
	Month    string  `json:"month"`
	Invites  int     `json:"invites"`
	Rebate   float64 `json:"rebate"`
}

// affiliateService 邀请返利服务实现
type affiliateService struct {
	db     *ent.Client
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger
}

// NewAffiliateService 创建邀请返利服务
func NewAffiliateService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) AffiliateService {
	return &affiliateService{
		db:     db,
		redis:  redis,
		cfg:    cfg,
		logger: logger,
	}
}

// GenerateInviteCode 生成邀请码
func (s *affiliateService) GenerateInviteCode(ctx context.Context, userID int64) (string, error) {
	// 检查用户是否已有邀请码
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("获取用户失败: %w", err)
	}

	if user.InviteCode != "" {
		return user.InviteCode, nil
	}

	// 生成新的邀请码
	inviteCode := s.generateUniqueCode()

	// 更新用户邀请码
	_, err = s.db.User.UpdateOneID(userID).
		SetInviteCode(inviteCode).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("保存邀请码失败: %w", err)
	}

	s.logger.Info("生成邀请码成功",
		zap.Int64("user_id", userID),
		zap.String("invite_code", inviteCode))

	return inviteCode, nil
}

// GetInviteInfo 获取邀请信息
func (s *affiliateService) GetInviteInfo(ctx context.Context, userID int64) (*InviteInfo, error) {
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	// 如果没有邀请码，生成一个
	inviteCode := user.InviteCode
	if inviteCode == "" {
		inviteCode, err = s.GenerateInviteCode(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	// 构建邀请链接
	inviteLink := fmt.Sprintf("https://maas-router.com/register?ref=%s", inviteCode)

	return &InviteInfo{
		InviteCode:           inviteCode,
		InviteLink:           inviteLink,
		InviteCount:          user.InviteCount,
		AffiliateBalance:     user.AffiliateBalance,
		TotalEarnings:        user.TotalAffiliateEarnings,
		PendingWithdrawal:    0, // 可以添加待提现金额字段
		MinWithdrawal:        s.cfg.Affiliate.MinWithdrawal,
		RechargeRate:         s.cfg.Affiliate.RechargeRate,
		ConsumptionRate:      s.cfg.Affiliate.ConsumptionRate,
		RegisterReward:       s.cfg.Affiliate.RegisterReward,
	}, nil
}

// ProcessInvite 处理邀请关系
func (s *affiliateService) ProcessInvite(ctx context.Context, newUserID int64, inviteCode string) error {
	if inviteCode == "" {
		return nil
	}

	// 查找邀请人
	inviter, err := s.db.User.Query().
		Where(ent.UserInviteCode(inviteCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			s.logger.Warn("邀请码不存在", zap.String("invite_code", inviteCode))
			return nil
		}
		return fmt.Errorf("查询邀请人失败: %w", err)
	}

	// 不能邀请自己
	if inviter.ID == newUserID {
		return nil
	}

	// 更新新用户的邀请人
	_, err = s.db.User.UpdateOneID(newUserID).
		SetInvitedBy(inviter.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新邀请关系失败: %w", err)
	}

	// 增加邀请人的邀请计数
	_, err = s.db.User.UpdateOneID(inviter.ID).
		AddInviteCount(1).
		Save(ctx)
	if err != nil {
		s.logger.Warn("更新邀请计数失败", zap.Error(err))
	}

	// 记录注册返利
	if s.cfg.Affiliate.RegisterReward > 0 {
		err = s.RecordRebate(ctx, inviter.ID, newUserID, "register", s.cfg.Affiliate.RegisterReward, 0, 0, "邀请注册奖励")
		if err != nil {
			s.logger.Warn("记录注册返利失败", zap.Error(err))
		}
	}

	s.logger.Info("处理邀请关系成功",
		zap.Int64("inviter_id", inviter.ID),
		zap.Int64("new_user_id", newUserID),
		zap.String("invite_code", inviteCode))

	return nil
}

// CalculateRechargeRebate 计算充值返利
func (s *affiliateService) CalculateRechargeRebate(ctx context.Context, userID int64, amount float64) (float64, error) {
	// 获取用户的邀请人
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("获取用户失败: %w", err)
	}

	if user.InvitedBy == nil {
		// 没有邀请人，不计算返利
		return 0, nil
	}

	// 计算返利金额
	rate := s.cfg.Affiliate.RechargeRate
	rebate := amount * rate

	// 记录返利
	err = s.RecordRebate(ctx, *user.InvitedBy, userID, "recharge", rebate, amount, rate, fmt.Sprintf("充值返利 %.0f%%", rate*100))
	if err != nil {
		return 0, fmt.Errorf("记录返利失败: %w", err)
	}

	return rebate, nil
}

// RecordRebate 记录返利
func (s *affiliateService) RecordRebate(ctx context.Context, userID int64, fromUserID int64, rebateType string, amount float64, sourceAmount float64, rate float64, description string) error {
	// 创建返利记录
	_, err := s.db.AffiliateRecord.Create().
		SetUserID(userID).
		SetFromUserID(fromUserID).
		SetType(ent.AffiliateRecordType(rebateType)).
		SetAmount(amount).
		SetSourceAmount(sourceAmount).
		SetRate(rate).
		SetStatus(ent.AffiliateRecordStatusConfirmed).
		SetDescription(description).
		SetConfirmedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建返利记录失败: %w", err)
	}

	// 更新用户的返利余额和累计收益
	_, err = s.db.User.UpdateOneID(userID).
		AddAffiliateBalance(amount).
		AddTotalAffiliateEarnings(amount).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新返利余额失败: %w", err)
	}

	s.logger.Info("记录返利成功",
		zap.Int64("user_id", userID),
		zap.Int64("from_user_id", fromUserID),
		zap.String("type", rebateType),
		zap.Float64("amount", amount))

	return nil
}

// TransferRebateToBalance 将返利余额转入账户余额
func (s *affiliateService) TransferRebateToBalance(ctx context.Context, userID int64, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("转账金额必须大于0")
	}

	// 检查最低提现金额
	if amount < s.cfg.Affiliate.MinWithdrawal {
		return fmt.Errorf("最低提现金额为 %.2f 元", s.cfg.Affiliate.MinWithdrawal)
	}

	// 获取用户信息
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	// 检查返利余额
	if user.AffiliateBalance < amount {
		return fmt.Errorf("返利余额不足")
	}

	// 开始事务
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 扣除返利余额
	_, err = tx.User.UpdateOneID(userID).
		AddAffiliateBalance(-amount).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("扣除返利余额失败: %w", err)
	}

	// 增加账户余额
	_, err = tx.User.UpdateOneID(userID).
		AddBalance(amount).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("增加账户余额失败: %w", err)
	}

	// 记录提现返利
	_, err = tx.AffiliateRecord.Create().
		SetUserID(userID).
		SetFromUserID(userID).
		SetType(ent.AffiliateRecordTypeWithdrawal).
		SetAmount(-amount).
		SetStatus(ent.AffiliateRecordStatusConfirmed).
		SetDescription("返利提现到账户余额").
		SetConfirmedAt(time.Now()).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("记录提现失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	s.logger.Info("返利提现成功",
		zap.Int64("user_id", userID),
		zap.Float64("amount", amount))

	return nil
}

// GetRebateRecords 获取返利记录
func (s *affiliateService) GetRebateRecords(ctx context.Context, userID int64, page, pageSize int) ([]*RebateRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// 查询总数
	total, err := s.db.AffiliateRecord.Query().
		Where(ent.AffiliateRecordUserID(userID)).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询返利记录总数失败: %w", err)
	}

	// 查询记录
	records, err := s.db.AffiliateRecord.Query().
		Where(ent.AffiliateRecordUserID(userID)).
		Order(ent.Desc(ent.AffiliateRecordFieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		WithUser().
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询返利记录失败: %w", err)
	}

	result := make([]*RebateRecord, 0, len(records))
	for _, r := range records {
		record := &RebateRecord{
			ID:           r.ID,
			FromUserID:   r.FromUserID,
			Type:         string(r.Type),
			Amount:       r.Amount,
			SourceAmount: r.SourceAmount,
			Rate:         r.Rate,
			Status:       string(r.Status),
			Description:  r.Description,
			CreatedAt:    r.CreatedAt,
		}
		result = append(result, record)
	}

	return result, total, nil
}

// GetInviteRecords 获取邀请记录
func (s *affiliateService) GetInviteRecords(ctx context.Context, userID int64, page, pageSize int) ([]*InviteRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// 查询总数
	total, err := s.db.User.Query().
		Where(ent.UserInvitedBy(userID)).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询邀请记录总数失败: %w", err)
	}

	// 查询邀请的用户
	invitees, err := s.db.User.Query().
		Where(ent.UserInvitedBy(userID)).
		Order(ent.Desc(ent.UserFieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询邀请记录失败: %w", err)
	}

	result := make([]*InviteRecord, 0, len(invitees))
	for _, u := range invitees {
		result = append(result, &InviteRecord{
			UserID:    u.ID,
			UserName:  u.Name,
			Email:     u.Email,
			CreatedAt: u.CreatedAt,
		})
	}

	return result, total, nil
}

// GetAffiliateStats 获取返利统计
func (s *affiliateService) GetAffiliateStats(ctx context.Context, userID int64) (*AffiliateStats, error) {
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	// 计算待确认返利
	pendingRebate, err := s.db.AffiliateRecord.Query().
		Where(
			ent.AffiliateRecordUserID(userID),
			ent.AffiliateRecordStatusEQ(ent.AffiliateRecordStatusPending),
		).
		Aggregate(ent.Sum(ent.AffiliateRecordFieldAmount)).
		Float64(ctx)
	if err != nil {
		pendingRebate = 0
	}

	// 获取最近6个月的统计
	monthlyStats := make([]MonthlyStat, 0, 6)
	now := time.Now()
	for i := 0; i < 6; i++ {
		month := now.AddDate(0, -i, 0)
		startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		// 统计该月的邀请数
		inviteCount, _ := s.db.User.Query().
			Where(
				ent.UserInvitedBy(userID),
				ent.UserCreatedAtGTE(startOfMonth),
				ent.UserCreatedAtLT(endOfMonth),
			).
			Count(ctx)

		// 统计该月的返利
		monthRebate, _ := s.db.AffiliateRecord.Query().
			Where(
				ent.AffiliateRecordUserID(userID),
				ent.AffiliateRecordCreatedAtGTE(startOfMonth),
				ent.AffiliateRecordCreatedAtLT(endOfMonth),
				ent.AffiliateRecordStatusEQ(ent.AffiliateRecordStatusConfirmed),
			).
			Aggregate(ent.Sum(ent.AffiliateRecordFieldAmount)).
			Float64(ctx)

		monthlyStats = append(monthlyStats, MonthlyStat{
			Month:   month.Format("2006-01"),
			Invites: inviteCount,
			Rebate:  monthRebate,
		})
	}

	return &AffiliateStats{
		TotalInvites:     user.InviteCount,
		TotalRebate:      user.TotalAffiliateEarnings,
		AvailableBalance: user.AffiliateBalance,
		PendingRebate:    pendingRebate,
		MonthlyStats:     monthlyStats,
	}, nil
}

// generateUniqueCode 生成唯一邀请码
func (s *affiliateService) generateUniqueCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
