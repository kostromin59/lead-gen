package tdclient

import "context"

type AuthProvider interface {
	GetCode(ctx context.Context, id string) <-chan string
	GetPassword(ctx context.Context, id string) <-chan string
}

type EventAuthProvider struct {
	onGetCode     func(ctx context.Context, id string, ch chan<- string)
	onGetPassword func(ctx context.Context, id string, ch chan<- string)
}

func NewEventAuthProvider(onGetCode func(ctx context.Context, id string, ch chan<- string), onGetPassword func(ctx context.Context, id string, ch chan<- string)) *EventAuthProvider {
	return &EventAuthProvider{
		onGetCode:     onGetCode,
		onGetPassword: onGetPassword,
	}
}

func (eap *EventAuthProvider) GetCode(ctx context.Context, id string) <-chan string {
	ch := make(chan string)

	go func() {
		eap.onGetCode(ctx, id, ch)
		close(ch)
	}()

	return ch
}

func (eap *EventAuthProvider) GetPassword(ctx context.Context, id string) <-chan string {
	ch := make(chan string)

	go func() {
		eap.onGetPassword(ctx, id, ch)
		close(ch)
	}()

	return ch
}
