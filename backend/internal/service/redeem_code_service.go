// Package service 卡密充值服务
// 提供卡密生成、验证、充值等功能
package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedeemCodeService 卡密充值服务接口
type RedeemCodeService interface {
	// GenerateCodes 批量生成卡密
	GenerateCodes(ctx context.Context, adminID int64, amount float64, count int, expiresAt *time.Time, remark string) ([]*ent.RedeemCode, error)
	// ValidateCode 验证卡密
	ValidateCode(ctx context.Context, code string) (*ent.RedeemCode, error)
	// RedeemCode 使用卡密充值
	RedeemCode(ctx context.Context, userID int64, code string) (*RedeemResult, error)
	// GetCodeList 获取卡密列表（管理员）
	GetCodeList(ctx context.Context, status string, batchNo string, page, pageSize int) ([]*ent.RedeemCode, int, error)
	// GetCodeDetail 获取卡密详情
	GetCodeDetail(ctx context.Context, codeID int64) (*ent.RedeemCode, error)
	// DisableCode 禁用卡密
	DisableCode(ctx context.Context, codeID int64) error
	// GetUserRedeemRecords 获取用户的充值记录
	GetUserRedeemRecords(ctx context.Context, userID int64, page, pageSize int) ([]*RedeemRecord, int, error)
	// ExportCodes 导出卡密（批量）
	ExportCodes(ctx context.Context, batchNo string) ([]*ent.RedeemCode, error)
}

// RedeemResult 充值结果
type RedeemResult struct {
	Code      string    `json:"code"`
	Amount    float64   `json:"amount"`
	Balance   float64   `json:"balance"`
	RedeemedAt time.Time `json:"redeemed_at"`
}

// RedeemRecord 充值记录
type RedeemRecord struct {
	ID         int64     `json:"id"`
	Code       string    `json:"code"`
	Amount     float64   `json:"amount"`
	RedeemedAt time.Time `json:"redeemed_at"`
}

// redeemCodeService 卡密充值服务实现
type redeemCodeService struct {
	db     *ent.Client
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger
}

// NewRedeemCodeService 创建卡密充值服务
func NewRedeemCodeService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) RedeemCodeService {
	return &redeemCodeService{
		db:     db,
		redis:  redis,
		cfg:    cfg,
		logger: logger,
	}
}

// GenerateCodes 批量生成卡密
func (s *redeemCodeService) GenerateCodes(ctx context.Context, adminID int64, amount float64, count int, expiresAt *time.Time, remark string) ([]*ent.RedeemCode, error) {
	if count <= 0 || count > 1000 {
		return nil, fmt.Errorf("生成数量必须在 1-1000 之间")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("面额必须大于 0")
	}

	// 生成批次号
	batchNo := s.generateBatchNo()

	codes := make([]*ent.RedeemCode, 0, count)
	
	// 使用事务批量创建
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}

	for i := 0; i < count; i++ {
		code := s.generateCode()
		
		builder := tx.RedeemCode.Create().
			SetCode(code).
			SetAmount(amount).
			SetStatus(ent.RedeemCodeStatusUnused).
			SetCreatedBy(adminID).
			SetBatchNo(batchNo)

		if expiresAt != nil {
			builder.SetExpiresAt(*expiresAt)
		}
		if remark != "" {
			builder.SetRemark(remark)
		}

		redeemCode, err := builder.Save(ctx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建卡密失败: %w", err)
		}
		codes = append(codes, redeemCode)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	s.logger.Info("批量生成卡密成功",
		zap.Int64("admin_id", adminID),
		zap.String("batch_no", batchNo),
		zap.Int("count", count),
		zap.Float64("amount", amount))

	return codes, nil
}

// ValidateCode 验证卡密
func (s *redeemCodeService) ValidateCode(ctx context.Context, code string) (*ent.RedeemCode, error) {
	redeemCode, err := s.db.RedeemCode.Query().
		Where(ent.RedeemCodeCode(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("卡密不存在")
		}
		return nil, fmt.Errorf("查询卡密失败: %w", err)
	}

	// 检查状态
	if redeemCode.Status != ent.RedeemCodeStatusUnused {
		switch redeemCode.Status {
		case ent.RedeemCodeStatusUsed:
			return nil, fmt.Errorf("卡密已被使用")
		case ent.RedeemCodeStatusExpired:
			return nil, fmt.Errorf("卡密已过期")
		case ent.RedeemCodeStatusDisabled:
			return nil, fmt.Errorf("卡密已被禁用")
		default:
			return nil, fmt.Errorf("卡密状态异常")
		}
	}

	// 检查是否过期
	if redeemCode.ExpiresAt != nil && redeemCode.ExpiresAt.Before(time.Now()) {
		// 更新状态为过期
		_, _ = s.db.RedeemCode.UpdateOne(redeemCode).
			SetStatus(ent.RedeemCodeStatusExpired).
			Save(ctx)
		return nil, fmt.Errorf("卡密已过期")
	}

	return redeemCode, nil
}

