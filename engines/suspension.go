package engines

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// EngineBlockedError signals that an engine returned a response that
// looks like a bot block (captcha, rate limit, access denied, consent
// wall). FanOut treats this as a signal to suspend the engine for a
// cooldown window so successive queries don't keep hammering it.
type EngineBlockedError struct {
	Engine string
	Reason BlockReason
	Inner  error
}

func (e *EngineBlockedError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("%s: blocked (%s): %s", e.Engine, e.Reason, e.Inner.Error())
	}
	return fmt.Sprintf("%s: blocked (%s)", e.Engine, e.Reason)
}

func (e *EngineBlockedError) Unwrap() error { return e.Inner }

// BlockReason is a small enum of the failure modes we suspend on.
type BlockReason string

const (
	BlockReasonCaptcha      BlockReason = "captcha"
	BlockReasonRateLimit    BlockReason = "rate_limit"
	BlockReasonAccessDenied BlockReason = "access_denied"
	BlockReasonConsentWall  BlockReason = "consent_wall"
)

// suspensionBackoff returns the cooldown for the Nth consecutive block.
// Caps at 1800s after the third strike.
func suspensionBackoff(consecutive int) time.Duration {
	switch {
	case consecutive <= 1:
		return 60 * time.Second
	case consecutive == 2:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}

type suspensionEntry struct {
	expires     time.Time
	consecutive int
}

var (
	suspensionMu    sync.Mutex
	suspensionTable = map[string]*suspensionEntry{}
	// nowFn is overridable in tests so we can drive the cooldown clock
	// without sleeping.
	nowFn = time.Now
)

// IsSuspended reports whether the engine is currently in cooldown.
func IsSuspended(engine string) bool {
	suspensionMu.Lock()
	defer suspensionMu.Unlock()
	e, ok := suspensionTable[engine]
	if !ok {
		return false
	}
	if nowFn().Before(e.expires) {
		return true
	}
	// expired - drop the entry so the counter resets after a long quiet
	delete(suspensionTable, engine)
	return false
}

// Suspend records a block and bumps the consecutive counter. Cooldown
// length follows suspensionBackoff. Returns the duration applied.
func Suspend(engine string) time.Duration {
	suspensionMu.Lock()
	defer suspensionMu.Unlock()
	e, ok := suspensionTable[engine]
	if !ok || nowFn().After(e.expires) {
		e = &suspensionEntry{}
		suspensionTable[engine] = e
	}
	e.consecutive++
	d := suspensionBackoff(e.consecutive)
	e.expires = nowFn().Add(d)
	return d
}

// ClearSuspension drops the engine's record entirely. Called after a
// successful response so the consecutive counter doesn't carry over.
func ClearSuspension(engine string) {
	suspensionMu.Lock()
	defer suspensionMu.Unlock()
	delete(suspensionTable, engine)
}

// shouldSuspend reports whether the error from an engine call looks
// block-shaped enough to put the engine in cooldown. True for any
// *EngineBlockedError, plus HTTP 403/429 / 503 from fetch().
func shouldSuspend(err error) bool {
	if err == nil {
		return false
	}
	var be *EngineBlockedError
	if errors.As(err, &be) {
		return true
	}
	var he *httpError
	if errors.As(err, &he) {
		switch he.Status {
		case 403, 429, 503:
			return true
		}
	}
	return false
}
