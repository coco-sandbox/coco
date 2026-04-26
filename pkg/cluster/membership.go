package cluster

import (
	"fmt"
	"sync"
	"time"
)

type Membership struct {
	mu           sync.RWMutex
	members      map[string]*Member
	leaderID     string
	electionID   string
	lastElection time.Time
}

type Member struct {
	ID        string
	Name      string
	Addr      string
	Port      int
	IsLeader  bool
	IsHealthy bool
	JoinTime  time.Time
	Status    MemberStatus
}

type MemberStatus string

const (
	StatusJoined  MemberStatus = "joined"
	StatusLeft    MemberStatus = "left"
	StatusFailed  MemberStatus = "failed"
	StatusUnknown MemberStatus = "unknown"
)

func NewMembership() *Membership {
	return &Membership{
		members: make(map[string]*Member),
	}
}

func (m *Membership) AddMember(member *Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.members[member.ID]; exists {
		return fmt.Errorf("member %s already exists", member.ID)
	}

	m.members[member.ID] = member
	return nil
}

func (m *Membership) RemoveMember(memberID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.members[memberID]; !exists {
		return fmt.Errorf("member %s not found", memberID)
	}

	delete(m.members, memberID)
	return nil
}

func (m *Membership) GetMember(memberID string) (*Member, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	member, ok := m.members[memberID]
	return member, ok
}

func (m *Membership) ListMembers() []*Member {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := make([]*Member, 0, len(m.members))
	for _, member := range m.members {
		members = append(members, member)
	}

	return members
}

func (m *Membership) GetHealthyMembers() []*Member {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := make([]*Member, 0)
	for _, member := range m.members {
		if member.IsHealthy {
			members = append(members, member)
		}
	}

	return members
}

func (m *Membership) SetLeader(memberID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, member := range m.members {
		member.IsLeader = (id == memberID)
	}

	m.leaderID = memberID
	m.lastElection = time.Now()

	return nil
}

func (m *Membership) GetLeader() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.leaderID, m.leaderID != ""
}

func (m *Membership) UpdateMemberHealth(memberID string, healthy bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	member, ok := m.members[memberID]
	if !ok {
		return fmt.Errorf("member not found")
	}

	member.IsHealthy = healthy

	if !healthy {
		member.Status = StatusFailed
	}

	return nil
}

func (m *Membership) MemberCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.members)
}

func (m *Membership) HealthyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, member := range m.members {
		if member.IsHealthy {
			count++
		}
	}

	return count
}
