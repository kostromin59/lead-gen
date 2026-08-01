package tdclient

import (
	"sync"

	"github.com/zelenin/go-tdlib/client"
)

var once sync.Once

func init() {
	once.Do(func() {
		_, _ = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
			NewVerbosityLevel: 1,
		})
	})
}
