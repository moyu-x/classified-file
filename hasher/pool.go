package hasher

import (
	"sync"

	"github.com/panjf2000/ants/v2"

	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
)

type HashTask struct {
	Path string
	Size int64
}

type HashResult struct {
	Path  string
	Hash  uint64
	Size  int64
	Error error
}

type HashPool struct {
	workers int
	tasks   chan HashTask
	results chan HashResult
	wg      sync.WaitGroup
	pool    *ants.Pool
}

func NewHashPool(workers int) *HashPool {
	logger.Get().Info().Msgf("创建哈希计算池，工作线程数: %d", workers)
	return &HashPool{
		workers: workers,
		tasks:   make(chan HashTask, internal.DefaultBufferSize),
		results: make(chan HashResult, internal.DefaultBufferSize),
	}
}

func (p *HashPool) Start() {
	logger.Get().Info().Msgf("启动哈希计算池，启动 %d 个工作线程", p.workers)

	var err error
	p.pool, err = ants.NewPool(p.workers)

	if err != nil {
		logger.Get().Error().Err(err).Msg("创建 goroutine 池失败")
		panic(err)
	}

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		p.pool.Submit(p.worker)
	}
}

func (p *HashPool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		hash, err := CalculateHash(task.Path)
		p.results <- HashResult{
			Path:  task.Path,
			Hash:  hash,
			Size:  task.Size,
			Error: err,
		}
	}
}

func (p *HashPool) AddTask(task HashTask) {
	p.tasks <- task
}

func (p *HashPool) Results() <-chan HashResult {
	return p.results
}

func (p *HashPool) Close() {
	logger.Get().Info().Msg("关闭哈希计算池")

	close(p.tasks)

	if p.pool != nil {
		p.pool.Release()
	}

	close(p.results)
}
