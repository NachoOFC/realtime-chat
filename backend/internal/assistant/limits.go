package assistant

import (
	"sync"
	"time"
)

const (
	userMinuteLimit   = 6
	userDayLimit      = 60
	globalMinuteLimit = 30
	globalDayLimit    = 500
	globalInFlight    = 4
	maxTrackedUsers   = 4096
)

type usage struct {
	minuteStart time.Time
	dayStart    time.Time
	lastSeen    time.Time
	minute      int
	day         int
	inFlight    int
}

func (u *usage) roll(now time.Time) {
	if now.Sub(u.minuteStart) >= time.Minute {
		u.minuteStart, u.minute = now, 0
	}
	if now.Sub(u.dayStart) >= 24*time.Hour {
		u.dayStart, u.day = now, 0
	}
}

type limiter struct {
	mu          sync.Mutex
	users       map[string]*usage
	global      usage
	lastCleanup time.Time
	now         func() time.Time
}

func newLimiter() *limiter {
	return &limiter{users: make(map[string]*usage), now: time.Now}
}

func (l *limiter) acquire(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastCleanup) >= time.Minute || len(l.users) >= maxTrackedUsers {
		for id, current := range l.users {
			if current.inFlight == 0 && now.Sub(current.lastSeen) >= 24*time.Hour {
				delete(l.users, id)
			}
		}
		l.lastCleanup = now
	}
	l.global.roll(now)
	current := l.users[userID]
	if current == nil {
		if len(l.users) >= maxTrackedUsers {
			return false
		}
		current = &usage{}
		l.users[userID] = current
	}
	current.roll(now)
	current.lastSeen = now
	if current.inFlight != 0 || l.global.inFlight >= globalInFlight || current.minute >= userMinuteLimit || current.day >= userDayLimit || l.global.minute >= globalMinuteLimit || l.global.day >= globalDayLimit {
		return false
	}
	current.inFlight++
	current.minute++
	current.day++
	l.global.inFlight++
	l.global.minute++
	l.global.day++
	return true
}

func (l *limiter) release(userID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if current := l.users[userID]; current != nil && current.inFlight > 0 {
		current.inFlight--
		l.global.inFlight--
	}
}

func (l *limiter) retryAfter(userID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	wait := time.Second
	check := func(current *usage, minuteLimit, dayLimit int) {
		if current == nil {
			return
		}
		if current.minute >= minuteLimit {
			wait = max(wait, current.minuteStart.Add(time.Minute).Sub(now))
		}
		if current.day >= dayLimit {
			wait = max(wait, current.dayStart.Add(24*time.Hour).Sub(now))
		}
	}
	check(&l.global, globalMinuteLimit, globalDayLimit)
	check(l.users[userID], userMinuteLimit, userDayLimit)
	if len(l.users) >= maxTrackedUsers {
		wait = max(wait, time.Minute)
	}
	return int((wait + time.Second - 1) / time.Second)
}
