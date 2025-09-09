package pregel

import "sync"

// Message represents a message sent between vertices in Pregel
// PRINCIPLES:
// - Simple struct for vertex communication
// - Supports aggregation and combining

type Message struct {
	From  string      // Source vertex ID
	To    string      // Target vertex ID
	Value interface{} // Payload
	Step  int         // Superstep number
}

// MessageCombiner defines how to combine multiple messages to the same vertex
type MessageCombiner interface {
	Combine(messages []*Message) *Message
}

// MessageAggregator aggregates messages for efficient delivery
type MessageAggregator struct {
	mu       sync.RWMutex
	messages map[string][]*Message // vertex ID -> messages
	combiner MessageCombiner
}

func NewMessageAggregator(combiner MessageCombiner) *MessageAggregator {
	return &MessageAggregator{
		messages: make(map[string][]*Message),
		combiner: combiner,
	}
}

func (ma *MessageAggregator) AddMessage(msg *Message) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.messages[msg.To] = append(ma.messages[msg.To], msg)
}

func (ma *MessageAggregator) GetMessages(vertexID string) []*Message {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	msgs := ma.messages[vertexID]
	if ma.combiner != nil && len(msgs) > 1 {
		combined := ma.combiner.Combine(msgs)
		return []*Message{combined}
	}
	return msgs
}

func (ma *MessageAggregator) Clear() {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.messages = make(map[string][]*Message)
}

// GetAllMessages returns all messages for all vertices (useful for debugging)
func (ma *MessageAggregator) GetAllMessages() map[string][]*Message {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	result := make(map[string][]*Message)
	for vid, msgs := range ma.messages {
		result[vid] = make([]*Message, len(msgs))
		copy(result[vid], msgs)
	}
	return result
}

// GetMessageCount returns the total number of messages in the aggregator
func (ma *MessageAggregator) GetMessageCount() int {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	count := 0
	for _, msgs := range ma.messages {
		count += len(msgs)
	}
	return count
}

// HasMessages checks if there are any messages for any vertex
func (ma *MessageAggregator) HasMessages() bool {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	for _, msgs := range ma.messages {
		if len(msgs) > 0 {
			return true
		}
	}
	return false
}

// RemoveMessagesForVertex removes all messages for a specific vertex
func (ma *MessageAggregator) RemoveMessagesForVertex(vertexID string) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	delete(ma.messages, vertexID)
}

// SumCombiner combines messages by summing their numeric values
type SumCombiner struct{}

func (sc *SumCombiner) Combine(messages []*Message) *Message {
	if len(messages) == 0 {
		return nil
	}

	sum := 0.0
	first := messages[0]

	for _, msg := range messages {
		if val, ok := msg.Value.(int); ok {
			sum += float64(val)
		} else if val, ok := msg.Value.(float64); ok {
			sum += val
		}
	}

	return &Message{
		From:  first.From,
		To:    first.To,
		Value: sum,
		Step:  first.Step,
	}
}

// MinCombiner finds the minimum value among messages
type MinCombiner struct{}

func (mc *MinCombiner) Combine(messages []*Message) *Message {
	if len(messages) == 0 {
		return nil
	}

    minVal := float64(1000000) // Large initial value
    var minMsg *Message

	for _, msg := range messages {
		var val float64
		if v, ok := msg.Value.(int); ok {
			val = float64(v)
		} else if v, ok := msg.Value.(float64); ok {
			val = v
		} else {
			continue
		}

        if val < minVal {
            minVal = val
            minMsg = msg
        }
    }

	if minMsg != nil {
		return &Message{
			From:  minMsg.From,
			To:    minMsg.To,
            Value: minVal,
            Step:  minMsg.Step,
        }
    }

	return messages[0] // Fallback
}

// MaxCombiner finds the maximum value among messages
type MaxCombiner struct{}

func (mc *MaxCombiner) Combine(messages []*Message) *Message {
	if len(messages) == 0 {
		return nil
	}

    maxVal := float64(-1000000) // Small initial value
    var maxMsg *Message

	for _, msg := range messages {
		var val float64
		if v, ok := msg.Value.(int); ok {
			val = float64(v)
		} else if v, ok := msg.Value.(float64); ok {
			val = v
		} else {
			continue
		}

        if val > maxVal {
            maxVal = val
            maxMsg = msg
        }
    }

	if maxMsg != nil {
		return &Message{
			From:  maxMsg.From,
			To:    maxMsg.To,
            Value: maxVal,
            Step:  maxMsg.Step,
        }
    }

	return messages[0] // Fallback
}

// ListCombiner collects all message values into a list
type ListCombiner struct{}

func (lc *ListCombiner) Combine(messages []*Message) *Message {
	if len(messages) == 0 {
		return nil
	}

	values := make([]interface{}, 0, len(messages))
	first := messages[0]

	for _, msg := range messages {
		values = append(values, msg.Value)
	}

	return &Message{
		From:  first.From,
		To:    first.To,
		Value: values,
		Step:  first.Step,
	}
}

// NoCombiner doesn't combine messages - passes all through
type NoCombiner struct{}

func (nc *NoCombiner) Combine(messages []*Message) *Message {
	// This combiner should not be used in practice with MessageAggregator
	// since GetMessages will handle this case
	if len(messages) > 0 {
		return messages[0]
	}
	return nil
}
