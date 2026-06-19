package gtm

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/api/googleapi"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrConflict       = errors.New("resource conflict - fingerprint mismatch")
	ErrRateLimit      = errors.New("rate limit exceeded")
	ErrPermission     = errors.New("insufficient permissions")
	ErrInvalidRequest = errors.New("invalid request")
)

func retryWithBackoff[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) {
			if apiErr.Code == 403 || apiErr.Code == 429 {
				if attempt < maxRetries {
					baseWait := time.Duration(1<<uint(attempt)) * time.Second
					if baseWait > 32*time.Second {
						baseWait = 32 * time.Second
					}

					var jitterMs int64
					if baseWait > 0 {
						jitterMs = rand.Int63n(int64(baseWait) / 4)
					}
					waitTime := baseWait + time.Duration(jitterMs)

					select {
					case <-time.After(waitTime):
						lastErr = err
						continue
					case <-ctx.Done():
						return zero, ctx.Err()
					}
				}
			}
		}

		return zero, err
	}

	return zero, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func formatAPIErrorDetail(apiErr *googleapi.Error) string {
	detail := apiErr.Message
	if len(apiErr.Errors) > 0 {
		for _, e := range apiErr.Errors {
			detail += fmt.Sprintf("\n  reason=%s: %s", e.Reason, e.Message)
		}
	}
	if apiErr.Body != "" {
		detail += fmt.Sprintf("\n  body: %s", apiErr.Body)
	}
	return detail
}

func mapGoogleError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		detail := formatAPIErrorDetail(apiErr)
		switch apiErr.Code {
		case 404:
			return fmt.Errorf("%w: %s", ErrNotFound, detail)
		case 409:
			return fmt.Errorf("%w: %s", ErrConflict, detail)
		case 403:
			return fmt.Errorf("%w: %s", ErrPermission, detail)
		case 429:
			return fmt.Errorf("%w: %s", ErrRateLimit, detail)
		case 400:
			return fmt.Errorf("%w: %s", ErrInvalidRequest, detail)
		default:
			return fmt.Errorf("API error %d: %s", apiErr.Code, detail)
		}
	}

	return err
}
