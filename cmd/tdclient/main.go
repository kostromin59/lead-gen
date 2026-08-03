package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kostromin59/lead-gen/internal/clients/tdclient"
	"github.com/kostromin59/lead-gen/internal/models"
	"github.com/kostromin59/lead-gen/internal/repositories/messages"
	"github.com/zelenin/go-tdlib/client"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found", slog.String("err", err.Error()))
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_CONN"))
	if err != nil {
		panic(err)
	}

	messagesRepo := messages.New(pool)

	apiIDRaw := os.Getenv("APP_ID")
	apiID, err := strconv.Atoi(apiIDRaw)
	if err != nil {
		panic(err)
	}

	tdc := tdclient.New("s. kostromin", os.Getenv("PHONE"), os.Getenv("API_HASH"), int32(apiID))
	defer tdc.Close()

	eventAuthProvider := tdclient.NewEventAuthProvider(
		func(ctx context.Context, id string, ch chan<- string) {
			var code string

			fmt.Printf("Enter code for account id %q: ", id)
			fmt.Scanln(&code)
			ch <- code
		},
		func(ctx context.Context, id string, ch chan<- string) {
			var password string

			fmt.Printf("Enter password for account id %q: ", id)
			fmt.Scanln(&password)
			ch <- password
		},
	)

	if err := tdc.Authorize(context.Background(), eventAuthProvider); err != nil {
		panic(err)
	}

	me, err := tdc.Client().GetMe()
	if err != nil {
		panic(err)
	}
	log.Printf("FirstName: %q", me.FirstName)

	listener := tdc.Client().GetListener()

	for update := range listener.Updates {
		if update.GetClass() != client.ClassUpdate {
			continue
		}

		switch updateByType := update.(type) {
		case *client.UpdateNewMessage:
			switch content := updateByType.Message.Content.(type) {
			case *client.MessageText:

				chat, err := tdc.Client().GetChat(&client.GetChatRequest{
					ChatId: updateByType.Message.ChatId,
				})
				if err != nil {
					slog.Error("unable to get chat by id", slog.String("err", err.Error()))
					continue
				}

				var description *string
				switch chat.Type.(type) {
				case *client.ChatTypeBasicGroup:
					channelType := chat.Type.(*client.ChatTypeBasicGroup)
					channelID := channelType.BasicGroupId

					channel, err := tdc.Client().GetBasicGroupFullInfo(&client.GetBasicGroupFullInfoRequest{
						BasicGroupId: channelID,
					})
					if err != nil {
						slog.Error("Ошибка получения супергруппы",
							"error", err,
							"chat_id", chat.Id,
							"basic_group_id", channelID)
					} else {
						description = &channel.Description
					}
				case *client.ChatTypeSupergroup:
					channelType := chat.Type.(*client.ChatTypeSupergroup)
					superGroup, err := tdc.Client().GetSupergroupFullInfo(&client.GetSupergroupFullInfoRequest{
						channelType.SupergroupId,
					})
					if err != nil {
						slog.Error("Ошибка получения супергруппы",
							"error", err,
							"chat_id", chat.Id,
							"super_group_id", superGroup)
					} else {
						description = &superGroup.Description
					}
				}
				// sender := updateByType.Message.SenderId
				// userID := sender.(*client.MessageSenderUser).UserId
				// sender, err := tdc.Client().GetUserFullInfo(&client.GetUserFullInfoRequest{
				// 	// UserId:
				// })
				var senderID int64
				var senderName string

				switch s := updateByType.Message.SenderId.(type) {
				case *client.MessageSenderUser:
					senderID = s.UserId
					user, err := tdc.Client().GetUser(&client.GetUserRequest{
						UserId: s.UserId,
					})
					if err != nil {
						slog.Error("Ошибка получения пользователя", "error", err, "user_id", s.UserId)
						senderName = "Unknown User"
					} else {
						senderName = user.FirstName
						if user.LastName != "" {
							senderName += " " + user.LastName
						}
					}

				case *client.MessageSenderChat:
					senderID = s.ChatId
					chatInfo, err := tdc.Client().GetChat(&client.GetChatRequest{
						ChatId: s.ChatId,
					})
					if err != nil {
						slog.Error("Ошибка получения чата-отправителя", "error", err, "chat_id", s.ChatId)
						senderName = "Unknown Chat"
					} else {
						senderName = chatInfo.Title
					}
				}

				threadID := fmt.Sprintf("%d", updateByType.Message.MessageThreadId)

				log.Println("creating")
				if err := messagesRepo.Create(context.Background(), models.CreateMessage{
					Content:         content.Text.Text,
					ChatID:          fmt.Sprintf("%d", chat.Id),
					ChatName:        chat.Title,
					ChatDescription: description,
					MessageID:       fmt.Sprintf("%d", updateByType.Message.Id),
					ThreadID:        threadID,
					MessageTime:     time.Unix(int64(updateByType.Message.Date), 0),
					SenderID:        fmt.Sprintf("%d", senderID),
					SenderName:      senderName,
					AccountID:       tdc.ID(),
				}); err != nil {
					slog.Error("unable to create message", slog.String("err", err.Error()))
				}
			}
		}

	}
}
