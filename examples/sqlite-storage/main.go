package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	ilinksdk "github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/event"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
	"github.com/the-yex/wechat-ilink-sdk/types"
)

// SQLiteTokenStore implements login.TokenStore using SQLite.
type SQLiteTokenStore struct {
	db *sql.DB
}

// NewSQLiteTokenStore creates a new SQLite token store.
func NewSQLiteTokenStore(dbPath string) (*SQLiteTokenStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			account_id TEXT PRIMARY KEY,
			token TEXT NOT NULL,
			base_url TEXT,
			user_id TEXT,
			saved_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteTokenStore{db: db}, nil
}

// Save saves the token for an account.
func (s *SQLiteTokenStore) Save(accountID string, token *login.TokenInfo) error {
	_, err := s.db.Exec(`
		INSERT INTO tokens (account_id, token, base_url, user_id, saved_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(account_id) DO UPDATE SET
			token = excluded.token,
			base_url = excluded.base_url,
			user_id = excluded.user_id,
			saved_at = CURRENT_TIMESTAMP
	`, accountID, token.Token, token.BaseURL, token.UserID)
	return err
}

// Load loads the token for an account.
func (s *SQLiteTokenStore) Load(accountID string) (*login.TokenInfo, error) {
	token := &login.TokenInfo{}
	err := s.db.QueryRow(`
		SELECT token, base_url, user_id
		FROM tokens WHERE account_id = ?
	`, accountID).Scan(&token.Token, &token.BaseURL, &token.UserID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return token, err
}

// Delete removes the token for an account.
func (s *SQLiteTokenStore) Delete(accountID string) error {
	_, err := s.db.Exec("DELETE FROM tokens WHERE account_id = ?", accountID)
	return err
}

// List lists all stored account IDs.
func (s *SQLiteTokenStore) List() ([]string, error) {
	rows, err := s.db.Query("SELECT account_id FROM tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		accounts = append(accounts, id)
	}
	return accounts, nil
}

// Close closes the database connection.
func (s *SQLiteTokenStore) Close() error {
	return s.db.Close()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n正在关闭...")
		cancel()
	}()

	// Initialize SQLite token store
	store, err := NewSQLiteTokenStore("./wechat-bot.db")
	if err != nil {
		log.Fatalf("初始化 SQLite 失败: %v", err)
	}
	defer store.Close()
	fmt.Println("SQLite 存储已初始化: ./wechat-bot.db")

	// Create client - just pass the token store, SDK handles everything else
	// (login, re-login on session expiry)
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(slog.Default()),
		ilinksdk.WithTokenStore(store),
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// Setup event handlers
	client.OnMessage(func(ctx context.Context, e *event.Event) error {
		msg := e.Data.(*ilink.Message)

		// Only handle user messages
		if !msg.IsFromUser() {
			return nil
		}

		// Handle text message
		if text := msg.GetText(); text != "" {
			fmt.Printf("收到文本消息: from=%s, content=%s\n", msg.FromUserID, text)
			return client.SendText(ctx, msg.FromUserID, "收到: "+text)
		}

		// Handle media message
		if item := msg.GetFirstMediaItem(); item != nil {
			switch item.Type {
			case types.MessageItemTypeImage:
				fmt.Printf("收到图片消息: from=%s\n", msg.FromUserID)
				// Download image and send back
				if item.ImageItem != nil && item.ImageItem.Media != nil {
					imageData, err := client.DownloadMedia(ctx, &media.DownloadRequest{
						EncryptQueryParam: item.ImageItem.Media.EncryptQueryParam,
						AESKey:            item.ImageItem.Media.AESKey,
					})
					if err != nil {
						fmt.Printf("下载图片失败: %v\n", err)
						return err
					}
					fmt.Printf("图片下载成功，大小: %d bytes\n", len(imageData))
					return client.SendImage(ctx, msg.FromUserID, imageData)
				}
				return client.SendText(ctx, msg.FromUserID, "收到图片(无法下载)")

			case types.MessageItemTypeVoice:
				fmt.Printf("收到语音消息: from=%s\n", msg.FromUserID)
				// Voice sending is not supported yet
				// Only acknowledge receipt
				if item.VoiceItem != nil {
					text := item.VoiceItem.Text
					if text != "" {
						return client.SendText(ctx, msg.FromUserID, "收到语音: "+text)
					}
					return client.SendText(ctx, msg.FromUserID, "收到语音消息")
				}
				return nil

			case types.MessageItemTypeVideo:
				fmt.Printf("收到视频消息: from=%s\n", msg.FromUserID)
				// Download video and send back
				if item.VideoItem != nil && item.VideoItem.Media != nil {
					videoData, err := client.DownloadMedia(ctx, &media.DownloadRequest{
						EncryptQueryParam: item.VideoItem.Media.EncryptQueryParam,
						AESKey:            item.VideoItem.Media.AESKey,
					})
					if err != nil {
						fmt.Printf("下载视频失败: %v\n", err)
						return err
					}
					fmt.Printf("视频下载成功，大小: %d bytes\n", len(videoData))
					return client.SendVideo(ctx, msg.FromUserID, videoData)
				}
				return client.SendText(ctx, msg.FromUserID, "收到视频(无法下载)")

			case types.MessageItemTypeFile:
				fmt.Printf("收到文件消息: from=%s\n", msg.FromUserID)
				// Download file and send back
				if item.FileItem != nil && item.FileItem.Media != nil {
					fileData, err := client.DownloadMedia(ctx, &media.DownloadRequest{
						EncryptQueryParam: item.FileItem.Media.EncryptQueryParam,
						AESKey:            item.FileItem.Media.AESKey,
					})
					if err != nil {
						fmt.Printf("下载文件失败: %v\n", err)
						return err
					}
					fileName := item.FileItem.FileName
					if fileName == "" {
						fileName = "file"
					}
					fmt.Printf("文件下载成功，大小: %d bytes\n", len(fileData))
					return client.SendFile(ctx, msg.FromUserID, fileName, fileData)
				}
				return client.SendText(ctx, msg.FromUserID, "收到文件(无法下载)")
			}
		}

		return nil
	})

	client.OnSessionExpired(func(ctx context.Context, e *event.Event) error {
		fmt.Println("会话已过期，需要重新登录")
		return nil
	})

	// Run the bot
	fmt.Println("启动机器人...")
	if err := client.Run(ctx, nil); err != nil {
		log.Printf("运行错误: %v", err)
	}

	fmt.Println("已关闭")
}
