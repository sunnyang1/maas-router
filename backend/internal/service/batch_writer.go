// Package service 业务服务层
// 批量写入服务 - 高吞吐场景下的数据库批量写入
package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// BatchWriterStats 批量写入统计信息
type BatchWriterStats struct {
	PendingCount  int64     `json:"pending_count"`
	FlushedTotal  int64     `json:"flushed_total"`
	FlushCount    int64     `json:"flush_count"`
	FailedCount   int64     `json:"failed_count"`
	LastFlushTime time.Time `json:"last_flush_time"`
	BufferSize    int       `json:"buffer_size"`
}

// BatchWriter 批量写入接口
type BatchWriter interface {
	// Submit 提交一条记录到批量写入缓冲区
	Submit(record interface{}) error

	// Flush 强制刷新所有待处理记录
	Flush() error

	// Start 启动后台定时刷新 goroutine
	Start(ctx context.Context)

	// Stop 停止批量写入器并刷新剩余记录
	Stop() error

	// Stats 返回批量写入统计信息
	Stats() BatchWriterStats
}

// BatchWriterOption 批量写入器配置选项
type BatchWriterOption func(*batchWriter)

// WithBufferSize 设置缓冲区大小
func WithBufferSize(size int) BatchWriterOption {
	return func(w *batchWriter) {
		w.bufferSize = size
	}
}

// WithMaxBatchSize 设置最大批量大小
func WithMaxBatchSize(size int) BatchWriterOption {
	return func(w *batchWriter) {
		w.maxBatchSize = size
	}
}

// WithFlushInterval 设置刷新间隔
func WithFlushInterval(interval time.Duration) BatchWriterOption {
	return func(w *batchWriter) {
		w.flushInterval = interval
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *zap.Logger) BatchWriterOption {
	return func(w *batchWriter) {
		w.logger = logger
	}
}

type batchWriter struct {
	buffer        chan interface{}
	flushFunc     func(ctx context.Context, records []interface{}) error
	mu            sync.Mutex
	logger        *zap.Logger
	flushInterval time.Duration
	maxBatchSize  int
	bufferSize    int

	// 统计信息（使用原子操作）
	pendingCount int64
	flushedTotal int64
	flushCount   int64
	failedCount  int64
	lastFlush    int64 // UnixNano timestamp

	// 控制信号
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBatchWriter 创建批量写入器
// flushFunc: 批量写入回调函数，接收一批记录并执行写入
func NewBatchWriter(
	flushFunc func(ctx context.Context, records []interface{}) error,
	opts ...BatchWriterOption,
) BatchWriter {
	w := &batchWriter{
		flushFunc:     flushFunc,
		bufferSize:    10000, // 默认缓冲区大小
		maxBatchSize:  500,   // 默认最大批量大小
		flushInterval: 3 * time.Second, // 默认刷新间隔
		logger:        zap.NewNop(),
		stopCh:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	w.buffer = make(chan interface{}, w.bufferSize)

	return w
}

// Submit 提交一条记录到缓冲区
func (w *batchWriter) Submit(record interface{}) error {
	select {
	case w.buffer <- record:
		atomic.AddInt64(&w.pendingCount, 1)
		return nil
	default:
		// 缓冲区已满，触发立即刷新
		go func() {
			if err := w.Flush(); err != nil && w.logger != nil {
				w.logger.Warn("缓冲区满时刷新失败", zap.Error(err))
			}
		}()
		// 等待一小段时间后重试
		select {
		case w.buffer <- record:
			atomic.AddInt64(&w.pendingCount, 1)
			return nil
		default:
			return fmt.Errorf("批量写入缓冲区已满，记录被丢弃")
		}
	}
}

// Flush 强制刷新所有待处理记录
func (w *batchWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flush(context.Background())
}

// flush 内部刷新方法（需要调用方持有锁）
func (w *batchWriter) flush(ctx context.Context) error {
	batch := make([]interface{}, 0, w.maxBatchSize)

	// 从缓冲区读取记录
	for {
		select {
		case record := <-w.buffer:
			batch = append(batch, record)
			if len(batch) >= w.maxBatchSize {
				break
			}
		default:
			// 缓冲区已空
			goto done
		}
	}

done:
	if len(batch) == 0 {
		return nil
	}

	// 执行批量写入
	if err := w.flushFunc(ctx, batch); err != nil {
		atomic.AddInt64(&w.failedCount, int64(len(batch)))
		return fmt.Errorf("批量写入 %d 条记录失败: %w", len(batch), err)
	}

	// 更新统计
	atomic.AddInt64(&w.pendingCount, -int64(len(batch)))
	atomic.AddInt64(&w.flushedTotal, int64(len(batch)))
	atomic.AddInt64(&w.flushCount, 1)
	atomic.StoreInt64(&w.lastFlush, time.Now().UnixNano())

	if w.logger != nil {
		w.logger.Debug("批量写入完成",
			zap.Int("batch_size", len(batch)),
			zap.Int64("pending", atomic.LoadInt64(&w.pendingCount)),
			zap.Int64("flushed_total", atomic.LoadInt64(&w.flushedTotal)))
	}

	return nil
}

// Start 启动后台定时刷新 goroutine
func (w *batchWriter) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.flushInterval)
		defer ticker.Stop()

		if w.logger != nil {
			w.logger.Info("批量写入器已启动",
				zap.Int("buffer_size", w.bufferSize),
				zap.Int("max_batch_size", w.maxBatchSize),
				zap.Duration("flush_interval", w.flushInterval))
		}

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("批量写入器收到上下文取消信号")
				return
			case <-w.stopCh:
				w.logger.Info("批量写入器收到停止信号")
				return
			case <-ticker.C:
				// 检查是否有待处理记录
				if atomic.LoadInt64(&w.pendingCount) > 0 {
					w.mu.Lock()
					if err := w.flush(ctx); err != nil && w.logger != nil {
						w.logger.Error("定时刷新失败", zap.Error(err))
					}
					w.mu.Unlock()
				}
			}
		}
	}()
}

// Stop 停止批量写入器并刷新剩余记录
func (w *batchWriter) Stop() error {
	// 发送停止信号
	close(w.stopCh)

	// 等待后台 goroutine 结束
	w.wg.Wait()

	// 最后一次刷新
	if err := w.Flush(); err != nil && w.logger != nil {
		w.logger.Error("停止时最终刷新失败", zap.Error(err))
		return err
	}

	if w.logger != nil {
		w.logger.Info("批量写入器已停止",
			zap.Int64("total_flushed", atomic.LoadInt64(&w.flushedTotal)),
			zap.Int64("total_flushes", atomic.LoadInt64(&w.flushCount)),
			zap.Int64("total_failed", atomic.LoadInt64(&w.failedCount)))
	}

	return nil
}

// Stats 返回批量写入统计信息
func (w *batchWriter) Stats() BatchWriterStats {
	lastFlushNano := atomic.LoadInt64(&w.lastFlush)
	var lastFlushTime time.Time
	if lastFlushNano > 0 {
		lastFlushTime = time.Unix(0, lastFlushNano)
	}

	return BatchWriterStats{
		PendingCount:  atomic.LoadInt64(&w.pendingCount),
		FlushedTotal:  atomic.LoadInt64(&w.flushedTotal),
		FlushCount:    atomic.LoadInt64(&w.flushCount),
		FailedCount:   atomic.LoadInt64(&w.failedCount),
		LastFlushTime: lastFlushTime,
		BufferSize:    len(w.buffer),
	}
}
