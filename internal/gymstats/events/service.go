package events

import (
	"context"
	"fmt"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"go.opentelemetry.io/otel/codes"
)

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) AddTrainingStart(ctx context.Context, ts TrainingStart) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.add.trainingstart")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	tsEvent := NewTrainingStartEvent(ts)
	event, err := s.repo.Add(ctx, tsEvent)
	if err != nil {
		return 0, fmt.Errorf("add training start event: %w", err)
	}
	return event.ID, nil
}

func (s *Service) AddTrainingFinish(ctx context.Context, tf TrainingFinish) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.add.trainingfinish")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	tfEvent := NewTrainingFinishEvent(tf)
	event, err := s.repo.Add(ctx, tfEvent)
	if err != nil {
		return 0, fmt.Errorf("add training finish event: %w", err)
	}
	return event.ID, nil
}

func (s *Service) AddWeightReport(ctx context.Context, wr WeightReport) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.add.weightreport")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	wrEvent := NewWeightReportEvent(wr)
	event, err := s.repo.Add(ctx, wrEvent)
	if err != nil {
		return 0, fmt.Errorf("add weight report event: %w", err)
	}
	return event.ID, nil
}

func (s *Service) AddPainReport(ctx context.Context, pr PainReport) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.add.painreport")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	prEvent := NewPainReportEvent(pr)
	event, err := s.repo.Add(ctx, prEvent)
	if err != nil {
		return 0, fmt.Errorf("add pain report event: %w", err)
	}
	return event.ID, nil
}

func (s *Service) List(ctx context.Context, params ListParams) (_ []*Event, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.list")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	events, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

func (s *Service) Count(ctx context.Context, params EventParams) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "service.gymstats.events.count")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	count, err := s.repo.Count(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}