// RedeemCode 使用卡密充值
func (s *redeemCodeService) RedeemCode(ctx context.Context, userID int64, code string) (*RedeemResult, error) {
	// 验证卡密
	redeemCode, err := s.ValidateCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// 开始事务
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}

	// 更新卡密状态
	now := time.Now()
	_, err = tx.RedeemCode.UpdateOne(redeemCode).
		SetStatus(ent.RedeemCodeStatusUsed).
		SetUsedAt(now).
		SetUsedBy(userID).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新卡密状态失败: %w", err)
	}

	// 增加用户余额
	user, err := tx.User.UpdateOneID(userID).
		AddBalance(redeemCode.Amount).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("充值失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除余额缓存
	s.redis.Del(ctx, fmt.Sprintf("user:balance:%d", userID))

	s.logger.Info("卡密充值成功",
		zap.Int64("user_id", userID),
		zap.String("code", code),
		zap.Float64("amount", redeemCode.Amount))

	return &RedeemResult{
		Code:       code,
		Amount:     redeemCode.Amount,
		Balance:    user.Balance,
		RedeemedAt: now,
	}, nil
}

// GetCodeList 获取卡密列表
func (s *redeemCodeService) GetCodeList(ctx context.Context, status string, batchNo string, page, pageSize int) ([]*ent.RedeemCode, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 构建查询
	query := s.db.RedeemCode.Query()

	if status != "" {
		query = query.Where(ent.RedeemCodeStatusEQ(ent.RedeemCodeStatus(status)))
	}
	if batchNo != "" {
		query = query.Where(ent.RedeemCodeBatchNo(batchNo))
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询卡密总数失败: %w", err)
	}

	// 查询列表
	codes, err := query.
		Order(ent.Desc(ent.RedeemCodeFieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询卡密列表失败: %w", err)
	}

	return codes, total, nil
}

// GetCodeDetail 获取卡密详情
func (s *redeemCodeService) GetCodeDetail(ctx context.Context, codeID int64) (*ent.RedeemCode, error) {
	code, err := s.db.RedeemCode.Get(ctx, codeID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("卡密不存在")
		}
		return nil, fmt.Errorf("查询卡密失败: %w", err)
	}
	return code, nil
}

// DisableCode 禁用卡密
func (s *redeemCodeService) DisableCode(ctx context.Context, codeID int64) error {
	code, err := s.db.RedeemCode.Get(ctx, codeID)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("卡密不存在")
		}
		return fmt.Errorf("查询卡密失败: %w", err)
	}

	if code.Status != ent.RedeemCodeStatusUnused {
		return fmt.Errorf("只能禁用未使用的卡密")
	}

	_, err = s.db.RedeemCode.UpdateOne(code).
		SetStatus(ent.RedeemCodeStatusDisabled).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("禁用卡密失败: %w", err)
	}

	s.logger.Info("禁用卡密成功",
		zap.Int64("code_id", codeID),
		zap.String("code", code.Code))

	return nil
}

// GetUserRedeemRecords 获取用户的充值记录
func (s *redeemCodeService) GetUserRedeemRecords(ctx context.Context, userID int64, page, pageSize int) ([]*RedeemRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// 查询总数
	total, err := s.db.RedeemCode.Query().
		Where(
			ent.RedeemCodeUsedBy(userID),
			ent.RedeemCodeStatusEQ(ent.RedeemCodeStatusUsed),
		).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询充值记录总数失败: %w", err)
	}

	// 查询记录
	codes, err := s.db.RedeemCode.Query().
		Where(
			ent.RedeemCodeUsedBy(userID),
			ent.RedeemCodeStatusEQ(ent.RedeemCodeStatusUsed),
		).
		Order(ent.Desc(ent.RedeemCodeFieldUsedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询充值记录失败: %w", err)
	}

	records := make([]*RedeemRecord, 0, len(codes))
	for _, c := range codes {
		records = append(records, &RedeemRecord{
			ID:         c.ID,
			Code:       s.maskCode(c.Code),
			Amount:     c.Amount,
			RedeemedAt: *c.UsedAt,
		})
	}

	return records, total, nil
}

// ExportCodes 导出卡密
func (s *redeemCodeService) ExportCodes(ctx context.Context, batchNo string) ([]*ent.RedeemCode, error) {
	codes, err := s.db.RedeemCode.Query().
		Where(ent.RedeemCodeBatchNo(batchNo)).
		Order(ent.Asc(ent.RedeemCodeFieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("导出卡密失败: %w", err)
	}

	return codes, nil
}

// generateCode 生成卡密
func (s *redeemCodeService) generateCode() string {
	prefix := s.cfg.RedeemCode.CodePrefix
	if prefix == "" {
		prefix = "MR"
	}
	
	length := s.cfg.RedeemCode.CodeLength
	if length < 10 {
		length = 16
	}

	// 生成随机字符
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length-len(prefix))
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	return prefix + string(b)
}

// generateBatchNo 生成批次号
func (s *redeemCodeService) generateBatchNo() string {
	return fmt.Sprintf("B%s%d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
}

// maskCode 掩码显示卡密
func (s *redeemCodeService) maskCode(code string) string {
	if len(code) <= 8 {
		return strings.Repeat("*", len(code))
	}
	return code[:4] + strings.Repeat("*", len(code)-8) + code[len(code)-4:]
}
