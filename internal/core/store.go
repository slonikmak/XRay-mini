package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Profile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}
type storedProfile struct {
	Profile
	Config json.RawMessage `json:"config"`
}
type profileStore struct {
	dir       string
	mu        sync.Mutex
	release   func() error
	closeOnce sync.Once
}

func newStore(dir string) (*profileStore, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.New("cannot resolve profile storage")
	}
	dir = absolute
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, errors.New("cannot create profile storage")
	}
	if err := secureDirectory(dir); err != nil {
		return nil, errors.New("cannot protect profile directory")
	}
	release, err := acquireStoreLock(filepath.Join(dir, ".instance.lock"))
	if err != nil {
		return nil, errors.New("another Xray Mini instance owns this profile directory")
	}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "runtime-") {
				if err = os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
					break
				}
			}
		}
	}
	if err != nil {
		_ = release()
		return nil, errors.New("cannot clean previous runtime configuration")
	}
	return &profileStore{dir: dir, release: release}, nil
}
func validID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 16 && id == strings.ToLower(id)
}
func (s *profileStore) read(id string) (storedProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readUnlocked(id)
}
func (s *profileStore) readUnlocked(id string) (storedProfile, error) {
	var p storedProfile
	if !validID(id) {
		return p, errors.New("invalid profile ID")
	}
	raw, e := os.ReadFile(filepath.Join(s.dir, id+".profile"))
	if e != nil {
		return p, errors.New("cannot read profile")
	}
	raw, e = unprotect(raw)
	if e != nil {
		return p, errors.New("cannot decrypt profile for this user")
	}
	if json.Unmarshal(raw, &p) != nil || p.ID != id {
		return p, errors.New("invalid stored profile")
	}
	return p, nil
}
func (s *profileStore) list() ([]Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, e := os.ReadDir(s.dir)
	if e != nil {
		return nil, errors.New("cannot list profiles")
	}
	result := []Profile{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".profile") {
			continue
		}
		p, e := s.readUnlocked(strings.TrimSuffix(entry.Name(), ".profile"))
		if e != nil {
			return nil, e
		}
		result = append(result, p.Profile)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result, nil
}
func (s *profileStore) add(name string, input []byte) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return Profile{}, errors.New("profile name must be 1–100 bytes")
	}
	cfg, e := PrepareConfig(input)
	if e != nil {
		return Profile{}, e
	}
	id := make([]byte, 16)
	if _, e = rand.Read(id); e != nil {
		return Profile{}, errors.New("cannot create profile ID")
	}
	p := storedProfile{Profile: Profile{hex.EncodeToString(id), name, time.Now().UTC().Format(time.RFC3339Nano)}, Config: cfg}
	raw, e := json.Marshal(p)
	if e != nil {
		return Profile{}, errors.New("cannot encode profile")
	}
	raw, e = protect(raw)
	if e != nil {
		return Profile{}, errors.New("cannot encrypt profile")
	}
	f, e := os.OpenFile(filepath.Join(s.dir, p.ID+".profile"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return Profile{}, errors.New("cannot save profile")
	}
	_, e = f.Write(raw)
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		_ = os.Remove(f.Name())
		return Profile{}, errors.New("cannot save profile")
	}
	return p.Profile, nil
}
func (s *profileStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validID(id) {
		return errors.New("invalid profile ID")
	}
	if os.Remove(filepath.Join(s.dir, id+".profile")) != nil {
		return errors.New("cannot delete profile")
	}
	return nil
}

// close releases the OS-held instance lock only after session cleanup has completed.
func (s *profileStore) close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.release != nil {
			err = s.release()
		}
	})
	return err
}
func (s *profileStore) rename(id, name string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return Profile{}, errors.New("profile name must be 1–100 bytes")
	}
	p, err := s.readUnlocked(id)
	if err != nil {
		return Profile{}, err
	}
	p.Name = name
	raw, err := json.Marshal(p)
	if err == nil {
		raw, err = protect(raw)
	}
	if err != nil {
		return Profile{}, errors.New("cannot encrypt renamed profile")
	}
	f, err := os.CreateTemp(s.dir, ".profile-update-")
	if err != nil {
		return Profile{}, errors.New("cannot save renamed profile")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return Profile{}, errors.New("cannot save renamed profile")
	}
	if os.Rename(f.Name(), filepath.Join(s.dir, id+".profile")) != nil {
		return Profile{}, errors.New("cannot replace renamed profile")
	}
	return p.Profile, nil
}
