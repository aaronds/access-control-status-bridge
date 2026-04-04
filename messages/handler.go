package messages

import "time"

type RawMessage struct {
    Payload []byte
    Received time.Time
    Version string
}

type AccessMessageHandler interface {
    In() chan RawMessage
    Poll()
}
