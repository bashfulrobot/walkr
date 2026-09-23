// Package secrets holds credential values in a type that cannot leak through
// formatting or logging, and resolves them from a secret store.
package secrets

import (
	"context"
	"fmt"
	"log/slog"
)

const redacted = "[redacted]"

// Secret is a credential value. Every formatting path prints "[redacted]";
// the only way to read the value is Reveal, which callers use to set a
// request header and nothing else.
type Secret struct{ v string }

// New wraps a raw value.
func New(v string) Secret { return Secret{v: v} }

// Reveal returns the raw value.
func (s Secret) Reveal() string { return s.v }

// Empty reports whether the secret holds no value.
func (s Secret) Empty() bool { return s.v == "" }

func (Secret) String() string               { return redacted }
func (Secret) GoString() string             { return redacted }
func (Secret) Format(f fmt.State, _ rune)   { _, _ = f.Write([]byte(redacted)) }
func (Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }
func (Secret) LogValue() slog.Value         { return slog.StringValue(redacted) }

// Resolver turns a secret reference (for example an op:// URI) into a Secret.
type Resolver interface {
	Resolve(ctx context.Context, ref string) (Secret, error)
}
