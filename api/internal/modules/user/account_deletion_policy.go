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

// AccountDeletionPolicy composes active starter guards in deterministic registration order.
type AccountDeletionPolicy struct {
	mu     sync.RWMutex
	guards []AccountDeletionGuard
	names  map[string]struct{}
}

// NewAccountDeletionPolicy creates an empty policy. With no guards, deletion behavior is unchanged.
func NewAccountDeletionPolicy() *AccountDeletionPolicy {
	return &AccountDeletionPolicy{names: make(map[string]struct{})}
}

// Register adds one active starter guard.
func (p *AccountDeletionPolicy) Register(guard AccountDeletionGuard) error {
	if p == nil {
		return fmt.Errorf("account deletion policy is required")
	}
	if isNilAccountDeletionGuard(guard) {
		return fmt.Errorf("account deletion guard is required")
	}
	name := strings.TrimSpace(guard.AccountDeletionGuardName())
	if name == "" {
		return fmt.Errorf("account deletion guard name is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.names[name]; exists {
		return fmt.Errorf("account deletion guard %q is already registered", name)
	}
	p.names[name] = struct{}{}
	p.guards = append(p.guards, guard)
	return nil
}

func isNilAccountDeletionGuard(guard AccountDeletionGuard) bool {
	if guard == nil {
		return true
	}
	value := reflect.ValueOf(guard)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
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
