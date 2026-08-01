package tdclient

import (
	"path/filepath"

	"github.com/zelenin/go-tdlib/client"
)

func parameters(id, apiHash string, apiID int32) *client.SetTdlibParametersRequest {
	return &client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join(".tdlib", id),
		FilesDirectory:      filepath.Join(".tdlib", id, "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               apiID,
		ApiHash:             apiHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
	}
}
