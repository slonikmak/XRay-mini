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
type profileStore struct{ dir string }

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
	return &profileStore{dir}, nil
}
func validID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 16 && id == strings.ToLower(id)
}
func (s *profileStore) read(id string) (storedProfile, error) {
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
	entries, e := os.ReadDir(s.dir)
	if e != nil {
		return nil, errors.New("cannot list profiles")
	}
	result := []Profile{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".profile") {
			continue
		}
		p, e := s.read(strings.TrimSuffix(entry.Name(), ".profile"))
		if e != nil {
			return nil, e
		}
		result = append(result, p.Profile)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result, nil
}
func (s *profileStore) add(name string, input []byte) (Profile, error) {
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
	if !validID(id) {
		return errors.New("invalid profile ID")
	}
	if os.Remove(filepath.Join(s.dir, id+".profile")) != nil {
		return errors.New("cannot delete profile")
	}
	return nil
}
