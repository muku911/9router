package db

import (
	"fmt"
	"time"
)

// CheckLock evaluates if a connection is currently locked for a specific model or model family.
func (s *Store) CheckLock(connID, model, family string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.payload.ProviderConnections {
		if conn.ID == connID {
			if conn.Locks == nil {
				return nil
			}

			// Check model specific lock
			if model != "" {
				lockKey := fmt.Sprintf("modelLock_%s", model)
				if expireStr, exists := conn.Locks[lockKey]; exists {
					expireTime, err := time.Parse(time.RFC3339, expireStr)
					if err == nil && time.Now().Before(expireTime) {
						return fmt.Errorf("connection %s is locked for model %s until %s", connID, model, expireStr)
					}
				}
			}

			// Check group/family lock
			if family != "" {
				lockKey := fmt.Sprintf("modelGroupLock_%s", family)
				if expireStr, exists := conn.Locks[lockKey]; exists {
					expireTime, err := time.Parse(time.RFC3339, expireStr)
					if err == nil && time.Now().Before(expireTime) {
						return fmt.Errorf("connection %s is locked for family %s until %s", connID, family, expireStr)
					}
				}
			}
			break
		}
	}

	return nil
}

// ApplyLock sets locks (model & group/family) for a connection in the database JSON
func (s *Store) ApplyLock(connID, model, family string, duration time.Duration) {
	if duration <= 0 {
		return
	}

	s.mu.Lock()
	expireStr := time.Now().Add(duration).Format(time.RFC3339)

	for i, conn := range s.payload.ProviderConnections {
		if conn.ID == connID {
			if conn.Locks == nil {
				conn.Locks = make(map[string]string)
			}

			if model != "" {
				conn.Locks[fmt.Sprintf("modelLock_%s", model)] = expireStr
			}
			if family != "" {
				conn.Locks[fmt.Sprintf("modelGroupLock_%s", family)] = expireStr
			}

			s.payload.ProviderConnections[i] = conn
			break
		}
	}
	s.mu.Unlock()

	// Persist changes to disk asynchronously
	go func() {
		_ = s.Save()
	}()
}

// IncrementUsage updates the request count, prompt token count, and completion token count in connection metrics
func (s *Store) IncrementUsage(connID, model string, promptTokens, completionTokens int) {
	s.mu.Lock()
	for i, conn := range s.payload.ProviderConnections {
		if conn.ID == connID {
			if conn.Data == nil {
				conn.Data = make(map[string]interface{})
			}

			// 1. Read/Increment dynamic totals per model (e.g. usageByModel_gemini-2.5-pro-exp)
			// Matches the requested model-specific token usage tracking format
			if model != "" {
				modelKey := fmt.Sprintf("usageByModel_%s", model)
				var modelStats map[string]interface{}
				if existingStats, ok := conn.Data[modelKey]; ok {
					if mStats, isMap := existingStats.(map[string]interface{}); isMap {
						modelStats = mStats
					}
				}
				if modelStats == nil {
					modelStats = map[string]interface{}{
						"requests":         0.0,
						"promptTokens":     0.0,
						"completionTokens": 0.0,
					}
				}

				reqs := 0.0
				if r, ok := modelStats["requests"]; ok {
					reqs, _ = r.(float64)
				}
				pToks := 0.0
				if p, ok := modelStats["promptTokens"]; ok {
					pToks, _ = p.(float64)
				}
				cToks := 0.0
				if c, ok := modelStats["completionTokens"]; ok {
					cToks, _ = c.(float64)
				}

				modelStats["requests"] = reqs + 1
				modelStats["promptTokens"] = pToks + float64(promptTokens)
				modelStats["completionTokens"] = cToks + float64(completionTokens)

				conn.Data[modelKey] = modelStats
			}

			// 2. Global totals
			reqCount := 0.0
			if val, ok := conn.Data["requestsLifetime"]; ok {
				if fVal, isFloat := val.(float64); isFloat {
					reqCount = fVal
				}
			}
			reqCount++
			conn.Data["requestsLifetime"] = reqCount

			pTokens := 0.0
			if val, ok := conn.Data["promptTokensLifetime"]; ok {
				if fVal, isFloat := val.(float64); isFloat {
					pTokens = fVal
				}
			}
			pTokens += float64(promptTokens)
			conn.Data["promptTokensLifetime"] = pTokens

			cTokens := 0.0
			if val, ok := conn.Data["completionTokensLifetime"]; ok {
				if fVal, isFloat := val.(float64); isFloat {
					cTokens = fVal
				}
			}
			cTokens += float64(completionTokens)
			conn.Data["completionTokensLifetime"] = cTokens

			s.payload.ProviderConnections[i] = conn
			break
		}
	}
	s.mu.Unlock()

	// Persist changes to disk asynchronously
	go func() {
		_ = s.Save()
	}()
}
