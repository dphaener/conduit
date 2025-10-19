package middleware

import (
	"net/http"
)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain represents a composable chain of middleware
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Use adds a middleware to the chain
func (c *Chain) Use(m Middleware) *Chain {
	c.middlewares = append(c.middlewares, m)
	return c
}

// Apply wraps the given handler with all middleware in the chain
// Middleware is applied in reverse order (last added wraps first)
// This ensures that middleware added first executes first
func (c *Chain) Apply(handler http.Handler) http.Handler {
	// Apply middleware in reverse order
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// Then is an alias for Apply that returns the wrapped handler
func (c *Chain) Then(handler http.Handler) http.Handler {
	return c.Apply(handler)
}

// ThenFunc wraps an http.HandlerFunc with the middleware chain
func (c *Chain) ThenFunc(handlerFunc http.HandlerFunc) http.Handler {
	return c.Apply(handlerFunc)
}

// Append creates a new chain by appending middleware to the current chain
// This allows for creating variations of a base chain without mutation
func (c *Chain) Append(middlewares ...Middleware) *Chain {
	newMiddlewares := make([]Middleware, len(c.middlewares)+len(middlewares))
	copy(newMiddlewares, c.middlewares)
	copy(newMiddlewares[len(c.middlewares):], middlewares)
	return &Chain{middlewares: newMiddlewares}
}

// Extend adds multiple middleware to the chain
func (c *Chain) Extend(middlewares ...Middleware) *Chain {
	c.middlewares = append(c.middlewares, middlewares...)
	return c
}
