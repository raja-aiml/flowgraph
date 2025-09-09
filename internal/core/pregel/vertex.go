package pregel

// VertexProgram defines the interface for vertex-centric computation
// PRINCIPLES:
// - User-defined logic for each vertex
// - Processes incoming messages and updates state
// - Returns halt signal to indicate vertex completion

type VertexProgram interface {
	Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error)
}

// Common vertex program implementations

// SimpleCounterVertex demonstrates basic vertex computation
type SimpleCounterVertex struct {
	MaxCount int
}

func (v *SimpleCounterVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	count := 0
	if c, ok := state["count"]; ok {
		if ci, ok := c.(int); ok {
			count = ci
		}
	}

	count++
	newState := map[string]interface{}{
		"count":         count,
		"last_messages": len(messages),
	}

	// Copy other state fields
	for k, v := range state {
		if k != "count" && k != "last_messages" {
			newState[k] = v
		}
	}

	var outMessages []*Message
	halt := count >= v.MaxCount

	// Send count to neighboring vertices (if any specified in state)
	if neighbors, ok := state["neighbors"]; ok && !halt {
		if neighborsList, ok := neighbors.([]string); ok {
			for _, neighbor := range neighborsList {
				outMessages = append(outMessages, &Message{
					From:  vertexID,
					To:    neighbor,
					Value: count,
				})
			}
		}
	}

	return newState, outMessages, halt, nil
}

// PageRankVertex implements PageRank algorithm vertex computation
type PageRankVertex struct {
	DampingFactor float64
	Convergence   float64
}

// initializeDefaults sets default values if not specified
func (v *PageRankVertex) initializeDefaults() {
	if v.DampingFactor == 0 {
		v.DampingFactor = 0.85
	}
	if v.Convergence == 0 {
		v.Convergence = 0.001
	}
}

// getRankFromState extracts current rank from vertex state
func (v *PageRankVertex) getRankFromState(state map[string]interface{}) float64 {
	if r, ok := state["rank"]; ok {
		if rf, ok := r.(float64); ok {
			return rf
		}
	}
	return 1.0
}

// sumIncomingRank calculates total incoming PageRank contributions
func (v *PageRankVertex) sumIncomingRank(messages []*Message) float64 {
	incomingRank := 0.0
	for _, msg := range messages {
		if rank, ok := msg.Value.(float64); ok {
			incomingRank += rank
		}
	}
	return incomingRank
}

// createRankMessages creates outgoing rank messages for neighbors
func (v *PageRankVertex) createRankMessages(vertexID string, rank float64, state map[string]interface{}) []*Message {
	var outMessages []*Message
	if edges, ok := state["edges"]; ok {
		if edgesList, ok := edges.([]string); ok {
			rankContribution := rank / float64(len(edgesList))
			for _, target := range edgesList {
				outMessages = append(outMessages, &Message{
					From:  vertexID,
					To:    target,
					Value: rankContribution,
				})
			}
		}
	}
	return outMessages
}

func (v *PageRankVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	v.initializeDefaults()
	oldRank := v.getRankFromState(state)
	incomingRank := v.sumIncomingRank(messages)

	// Calculate new PageRank
	newRank := (1.0 - v.DampingFactor) + v.DampingFactor*incomingRank

	// Check convergence
	delta := newRank - oldRank
	if delta < 0 {
		delta = -delta
	}
	halt := delta < v.Convergence

	newState := map[string]interface{}{
		"rank":      newRank,
		"delta":     delta,
		"iteration": state["iteration"].(int) + 1,
	}

	var outMessages []*Message
	if !halt {
		outMessages = v.createRankMessages(vertexID, newRank, state)
	}

	return newState, outMessages, halt, nil
}

// ShortestPathVertex implements single-source shortest path computation
type ShortestPathVertex struct {
	SourceVertex string
}

// getCurrentDistance extracts current distance from state
func (v *ShortestPathVertex) getCurrentDistance(state map[string]interface{}) float64 {
	const infinity = 1000000
	if d, ok := state["distance"]; ok {
		if df, ok := d.(float64); ok {
			return df
		}
	}
	return infinity
}

// findMinDistance calculates minimum distance from incoming messages
func (v *ShortestPathVertex) findMinDistance(currentDistance float64, messages []*Message) float64 {
	minDistance := currentDistance
	for _, msg := range messages {
		if dist, ok := msg.Value.(float64); ok {
			if dist < minDistance {
				minDistance = dist
			}
		}
	}
	return minDistance
}

// createPropagationMessages creates messages to propagate to neighbors
func (v *ShortestPathVertex) createPropagationMessages(vertexID string, distance float64, state map[string]interface{}) []*Message {
	var outMessages []*Message
	if edges, ok := state["edges"]; ok {
		if edgesMap, ok := edges.(map[string]float64); ok {
			for neighbor, weight := range edgesMap {
				outMessages = append(outMessages, &Message{
					From:  vertexID,
					To:    neighbor,
					Value: distance + weight,
				})
			}
		}
	}
	return outMessages
}

func (v *ShortestPathVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	currentDistance := v.getCurrentDistance(state)

	// Initialize source vertex
	isSource := vertexID == v.SourceVertex
	if isSource && currentDistance == 1000000 {
		currentDistance = 0
	}

	minDistance := v.findMinDistance(currentDistance, messages)
	changed := minDistance < currentDistance || (isSource && currentDistance == 0 && len(messages) == 0)

	var outMessages []*Message
	if changed {
		outMessages = v.createPropagationMessages(vertexID, minDistance, state)
	}

	newState := map[string]interface{}{
		"distance": minDistance,
		"edges":    state["edges"],
		"changed":  changed,
	}

	// Halt if no change occurred (except for initial source vertex)
	halt := !changed
	if isSource && currentDistance == 0 && len(messages) == 0 {
		halt = false
	}

	return newState, outMessages, halt, nil
}
