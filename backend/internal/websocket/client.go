package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
)

const (
	sendBufferSize = 32
	writeTimeout   = 5 * time.Second
)

type Client struct {
	UserID string
	conn   *websocket.Conn
	hub    *Hub
	send   chan OutboundEvent
}

func NewClient(userID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		UserID: userID,
		conn:   conn,
		hub:    hub,
		send:   make(chan OutboundEvent, sendBufferSize),
	}
}

func (c *Client) Send(event OutboundEvent) {
	select {
	case c.send <- event:
	default:
	}
}

func (c *Client) WriteLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.write(ctx, event); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadLoop(ctx context.Context, handler func(context.Context, InboundEvent)) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		var event InboundEvent
		if err := json.Unmarshal(data, &event); err != nil {
			c.Send(OutboundEvent{Type: EventError, Error: "evento inválido"})
			continue
		}

		handler(ctx, event)
	}
}

func (c *Client) write(ctx context.Context, event OutboundEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	return c.conn.Write(writeCtx, websocket.MessageText, data)
}
