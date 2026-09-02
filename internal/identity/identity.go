// Package identity deduces the local pilot's callsign without user configuration.
//
// War Thunder's local API exposes no "this message is yours" flag. However,
// /indicators reports the vehicle the local player is currently flying, and
// /hudmsg damage lines are formatted as "<clan> <player> (<Vehicle>) <event>".
// Correlating the local vehicle against actors seen in the damage feed narrows
// the candidates to the local player, because only the local player is
// guaranteed to be flying the vehicle /indicators reports.
package identity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// actorPattern captures the actor and vehicle from a damage line such as
// "=GRIND= DEERSLUG (JH-7) destroyed Studebaker US6".
var actorPattern = regexp.MustCompile(`^(.*?)\s*\(([^()]+)\)`)

// minConfirmations is how many independent damage lines must agree on a single
// candidate before it is accepted. A teammate can briefly appear to match when
// vehicle names normalize alike, so require corroboration.
const minConfirmations = 2

// Resolver deduces and remembers the local pilot's callsign.
type Resolver struct {
	mu         sync.RWMutex
	callsign   string
	confirmed  bool
	candidates map[string]int
	storePath  string
}

type persisted struct {
	Callsign string `json:"callsign"`
}

// NewResolver creates a resolver that persists the learned callsign to
// storePath. An empty storePath disables persistence.
func NewResolver(storePath string) *Resolver {
	resolver := &Resolver{
		candidates: make(map[string]int),
		storePath:  storePath,
	}
	resolver.load()
	return resolver
}

// Callsign returns the current best-known callsign and whether it has been
// confirmed.
func (r *Resolver) Callsign() (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callsign, r.confirmed
}

// SetCallsign records an explicit override, which always wins over deduction.
func (r *Resolver) SetCallsign(callsign string) {
	callsign = strings.TrimSpace(callsign)
	if callsign == "" {
		return
	}
	r.mu.Lock()
	r.callsign = callsign
	r.confirmed = true
	r.mu.Unlock()
	r.save(callsign)
}

