package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
)

// OnePassword resolves op:// references through the 1Password SDK using a
// service account. The service account token is the only credential that has
// to exist outside 1Password.
type OnePassword struct {
	api onepassword.SecretsAPI
}

// NewOnePassword authenticates with the given service account token.
func NewOnePassword(ctx context.Context, serviceAccountToken Secret, version string) (*OnePassword, error) {
	if serviceAccountToken.Empty() {
		return nil, errors.New("1Password service account token is empty")
	}
	client, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(serviceAccountToken.Reveal()),
		onepassword.WithIntegrationInfo("walkr", version),
	)
	if err != nil {
		return nil, fmt.Errorf("1Password client: %w", err)
	}
	return &OnePassword{api: client.Secrets()}, nil
}

// Resolve returns the secret the reference points to. The reference itself is
// a pointer, not a secret, so it is safe in error messages.
func (o *OnePassword) Resolve(ctx context.Context, ref string) (Secret, error) {
	v, err := o.api.Resolve(ctx, ref)
	if err != nil {
		return Secret{}, fmt.Errorf("resolve %s: %w", ref, err)
	}
	return New(v), nil
}
