package service

import (
	"fmt"
	"Haruki-Service-API/pkg/masterdata"
	"time"
)

// EventSearchService 活动搜索服务
type EventSearchService struct {
	repo   *MasterDataService
	parser *EventParser
}

// NewEventSearchService 创建活动搜索服务
func NewEventSearchService(repo *MasterDataService, parser *EventParser) *EventSearchService {
	return &EventSearchService{
		repo:   repo,
		parser: parser,
	}
}

// Search 单个活动搜索
func (s *EventSearchService) Search(query string) (*masterdata.Event, error) {
	// 1. Parse
	info, err := s.parser.Parse(query)
	if err != nil {
		return nil, err
	}

	// 2. Dispatch
	switch info.Type {
	case QueryTypeEventID:
		return s.repo.GetEventByID(info.EventID)

	case QueryTypeEventSeq:
		events := s.repo.GetEvents()
		// Handle keyword: current, next, prev
		if info.Keyword != "" {
			return s.findEventByKeyword(events, info.Keyword)
		}
		// Handle index
		return s.findEventByIndex(events, info.Index)

	case QueryTypeEventBan:
		banEvents := s.repo.GetBanEvents(info.BanCharID)
		if len(banEvents) == 0 {
			return nil, fmt.Errorf("no ban events found for charID %d", info.BanCharID)
		}
		// Index is 1-based in command (mnr1)
		if info.BanSeq < 1 || info.BanSeq > len(banEvents) {
			return nil, fmt.Errorf("ban event index out of range: %d (total: %d)", info.BanSeq, len(banEvents))
		}
		return banEvents[info.BanSeq-1], nil

	case QueryTypeEventFilter:
		// For single search, return the latest matching event?
		// Or should filter only be for list?
		// Typically users expect "25h" to return the latest 25h event if used in single context.
		events := s.repo.FilterEvents(info.Filter)
		if len(events) == 0 {
			return nil, fmt.Errorf("no events found for filter")
		}
		return events[len(events)-1], nil
	}

	return nil, fmt.Errorf("unknown query type")
}

// SearchList 活动列表搜索
func (s *EventSearchService) SearchList(query string) ([]*masterdata.Event, error) {
	// 1. Parse
	info, err := s.parser.Parse(query)
	if err != nil {
		return nil, err
	}

	// 2. Dispatch
	switch info.Type {
	case QueryTypeEventFilter:
		events := s.repo.FilterEvents(info.Filter)
		if len(events) == 0 {
			return nil, fmt.Errorf("no events found for filter")
		}
		return events, nil

	case QueryTypeEventBan:
		// Return all ban events for this char?
		// If command was mnr1, it implies single.
		// Use "mnr box" or similar for all?
		// If parser parsed it as 'BanSeq', it's specific.
		// But maybe we want logic to list all if seq is missing?
		// Current parser logic requires number for tryParseBanEvent.
		// If query is just "mnr", it goes to Filter logic (CharID set).
		// So "mnr" -> Filter -> All events with mnr cards (which includes ban events).
		// Strict "Ban Events Only" might need a new filter flag.
		// For now, respect explicit parsing results.
		single, err := s.Search(query)
		if err != nil {
			return nil, err
		}
		return []*masterdata.Event{single}, nil

	default:
		// Other types denote a single event, wrap in list
		single, err := s.Search(query)
		if err != nil {
			return nil, err
		}
		return []*masterdata.Event{single}, nil
	}
}

// Helper: findEventByIndex
func (s *EventSearchService) findEventByIndex(events []*masterdata.Event, index int) (*masterdata.Event, error) {
	count := len(events)
	if index < 0 {
		idx := count + index
		if idx < 0 || idx >= count {
			return nil, fmt.Errorf("index out of range: %d", index)
		}
		return events[idx], nil
	}
	// Postive index (ID? or Index?)
	// Python: -1 (Index), 123 (ID).
	// But Parser handles numeric as ID.
	// So positive index here might be explicitly requested for some reason,
	// but Parser usually puts positive numbers into ID if they look like IDs.
	// Users barely use positive index for events unless "3rd event".
	// Let's assume 1-based index if it reached here.
	if index < 1 || index > count {
		return nil, fmt.Errorf("index out of range: %d", index)
	}
	return events[index-1], nil
}

// Helper: findEventByKeyword
func (s *EventSearchService) findEventByKeyword(events []*masterdata.Event, keyword string) (*masterdata.Event, error) {
	now := time.Now().UnixNano() / 1e6 // ms

	var cur, prev, next *masterdata.Event

	for i, e := range events {
		// Event duration: StartAt ~ AggregateAt
		// ClosedAt is usually later (exchange end).
		// Active event: StartAt <= now <= AggregateAt
		if e.StartAt <= now && now <= e.AggregateAt {
			cur = events[i]
		}
		if e.AggregateAt < now {
			prev = events[i] // Keep updating to find the latest "previous"
		}
		if e.StartAt > now {
			if next == nil { // Found first "next"
				next = events[i]
			}
		}
	}

	switch keyword {
	case "current":
		if cur != nil {
			return cur, nil
		}
		// Fallback: Python logic 'get_current_event' has complex fallback (prev or next).
		// Let's return prev if no current (event ended, waiting for results)
		if prev != nil {
			return prev, nil
		}
		return next, nil // No prev means we are before first event?
	case "next":
		if next != nil {
			return next, nil
		}
		return nil, fmt.Errorf("no next event found")
	case "prev":
		if prev != nil {
			return prev, nil
		}
		return nil, fmt.Errorf("no previous event found")
	}
	return nil, fmt.Errorf("unknown keyword: %s", keyword)
}