// Clear removes any explicit or deduced identity, including the persisted file.
func (r *Resolver) Clear() error {
	r.mu.Lock()
	r.callsign = ""
	r.confirmed = false
	r.candidates = make(map[string]int)
	storePath := r.storePath
	r.mu.Unlock()
	if storePath == "" {
		return nil
	}
	if err := os.Remove(storePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Observe correlates a damage-feed line against the locally flown vehicle. It
// returns true when the callsign becomes confirmed as a result of this record.
func (r *Resolver) Observe(damageMessage, localVehicle string) bool {
	if localVehicle == "" || damageMessage == "" {
		return false
	}
	actor, vehicle, ok := parseActor(damageMessage)
	if !ok || !vehicleMatches(vehicle, localVehicle) {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.confirmed {
		return false
	}
	r.candidates[actor]++
	if r.candidates[actor] < minConfirmations {
		// Keep the best guess available for display, but do not lock it in.
		r.callsign = actor
		return false
	}
	r.callsign = actor
	r.confirmed = true
	callsign := actor
	go r.save(callsign)
	return true
}

// Matches reports whether a /gamechat sender is the local pilot. Chat senders
// omit the clan tag that damage lines include, so compare on the bare name.
func (r *Resolver) Matches(sender string) bool {
	sender = strings.TrimSpace(sender)
	if sender == "" {
		return false
	}
	r.mu.RLock()
	callsign := r.callsign
	confirmed := r.confirmed
	r.mu.RUnlock()
	if callsign == "" || !confirmed {
		return false
	}
	if strings.EqualFold(callsign, sender) {
		return true
	}
	return BareName(callsign) != "" && BareName(callsign) == BareName(sender)
}

// MentionsLocalPilot reports whether a damage line refers to the local pilot.
func (r *Resolver) MentionsLocalPilot(message string) bool {
	r.mu.RLock()
	callsign := r.callsign
	confirmed := r.confirmed
	r.mu.RUnlock()
	if callsign == "" || !confirmed {
		return false
	}
	actor, _, ok := parseActor(message)
	if !ok {
		return false
	}
	return strings.EqualFold(actor, callsign) ||
		(BareName(actor) != "" && BareName(actor) == BareName(callsign))
}

// IsLocalPilotLoss reports whether a combat-feed line records the local pilot
// as the aircraft that was lost. The local pilot may be either the leading
// actor ("DEERSLUG (J-7D) has crashed") or the victim after a combat verb
// ("attacker (...) shot down DEERSLUG (J-7D)").
func (r *Resolver) IsLocalPilotLoss(message string) bool {
	r.mu.RLock()
	callsign := r.callsign
	confirmed := r.confirmed
	r.mu.RUnlock()
	if callsign == "" || !confirmed {
		return false
	}

	lowerMessage := strings.ToLower(message)
	lowerCallsign := strings.ToLower(callsign)
	index := strings.Index(lowerMessage, lowerCallsign)
	if index < 0 {
		return false
	}

	before := strings.TrimSpace(lowerMessage[:index])
	after := lowerMessage[index+len(lowerCallsign):]
	if strings.HasSuffix(before, "shot down") ||
		strings.HasSuffix(before, "destroyed") {
		return strings.Contains(after, "(")
	}

	closeVehicle := strings.Index(after, ")")
	if closeVehicle < 0 {
		return false
	}
	result := strings.TrimSpace(after[closeVehicle+1:])
	return strings.HasPrefix(result, "has crashed") ||
		strings.HasPrefix(result, "has been destroyed") ||
		strings.HasPrefix(result, "has been shot down")
}

// ResetSession clears per-match deduction progress while retaining a callsign
// that has already been confirmed or persisted.
func (r *Resolver) ResetSession() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.candidates = make(map[string]int)
	if !r.confirmed {
		r.callsign = ""
	}
}

// BareName strips clan tags and decorations, leaving the player name that
// /gamechat reports. "=GRIND= DEERSLUG" becomes "deerslug".
func BareName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	fields := strings.Fields(value)
	// The player name is the last whitespace-separated token that still has
	// alphanumeric content once decorations are removed.
	for index := len(fields) - 1; index >= 0; index-- {
		if candidate := normalizeName(fields[index]); candidate != "" {
			return candidate
		}
	}
	return normalizeName(value)
}

func normalizeName(value string) string {
	var builder strings.Builder
	for _, symbol := range value {
		switch {
		case symbol >= 'a' && symbol <= 'z',
			symbol >= '0' && symbol <= '9':
			builder.WriteRune(symbol)
		case symbol >= 'A' && symbol <= 'Z':
			builder.WriteRune(symbol + ('a' - 'A'))
		}
	}
	return builder.String()
}

func parseActor(message string) (string, string, bool) {
	match := actorPattern.FindStringSubmatch(message)
	if match == nil {
		return "", "", false
	}
	actor := strings.TrimSpace(match[1])
	vehicle := strings.TrimSpace(match[2])
	if actor == "" || vehicle == "" {
		return "", "", false
	}
	return actor, vehicle, true
}

// vehicleMatches compares a display vehicle name such as "J-7D" against the
// /indicators type such as "j_7d", ignoring separators and decoration glyphs.
func vehicleMatches(display, indicatorType string) bool {
	left := normalizeName(display)
	right := normalizeName(indicatorType)
	if left == "" || right == "" {
		return false
	}
	return left == right
}

func (r *Resolver) load() {
	if r.storePath == "" {
		return
	}
	contents, err := os.ReadFile(r.storePath)
	if err != nil {
		return
	}
	var stored persisted
	if err := json.Unmarshal(contents, &stored); err != nil {
		return
	}
	if callsign := strings.TrimSpace(stored.Callsign); callsign != "" {
		r.mu.Lock()
		r.callsign = callsign
		r.confirmed = true
		r.mu.Unlock()
	}
}

func (r *Resolver) save(callsign string) {
	if r.storePath == "" {
		return
	}
	contents, err := json.Marshal(persisted{Callsign: callsign})
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.storePath), 0o755); err != nil {
		return
	}
	temporary := r.storePath + ".tmp"
	if err := os.WriteFile(temporary, contents, 0o644); err != nil {
		return
	}
	if err := os.Rename(temporary, r.storePath); err != nil {
		os.Remove(temporary)
	}
}

// DefaultStorePath returns the per-user location for the remembered callsign.
func DefaultStorePath() string {
	directory, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(directory, "wt-modern-8111", "identity.json")
}
