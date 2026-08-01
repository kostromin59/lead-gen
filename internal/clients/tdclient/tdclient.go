package tdclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/zelenin/go-tdlib/client"
)

type TDClient struct {
	id            string
	phone         string
	apiHash       string
	apiID         int32
	client        *client.Client
	isAuthorized  bool
	isAuthorizing bool
	mu            sync.RWMutex
}

func New(id, phone, apiHash string, apiID int32) *TDClient {
	return &TDClient{
		id:      id,
		phone:   phone,
		apiID:   apiID,
		apiHash: apiHash,
	}
}

func (tdc *TDClient) ID() string {
	return tdc.id
}

func (tdc *TDClient) Phone() string {
	return tdc.phone
}

func (tdc *TDClient) IsAuthorized() bool {
	tdc.mu.RLock()
	defer tdc.mu.RUnlock()

	return tdc.isAuthorized
}

func (tdc *TDClient) Client() *client.Client {
	tdc.mu.RLock()
	defer tdc.mu.RUnlock()

	return tdc.client
}

func (tdc *TDClient) Close() error {
	tdc.mu.Lock()
	defer tdc.mu.Unlock()

	if tdc.client != nil {
		tdc.isAuthorized = false
		tdc.isAuthorizing = false

		if _, err := tdc.client.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (tdc *TDClient) Authorize(ctx context.Context, authProvider AuthProvider) error {
	if tdc.IsAuthorized() {
		return nil
	}

	tdc.mu.Lock()
	if tdc.isAuthorizing {
		tdc.mu.Unlock()
		return fmt.Errorf("already authorizing")
	}
	tdc.isAuthorizing = true
	tdc.mu.Unlock()

	defer func() {
		tdc.mu.Lock()
		tdc.isAuthorizing = false
		tdc.mu.Unlock()
	}()

	authCh := make(chan error)
	authorizer := client.ClientAuthorizer(parameters(tdc.id, tdc.apiHash, tdc.apiID))

	go func() {
		defer close(authCh)
		for {
			select {
			case <-ctx.Done():
				authCh <- ctx.Err()
				return
			case state, ok := <-authorizer.State:
				if !ok {
					tdc.mu.Lock()
					tdc.isAuthorized = true
					tdc.mu.Unlock()
					return
				}

				switch state.AuthorizationStateType() {
				case client.TypeAuthorizationStateWaitPhoneNumber:
					select {
					case authorizer.PhoneNumber <- tdc.phone:
					case <-ctx.Done():
						authCh <- ctx.Err()
						return
					}

				case client.TypeAuthorizationStateWaitCode:
					select {
					case code := <-authProvider.GetCode(ctx, tdc.id):
						select {
						case authorizer.Code <- code:
						case <-ctx.Done():
							authCh <- ctx.Err()
							return
						}

					case <-ctx.Done():
						authCh <- ctx.Err()
						return
					}

				case client.TypeAuthorizationStateWaitPassword:
					select {
					case password := <-authProvider.GetPassword(ctx, tdc.id):
						select {
						case authorizer.Password <- password:
						case <-ctx.Done():
							authCh <- ctx.Err()
							return
						}

					case <-ctx.Done():
						authCh <- ctx.Err()
						return
					}

				case client.TypeAuthorizationStateReady:
					tdc.mu.Lock()
					tdc.isAuthorized = true
					tdc.mu.Unlock()
					return
				}
			}
		}
	}()

	tdClient, err := client.NewClient(authorizer)
	if err != nil {
		return err
	}

	tdc.mu.Lock()
	tdc.client = tdClient
	tdc.mu.Unlock()

	authErr := <-authCh
	if authErr != nil {
		_ = tdc.Close()
	}

	return authErr
}
