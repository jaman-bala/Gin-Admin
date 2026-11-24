package auth

import (
	"context"
	"errors"
	"testing"

	domainUser "gin_auth_service/internal/domain/user"
	appErrors "gin_auth_service/pkg/errors"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	// dummyHash is compared on every unknown-phone login; at production cost
	// that makes the throttling tests needlessly slow.
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-timing-equalizer"), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	dummyHash = string(h)
}

func tryLogin(e *env, phone, password string) error {
	_, err := e.uc.Login(context.Background(), LoginRequestDTO{Phone: phone, Password: password})
	return err
}

func TestLogin_AccountLockedAfterTooManyFailures(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)

	for i := int64(0); i < accountMaxFailures; i++ {
		if err := tryLogin(e, u.Phone, "wrong"); !errors.Is(err, appErrors.ErrInvalidCredentials) {
			t.Fatalf("guess %d: err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}

	// Budget spent: even the CORRECT password is refused, and further guesses
	// get the throttling answer rather than another credential check.
	if err := tryLogin(e, u.Phone, testPassword); !errors.Is(err, appErrors.ErrTooManyAttempts) {
		t.Fatalf("correct password while locked: err = %v, want ErrTooManyAttempts", err)
	}
	if err := tryLogin(e, u.Phone, "wrong"); !errors.Is(err, appErrors.ErrTooManyAttempts) {
		t.Fatalf("guess while locked: err = %v, want ErrTooManyAttempts", err)
	}
}

// The original bypass: a successful login must not erase the failure count.
// Here the attacker owns a valid account of their own and interleaves it with
// guesses against the victim, spread over the same budget key space they can reach.
func TestLogin_SuccessDoesNotRefundFailures(t *testing.T) {
	victim := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, victim)

	guesses := 0
	for round := 0; round < 100; round++ {
		err := tryLogin(e, victim.Phone, "wrong")
		if errors.Is(err, appErrors.ErrTooManyAttempts) {
			break
		}
		guesses++
		// the victim's own legitimate success in between must not help the attacker either
		if err := tryLogin(e, victim.Phone, testPassword); err == nil {
			continue
		}
	}
	if guesses > int(accountMaxFailures) {
		t.Fatalf("%d wrong guesses were allowed, limit is %d: successes are resetting the counter", guesses, accountMaxFailures)
	}
}

func TestLogin_SuccessesAloneNeverLockAnAccount(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)

	for i := 0; i < int(accountMaxFailures)*3; i++ {
		if err := tryLogin(e, u.Phone, testPassword); err != nil {
			t.Fatalf("successful login %d failed: %v", i+1, err)
		}
	}
}

func TestLogin_LimitIsPerAccount(t *testing.T) {
	a := newUser(t, "+996500500500", domainUser.RoleUser, true)
	b := newUser(t, "+996500500501", domainUser.RoleUser, true)
	e := newEnv(t, nil, a, b)

	for i := int64(0); i <= accountMaxFailures; i++ {
		_ = tryLogin(e, a.Phone, "wrong")
	}
	if err := tryLogin(e, b.Phone, testPassword); err != nil {
		t.Fatalf("another account must be unaffected: %v", err)
	}
}

// A 429 must not reveal whether an account exists: unknown numbers are
// throttled exactly like real ones.
func TestLogin_UnknownPhoneIsThrottledToo(t *testing.T) {
	e := newEnv(t, nil)
	const ghost = "+996700000000"

	for i := int64(0); i < accountMaxFailures; i++ {
		if err := tryLogin(e, ghost, "x"); !errors.Is(err, appErrors.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
	if err := tryLogin(e, ghost, "x"); !errors.Is(err, appErrors.ErrTooManyAttempts) {
		t.Fatalf("err = %v, want ErrTooManyAttempts (same behaviour as for a real account)", err)
	}
}

// "+996..." and "996..." are the same account: they must share one budget,
// otherwise a formatting change doubles the attacker's guesses.
func TestLogin_PhoneFormatsShareOneBudget(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)

	for i := int64(0); i < accountMaxFailures; i++ {
		phone := "+996500500500"
		if i%2 == 1 {
			phone = "996 500-500 500"
		}
		_ = tryLogin(e, phone, "wrong")
	}
	if err := tryLogin(e, "996500500500", "wrong"); !errors.Is(err, appErrors.ErrTooManyAttempts) {
		t.Fatalf("err = %v, want ErrTooManyAttempts: formats must not get separate budgets", err)
	}
}

func TestLogin_LimiterOutageFailsOpen(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	e.limiter.incErr = errors.New("redis down")

	if err := tryLogin(e, u.Phone, testPassword); err != nil {
		t.Fatalf("a Redis outage must not lock everyone out: %v", err)
	}
}

func TestLogin_ReleaseFailureDoesNotFailAValidLogin(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	e.limiter.relErr = errors.New("redis down")

	if err := tryLogin(e, u.Phone, testPassword); err != nil {
		t.Fatalf("login failed because the slot could not be released: %v", err)
	}
}
