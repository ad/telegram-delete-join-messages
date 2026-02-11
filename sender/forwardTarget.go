package sender

const maxForwardTargetsPerAdmin = 500

func (s *Sender) storeForwardTarget(adminID, forwardedMessageID, targetUserID int64) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.forwardTargets[adminID]; !ok {
		s.forwardTargets[adminID] = make(map[int64]int64)
	}

	if len(s.forwardTargets[adminID]) >= maxForwardTargetsPerAdmin {
		for messageID := range s.forwardTargets[adminID] {
			delete(s.forwardTargets[adminID], messageID)
			break
		}
	}

	s.forwardTargets[adminID][forwardedMessageID] = targetUserID
}

func (s *Sender) getForwardTarget(adminID, forwardedMessageID int64) (int64, bool) {
	s.RLock()
	defer s.RUnlock()

	adminTargets, ok := s.forwardTargets[adminID]
	if !ok {
		return 0, false
	}

	targetUserID, ok := adminTargets[forwardedMessageID]
	if !ok {
		return 0, false
	}

	return targetUserID, true
}
