package lsp

// This file contains test helpers for LSP server testing.
// Note: Due to unexported methods in the jsonrpc2.Request interface,
// unit testing the LSP layer directly is challenging. Instead, comprehensive
// tests exist in the /internal/tooling package which this LSP server wraps.
//
// Integration testing should be performed using a real LSP client.
