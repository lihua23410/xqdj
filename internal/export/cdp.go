package export

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type cdpConn struct {
	conn *websocket.Conn

	mu      sync.Mutex
	next    int64
	pending map[int64]chan cdpReply
	closed  bool
}

type cdpReply struct {
	result json.RawMessage
	err    error
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *cdpError) Error() string {
	return fmt.Sprintf("cdp %d: %s", e.Code, e.Message)
}

func dialCDP(url string) (*cdpConn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	c := &cdpConn{conn: conn, pending: make(map[int64]chan cdpReply)}
	go c.readLoop()
	return c, nil
}

func (c *cdpConn) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
	_ = c.conn.Close()
}

func (c *cdpConn) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.failAll(err)
			return
		}
		var msg struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *cdpError       `json:"error"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.ID == 0 {
			continue // 事件，逐帧导出不需要
		}
		c.mu.Lock()
		ch := c.pending[msg.ID]
		delete(c.pending, msg.ID)
		c.mu.Unlock()
		if ch == nil {
			continue
		}
		reply := cdpReply{result: msg.Result}
		if msg.Error != nil {
			reply.err = msg.Error
		}
		ch <- reply
		close(ch)
	}
}

func (c *cdpConn) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		ch <- cdpReply{err: err}
		close(ch)
		delete(c.pending, id)
	}
}

func (c *cdpConn) call(ctx context.Context, method string, params any, out any) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("cdp closed")
	}
	c.next++
	id := c.next
	ch := make(chan cdpReply, 1)
	c.pending[id] = ch
	err := c.conn.WriteJSON(map[string]any{"id": id, "method": method, "params": params})
	c.mu.Unlock()
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case reply, ok := <-ch:
		if !ok {
			return fmt.Errorf("cdp closed")
		}
		if reply.err != nil {
			return reply.err
		}
		if out != nil && len(reply.result) > 0 {
			return json.Unmarshal(reply.result, out)
		}
		return nil
	}
}

type evalResult struct {
	Result struct {
		Type        string          `json:"type"`
		Value       json.RawMessage `json:"value"`
		Description string          `json:"description"`
	} `json:"result"`
	ExceptionDetails *struct {
		Text string `json:"text"`
	} `json:"exceptionDetails"`
}

func (c *cdpConn) eval(ctx context.Context, expr string) (json.RawMessage, error) {
	var res evalResult
	err := c.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expr,
		"awaitPromise":  true,
		"returnByValue": true,
	}, &res)
	if err != nil {
		return nil, err
	}
	if res.ExceptionDetails != nil {
		return nil, fmt.Errorf("页面脚本出错: %s", res.ExceptionDetails.Text)
	}
	return res.Result.Value, nil
}

func (c *cdpConn) evalBool(ctx context.Context, expr string) (bool, error) {
	v, err := c.eval(ctx, expr)
	if err != nil {
		return false, err
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, err
	}
	return b, nil
}

func (c *cdpConn) evalString(ctx context.Context, expr string) (string, error) {
	v, err := c.eval(ctx, expr)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", err
	}
	return s, nil
}

func (c *cdpConn) navigate(ctx context.Context, url string) error {
	return c.call(ctx, "Page.navigate", map[string]any{"url": url}, nil)
}

func (c *cdpConn) setViewport(ctx context.Context, width, height int) error {
	return c.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width":             width,
		"height":            height,
		"deviceScaleFactor": 1,
		"mobile":            false,
	}, nil)
}

func (c *cdpConn) screenshot(ctx context.Context) ([]byte, error) {
	var res struct {
		Data string `json:"data"`
	}
	if err := c.call(ctx, "Page.captureScreenshot", map[string]any{
		"format":                "png",
		"fromSurface":           true,
		"captureBeyondViewport": false,
	}, &res); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.Data)
}

func (c *cdpConn) waitFor(ctx context.Context, expr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := c.evalBool(ctx, expr)
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("等待页面超时: %w", err)
			}
			return fmt.Errorf("等待页面超时")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
