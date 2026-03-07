package transaction

import "fmt"

// Action represents a reversible operation — the Command pattern applied to
// data mutations. Mirrors database/sql.Tx: you batch operations and commit
// atomically, or roll back on failure.
type Action interface {
	Apply() error
	Rollback() error
	String() string
}

// ---------------------------------------------------------------------------
// Receiver: Store
// ---------------------------------------------------------------------------

// Store is a simple in-memory key-value store (the receiver).
// Direct mutations via Put/Remove bypass transactions; transactional
// mutations go through Action objects — exactly like database/sql allows
// both db.Exec and tx.Exec.
type Store struct {
	data map[string]string
}

func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

// Put writes a key-value pair directly (outside any transaction).
func (s *Store) Put(key, val string) { s.data[key] = val }

// Remove deletes a key directly (outside any transaction).
func (s *Store) Remove(key string) { delete(s.data, key) }

// Get retrieves a value by key.
func (s *Store) Get(key string) (string, bool) {
	v, ok := s.data[key]
	return v, ok
}

// Len returns the number of keys.
func (s *Store) Len() int { return len(s.data) }

// Keys returns all keys in unspecified order.
func (s *Store) Keys() []string {
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// ---------------------------------------------------------------------------
// Concrete actions
// ---------------------------------------------------------------------------

// Set creates an action that sets a key to a value.
// Rollback restores the previous value (or removes the key if it didn't exist).
func Set(s *Store, key, val string) Action {
	return &setAction{store: s, key: key, val: val}
}

type setAction struct {
	store    *Store
	key, val string
	prev     string
	hadPrev  bool
}

func (a *setAction) Apply() error {
	a.prev, a.hadPrev = a.store.Get(a.key)
	a.store.Put(a.key, a.val)
	return nil
}

func (a *setAction) Rollback() error {
	if a.hadPrev {
		a.store.Put(a.key, a.prev)
	} else {
		a.store.Remove(a.key)
	}
	return nil
}

func (a *setAction) String() string {
	return fmt.Sprintf("Set(%q, %q)", a.key, a.val)
}

// Del creates an action that deletes a key.
// Fails if the key does not exist. Rollback re-inserts the deleted value.
func Del(s *Store, key string) Action {
	return &delAction{store: s, key: key}
}

type delAction struct {
	store   *Store
	key     string
	prev    string
	hadPrev bool
}

func (a *delAction) Apply() error {
	a.prev, a.hadPrev = a.store.Get(a.key)
	if !a.hadPrev {
		return fmt.Errorf("key %q does not exist", a.key)
	}
	a.store.Remove(a.key)
	return nil
}

func (a *delAction) Rollback() error {
	if a.hadPrev {
		a.store.Put(a.key, a.prev)
	}
	return nil
}

func (a *delAction) String() string {
	return fmt.Sprintf("Del(%q)", a.key)
}

// ---------------------------------------------------------------------------
// Transaction (invoker for batch + rollback)
// ---------------------------------------------------------------------------

// Transaction collects actions and applies them atomically.
// If any action fails during Commit, all previously applied actions are
// rolled back in reverse order — matching database/sql.Tx semantics.
type Transaction struct {
	actions   []Action
	applied   []Action
	committed bool
}

// Begin creates a transaction with optional initial actions.
func Begin(actions ...Action) *Transaction {
	return &Transaction{actions: actions}
}

// Add appends an action to the pending list. Panics if already committed.
func (tx *Transaction) Add(a Action) *Transaction {
	if tx.committed {
		panic("cannot add action to committed transaction")
	}
	tx.actions = append(tx.actions, a)
	return tx
}

// Commit applies all actions in order. On the first failure, it rolls back
// every previously applied action (reverse order) and returns the error.
func (tx *Transaction) Commit() error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	for i, a := range tx.actions {
		if err := a.Apply(); err != nil {
			for j := len(tx.applied) - 1; j >= 0; j-- {
				tx.applied[j].Rollback()
			}
			tx.applied = nil
			return fmt.Errorf("action %d (%s) failed, rolled back: %w", i, a, err)
		}
		tx.applied = append(tx.applied, a)
	}
	tx.committed = true
	return nil
}

// Rollback reverses all applied actions in reverse order.
// Useful for explicit rollback of a successfully committed transaction.
func (tx *Transaction) Rollback() error {
	for i := len(tx.applied) - 1; i >= 0; i-- {
		if err := tx.applied[i].Rollback(); err != nil {
			return fmt.Errorf("rollback failed at action %d: %w", i, err)
		}
	}
	tx.applied = nil
	tx.committed = false
	return nil
}

// Len returns the number of pending actions.
func (tx *Transaction) Len() int { return len(tx.actions) }
