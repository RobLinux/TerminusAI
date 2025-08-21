package config

import (
	"context"
	"fmt"
)

// ContextKey represents a key for context values
type ContextKey string

const (
	// ConfigManagerKey is the context key for the configuration manager
	ConfigManagerKey ContextKey = "config_manager"
)

// NewContext creates a new context with the configuration manager
func NewContext(ctx context.Context, cm *ConfigManager) context.Context {
	return context.WithValue(ctx, ConfigManagerKey, cm)
}

// FromContext extracts the configuration manager from context
func FromContext(ctx context.Context) (*ConfigManager, error) {
	cm, ok := ctx.Value(ConfigManagerKey).(*ConfigManager)
	if !ok {
		return nil, fmt.Errorf("configuration manager not found in context")
	}
	return cm, nil
}

// MustFromContext extracts the configuration manager from context, panicking if not found
func MustFromContext(ctx context.Context) *ConfigManager {
	cm, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return cm
}

// WithDefaults creates a context with the default configuration manager
func WithDefaults(ctx context.Context) context.Context {
	return NewContext(ctx, GetConfigManager())
}