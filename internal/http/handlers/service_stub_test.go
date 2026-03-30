package handlers

import (
	"context"
	"errors"

	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/service"
)

type stubService struct {
	create  func(context.Context, packets.DraftInput) (service.WorkDetail, packets.ValidationErrors, error)
	get     func(context.Context, string) (service.WorkDetail, error)
	list    func(context.Context, int) ([]service.WorkSummary, error)
	review  func(context.Context) (service.ReviewQueueData, error)
	history func(context.Context, string) ([]service.Event, error)
}

func (s stubService) CreateWork(ctx context.Context, input packets.DraftInput) (service.WorkDetail, packets.ValidationErrors, error) {
	if s.create != nil {
		return s.create(ctx, input)
	}
	return service.WorkDetail{}, packets.ValidationErrors{}, errors.New("not implemented")
}

func (s stubService) GetWork(ctx context.Context, id string) (service.WorkDetail, error) {
	if s.get != nil {
		return s.get(ctx, id)
	}
	return service.WorkDetail{}, errors.New("not implemented")
}

func (s stubService) ListRecentWork(ctx context.Context, limit int) ([]service.WorkSummary, error) {
	if s.list != nil {
		return s.list(ctx, limit)
	}
	return nil, errors.New("not implemented")
}

func (s stubService) ReviewQueue(ctx context.Context) (service.ReviewQueueData, error) {
	if s.review != nil {
		return s.review(ctx)
	}
	return service.ReviewQueueData{}, errors.New("not implemented")
}

func (s stubService) WorkHistory(ctx context.Context, id string) ([]service.Event, error) {
	if s.history != nil {
		return s.history(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func validationStubService() stubService {
	return stubService{
		create: func(ctx context.Context, input packets.DraftInput) (service.WorkDetail, packets.ValidationErrors, error) {
			validation := packets.ValidateDraftInput(input)
			if !validation.Empty() {
				return service.WorkDetail{}, validation, nil
			}
			return service.WorkDetail{}, packets.ValidationErrors{}, errors.New("unexpected input passed validation")
		},
	}
}
