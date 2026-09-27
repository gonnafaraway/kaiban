package handlers

import (
	"bufio"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	httptransport "kaiban/internal/api/transport/http"
)

func (d Deps) sse(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	ch := d.Hub.Subscribe()
	defer d.Hub.Unsubscribe(ch)
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var wrap map[string]any
				ev := httptransport.EventMessage
				if err := json.Unmarshal(msg, &wrap); err == nil {
					if s, ok := wrap[httptransport.FieldEvent].(string); ok && s != "" {
						ev = s
					}
				}
				_, _ = w.WriteString("event: " + ev + "\n")
				_, _ = w.WriteString("data: " + string(msg) + "\n\n")
				_ = w.Flush()
			case <-ticker.C:
				_, _ = w.WriteString(": ping\n\n")
				_ = w.Flush()
			}
		}
	})
	return nil
}
