/**
 * Conduit Hot Reload Client
 * Connects to the development server via WebSocket for live reload functionality
 */

(function() {
  'use strict';

  const RECONNECT_DELAY = 2000; // 2 seconds
  const MAX_RECONNECT_ATTEMPTS = 10;

  let ws = null;
  let reconnectAttempts = 0;
  let reconnectTimeout = null;

  // Get WebSocket URL
  function getWebSocketURL() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.hostname;
    const port = window.location.port || '3000';
    return `${protocol}//${host}:${port}/__conduit_reload`;
  }

  // Create status indicator
  function createStatusIndicator() {
    const indicator = document.createElement('div');
    indicator.id = 'conduit-status';
    indicator.style.cssText = `
      position: fixed;
      bottom: 20px;
      right: 20px;
      padding: 8px 12px;
      background: #4CAF50;
      color: white;
      border-radius: 4px;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      font-size: 12px;
      z-index: 999999;
      box-shadow: 0 2px 8px rgba(0,0,0,0.2);
      transition: all 0.3s ease;
      display: none;
    `;
    indicator.textContent = '● Connected';
    document.body.appendChild(indicator);
    return indicator;
  }

  // Create error overlay
  function createErrorOverlay() {
    const overlay = document.createElement('div');
    overlay.id = 'conduit-error-overlay';
    overlay.style.cssText = `
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.9);
      color: white;
      z-index: 9999999;
      overflow: auto;
      display: none;
      font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
    `;

    const content = document.createElement('div');
    content.style.cssText = `
      max-width: 900px;
      margin: 40px auto;
      padding: 20px;
    `;

    overlay.appendChild(content);
    document.body.appendChild(overlay);

    // Close on click
    overlay.addEventListener('click', function(e) {
      if (e.target === overlay) {
        hideErrorOverlay();
      }
    });

    return overlay;
  }

  // Show status indicator
  function showStatus(message, color) {
    const indicator = document.getElementById('conduit-status') || createStatusIndicator();
    indicator.textContent = message;
    indicator.style.background = color;
    indicator.style.display = 'block';

    // Auto-hide success messages
    if (color === '#4CAF50') {
      setTimeout(() => {
        indicator.style.display = 'none';
      }, 3000);
    }
  }

  // Show error overlay
  function showErrorOverlay(error) {
    const overlay = document.getElementById('conduit-error-overlay') || createErrorOverlay();
    const content = overlay.firstChild;

    let html = `
      <div style="margin-bottom: 20px;">
        <h1 style="color: #ef5350; margin: 0 0 10px 0; font-size: 24px;">
          ❌ Compilation Failed
        </h1>
        <p style="color: #ccc; margin: 0; font-size: 14px;">
          Fix the errors below and save to reload
        </p>
      </div>
    `;

    if (error) {
      html += `
        <div style="background: #1e1e1e; padding: 20px; border-radius: 8px; margin-bottom: 20px; border-left: 4px solid #ef5350;">
          <div style="margin-bottom: 10px;">
            <span style="background: #ef5350; color: white; padding: 2px 8px; border-radius: 3px; font-size: 11px; font-weight: bold;">
              ${error.phase ? error.phase.toUpperCase() : 'ERROR'}
            </span>
            ${error.code ? `<span style="color: #888; margin-left: 10px;">${error.code}</span>` : ''}
          </div>
          <div style="color: white; font-size: 14px; line-height: 1.5; margin-bottom: 10px;">
            ${escapeHtml(error.message)}
          </div>
          ${error.file ? `
            <div style="color: #888; font-size: 12px;">
              ${error.file}${error.line ? `:${error.line}` : ''}${error.column ? `:${error.column}` : ''}
            </div>
          ` : ''}
        </div>
      `;
    }

    html += `
      <div style="text-align: center; margin-top: 30px;">
        <button onclick="document.getElementById('conduit-error-overlay').style.display='none'"
                style="background: #333; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; font-size: 14px;">
          Dismiss (click outside to close)
        </button>
      </div>
    `;

    content.innerHTML = html;
    overlay.style.display = 'block';
  }

  // Hide error overlay
  function hideErrorOverlay() {
    const overlay = document.getElementById('conduit-error-overlay');
    if (overlay) {
      overlay.style.display = 'none';
    }
  }

  // Escape HTML
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  // Connect to WebSocket
  function connect() {
    const url = getWebSocketURL();
    console.log('[Conduit] Connecting to reload server:', url);

    try {
      ws = new WebSocket(url);

      ws.onopen = function() {
        console.log('[Conduit] Connected to reload server');
        reconnectAttempts = 0;
        showStatus('● Connected', '#4CAF50');
        hideErrorOverlay();
      };

      ws.onmessage = function(event) {
        try {
          const message = JSON.parse(event.data);
          console.log('[Conduit] Message:', message);
          handleMessage(message);
        } catch (err) {
          console.error('[Conduit] Failed to parse message:', err);
        }
      };

      ws.onerror = function(error) {
        console.error('[Conduit] WebSocket error:', error);
      };

      ws.onclose = function() {
        console.log('[Conduit] Disconnected from reload server');
        showStatus('● Disconnected', '#FF9800');
        attemptReconnect();
      };

    } catch (err) {
      console.error('[Conduit] Failed to create WebSocket:', err);
      attemptReconnect();
    }
  }

  // Handle incoming messages
  function handleMessage(message) {
    switch (message.type) {
      case 'building':
        console.log('[Conduit] Building...', message.files);
        showStatus('⚙ Building...', '#2196F3');
        break;

      case 'success':
        console.log('[Conduit] Build successful', message.duration + 'ms');
        showStatus(`✓ Built in ${message.duration}ms`, '#4CAF50');
        hideErrorOverlay();
        break;

      case 'reload':
        console.log('[Conduit] Reloading...', message.scope);

        // Hide error overlay before reload
        hideErrorOverlay();

        // Reload based on scope
        if (message.scope === 'ui') {
          // For UI changes, just reload CSS
          reloadCSS();
        } else {
          // For backend changes, full reload
          showStatus('🔄 Reloading...', '#2196F3');
          setTimeout(() => {
            window.location.reload();
          }, 100);
        }
        break;

      case 'error':
        console.error('[Conduit] Build error:', message.error);
        showStatus('❌ Build failed', '#ef5350');
        if (message.error) {
          showErrorOverlay(message.error);
        }
        break;

      default:
        console.log('[Conduit] Unknown message type:', message.type);
    }
  }

  // Reload CSS without full page reload
  function reloadCSS() {
    console.log('[Conduit] Reloading CSS...');
    const links = document.querySelectorAll('link[rel="stylesheet"]');
    links.forEach(link => {
      const href = link.href;
      const url = new URL(href);
      url.searchParams.set('_reload', Date.now());
      link.href = url.toString();
    });
    showStatus('🎨 CSS reloaded', '#4CAF50');
  }

  // Attempt to reconnect
  function attemptReconnect() {
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.log('[Conduit] Max reconnect attempts reached');
      showStatus('● Connection lost', '#f44336');
      return;
    }

    reconnectAttempts++;
    console.log(`[Conduit] Reconnecting... (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);

    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }

    reconnectTimeout = setTimeout(() => {
      connect();
    }, RECONNECT_DELAY);
  }

  // Initialize on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', connect);
  } else {
    connect();
  }

  // Expose reload function to window for manual triggers
  window.__conduit_reload = function() {
    console.log('[Conduit] Manual reload triggered');
    window.location.reload();
  };

})();
