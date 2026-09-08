package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type Scheduler struct {
	mu              sync.RWMutex
	workerPool      *WorkerPool
	jobRepo         *repository.JobRepository
	businessRepo    *repository.BusinessRepository
	interval        time.Duration
	nextRunAt       time.Time
	lastRunAt       *time.Time
	autoEnabled     bool
	defaultQuery    string
	defaultLocation string
	stopChan        chan struct{}
}

func NewScheduler(workerPool *WorkerPool, jobRepo *repository.JobRepository, businessRepo *repository.BusinessRepository) *Scheduler {
	now := time.Now().UTC()
	// Next daily run scheduled for next 00:00 UTC (or next 24h cycle)
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	if nextMidnight.Sub(now) < 1*time.Hour {
		nextMidnight = nextMidnight.Add(24 * time.Hour)
	}

	return &Scheduler{
		workerPool:      workerPool,
		jobRepo:         jobRepo,
		businessRepo:    businessRepo,
		interval:        24 * time.Hour,
		nextRunAt:       nextMidnight,
		autoEnabled:     true,
		defaultQuery:    "Specialty Coffee",
		defaultLocation: "Seattle, WA",
		stopChan:        make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-s.stopChan:
				ticker.Stop()
				return
			case now := <-ticker.C:
				s.checkAndTrigger(ctx, now.UTC())
			}
		}
	}()
}

func (s *Scheduler) checkAndTrigger(ctx context.Context, now time.Time) {
	s.mu.Lock()
	if !s.autoEnabled {
		s.mu.Unlock()
		return
	}

	if now.After(s.nextRunAt) || now.Equal(s.nextRunAt) {
		slog.Info("Triggering scheduled daily ingestion cycle", "time", now.Format(time.RFC3339))
		s.lastRunAt = &now
		s.nextRunAt = now.Add(s.interval)
		query := s.defaultQuery
		location := s.defaultLocation
		s.mu.Unlock()

		job := &model.ScrapeJob{
			Status:         model.JobStatusPending,
			Query:          query,
			Location:       location,
			Category:       "Scheduled Batch Ingestion",
			MaxResults:     50,
			EnrichContacts: true,
			Metadata: map[string]interface{}{
				"source": "scheduler_daily",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := s.jobRepo.Create(ctx, job); err == nil {
			s.workerPool.SubmitJob(job)
		}
		return
	}
	s.mu.Unlock()
}

type ScheduleInfo struct {
	NextRunAt        time.Time  `json:"next_run_at"`
	CountdownSeconds int64      `json:"countdown_seconds"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	IntervalHours    int        `json:"interval_hours"`
	AutoEnabled      bool       `json:"auto_enabled"`
	TargetQuery      string     `json:"target_query"`
	TargetLocation   string     `json:"target_location"`
}

func (s *Scheduler) GetScheduleInfo() ScheduleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	countdown := int64(s.nextRunAt.Sub(now).Seconds())
	if countdown < 0 {
		countdown = 0
	}

	return ScheduleInfo{
		NextRunAt:        s.nextRunAt,
		CountdownSeconds: countdown,
		LastRunAt:        s.lastRunAt,
		IntervalHours:    int(s.interval.Hours()),
		AutoEnabled:      s.autoEnabled,
		TargetQuery:      s.defaultQuery,
		TargetLocation:   s.defaultLocation,
	}
}

func (s *Scheduler) UpdateSchedule(intervalHours int, autoEnabled bool, targetQuery, targetLocation string) ScheduleInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	if intervalHours <= 0 {
		intervalHours = 24
	}
	s.interval = time.Duration(intervalHours) * time.Hour
	s.autoEnabled = autoEnabled
	if targetQuery != "" {
		s.defaultQuery = targetQuery
	}
	if targetLocation != "" {
		s.defaultLocation = targetLocation
	}

	now := time.Now().UTC()
	s.nextRunAt = now.Add(s.interval)

	return ScheduleInfo{
		NextRunAt:        s.nextRunAt,
		CountdownSeconds: int64(s.interval.Seconds()),
		LastRunAt:        s.lastRunAt,
		IntervalHours:    int(s.interval.Hours()),
		AutoEnabled:      s.autoEnabled,
		TargetQuery:      s.defaultQuery,
		TargetLocation:   s.defaultLocation,
	}
}

func (s *Scheduler) TriggerNow(ctx context.Context, query, location string, maxResults int) (*model.ScrapeJob, error) {
	s.mu.Lock()
	now := time.Now().UTC()
	s.lastRunAt = &now
	s.nextRunAt = now.Add(s.interval)
	defaultQ := s.defaultQuery
	defaultLoc := s.defaultLocation
	s.mu.Unlock()

	if query == "" {
		query = defaultQ
	}
	if location == "" {
		location = defaultLoc
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	job := &model.ScrapeJob{
		Status:         model.JobStatusPending,
		Query:          query,
		Location:       location,
		Category:       "Manual Ingestion Trigger",
		MaxResults:     maxResults,
		EnrichContacts: true,
		Metadata: map[string]interface{}{
			"source": "manual_timer_trigger",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	s.workerPool.SubmitJob(job)
	return job, nil
}
