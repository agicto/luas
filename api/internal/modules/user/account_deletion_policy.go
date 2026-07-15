package user

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// AccountDeletionGuard lets an active starter protect account-owned resources.
type AccountDeletionGuard interface {
	AccountDeletionGuardName() string
	CheckAccountDeletion(ctx context.Context, userID uint) error
}

// AccountDeletionCleaner removes starter-owned user data inside the account transaction.
type AccountDeletionCleaner interface {
	AccountDeletionCleanerName() string
	CleanAccountDeletion(ctx context.Context, userID uint) error
}

// AccountDeletionPolicy composes active starter guards and cleaners in registration order.
type AccountDeletionPolicy struct {
	mu           sync.RWMutex
	guards       []AccountDeletionGuard
	guardNames   map[string]struct{}
	cleaners     []AccountDeletionCleaner
	cleanerNames map[string]struct{}
}

// NewAccountDeletionPolicy creates an empty policy. With no guards, deletion behavior is unchanged.
func NewAccountDeletionPolicy() *AccountDeletionPolicy {
	return &AccountDeletionPolicy{
		guardNames:   make(map[string]struct{}),
		cleanerNames: make(map[string]struct{}),
	}
}

// Register adds one active starter guard.
func (p *AccountDeletionPolicy) Register(guard AccountDeletionGuard) error {
	if p == nil {
		return fmt.Errorf("account deletion policy is required")
	}
	if isNilAccountDeletionParticipant(guard) {
		return fmt.Errorf("account deletion guard is required")
	}
	name := strings.TrimSpace(guard.AccountDeletionGuardName())
	if name == "" {
		return fmt.Errorf("account deletion guard name is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.guardNames[name]; exists {
		return fmt.Errorf("account deletion guard %q is already registered", name)
	}
	p.guardNames[name] = struct{}{}
	p.guards = append(p.guards, guard)
	return nil
}

// RegisterCleaner adds one active starter cleanup participant.
func (p *AccountDeletionPolicy) RegisterCleaner(cleaner AccountDeletionCleaner) error {
	if p == nil {
		return fmt.Errorf("account deletion policy is required")
	}
	if isNilAccountDeletionParticipant(cleaner) {
		return fmt.Errorf("account deletion cleaner is required")
	}
	name := strings.TrimSpace(cleaner.AccountDeletionCleanerName())
	if name == "" {
		return fmt.Errorf("account deletion cleaner name is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.cleanerNames[name]; exists {
		return fmt.Errorf("account deletion cleaner %q is already registered", name)
	}
	p.cleanerNames[name] = struct{}{}
	p.cleaners = append(p.cleaners, cleaner)
	return nil
}

func isNilAccountDeletionParticipant(participant any) bool {
	if participant == nil {
		return true
	}
	value := reflect.ValueOf(participant)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Prepare evaluates every guard before running any cleanup participant.
func (p *AccountDeletionPolicy) Prepare(ctx context.Context, userID uint) error {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	guards := append([]AccountDeletionGuard(nil), p.guards...)
	cleaners := append([]AccountDeletionCleaner(nil), p.cleaners...)
	p.mu.RUnlock()

	for _, guard := range guards {
		if err := guard.CheckAccountDeletion(ctx, userID); err != nil {
			return err
		}
	}
	for _, cleaner := range cleaners {
		if err := cleaner.CleanAccountDeletion(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// Check evaluates guards until one blocks deletion.
func (p *AccountDeletionPolicy) Check(ctx context.Context, userID uint) error {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	guards := append([]AccountDeletionGuard(nil), p.guards...)
	p.mu.RUnlock()

	for _, guard := range guards {
		if err := guard.CheckAccountDeletion(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}
