package ilink

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_GetText(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		wantText string
	}{
		{
			name: "text message",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeText, TextItem: &TextItem{Text: "Hello"}},
				},
			},
			wantText: "Hello",
		},
		{
			name: "image message",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeImage, ImageItem: &ImageItem{}},
				},
			},
			wantText: "",
		},
		{
			name: "mixed message with text",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeImage, ImageItem: &ImageItem{}},
					{Type: MessageItemTypeText, TextItem: &TextItem{Text: "Caption"}},
				},
			},
			wantText: "Caption",
		},
		{
			name:     "empty message",
			msg:      &Message{},
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantText, tt.msg.GetText())
		})
	}
}

func TestMessage_GetFirstMediaItem(t *testing.T) {
	tests := []struct {
		name      string
		msg       *Message
		wantType  MessageItemType
		wantFound bool
	}{
		{
			name: "image first",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeImage, ImageItem: &ImageItem{}},
					{Type: MessageItemTypeText, TextItem: &TextItem{}},
				},
			},
			wantType:  MessageItemTypeImage,
			wantFound: true,
		},
		{
			name: "video first",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeVideo, VideoItem: &VideoItem{}},
				},
			},
			wantType:  MessageItemTypeVideo,
			wantFound: true,
		},
		{
			name: "text only",
			msg: &Message{
				ItemList: []*MessageItem{
					{Type: MessageItemTypeText, TextItem: &TextItem{}},
				},
			},
			wantFound: false,
		},
		{
			name:      "empty",
			msg:       &Message{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.msg.GetFirstMediaItem()
			if tt.wantFound {
				assert.NotNil(t, item)
				assert.Equal(t, tt.wantType, item.Type)
			} else {
				assert.Nil(t, item)
			}
		})
	}
}

func TestMessage_IsFromUser(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		wantIs  bool
	}{
		{"user message", MessageTypeUser, true},
		{"bot message", MessageTypeBot, false},
		{"none", MessageTypeNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{MessageType: tt.msgType}
			assert.Equal(t, tt.wantIs, msg.IsFromUser())
		})
	}
}

func TestMessage_IsNew(t *testing.T) {
	tests := []struct {
		name   string
		state  MessageState
		wantIs bool
	}{
		{"new", MessageStateNew, true},
		{"generating", MessageStateGenerating, false},
		{"finish", MessageStateFinish, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{MessageState: tt.state}
			assert.Equal(t, tt.wantIs, msg.IsNew())
		})
	}
}

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage("user1", "Hello World", "token123")

	assert.Equal(t, "user1", msg.ToUserID)
	assert.Equal(t, MessageTypeBot, msg.MessageType)
	assert.Equal(t, "token123", msg.ContextToken)
	assert.Len(t, msg.ItemList, 1)
	assert.Equal(t, MessageItemTypeText, msg.ItemList[0].Type)
	assert.Equal(t, "Hello World", msg.ItemList[0].TextItem.Text)
}

func TestNewImageMessage(t *testing.T) {
	imageItem := &ImageItem{
		Media: &CDNMedia{
			EncryptQueryParam: "param123",
			AESKey:            "key123",
		},
	}

	msg := NewImageMessage("user1", "token123", imageItem)

	assert.Equal(t, "user1", msg.ToUserID)
	assert.Equal(t, MessageTypeBot, msg.MessageType)
	assert.Equal(t, MessageItemTypeImage, msg.ItemList[0].Type)
	assert.Equal(t, imageItem, msg.ItemList[0].ImageItem)
}

func TestNewVideoMessage(t *testing.T) {
	videoItem := &VideoItem{
		Media: &CDNMedia{EncryptQueryParam: "param123"},
	}

	msg := NewVideoMessage("user1", "token123", videoItem)

	assert.Equal(t, MessageItemTypeVideo, msg.ItemList[0].Type)
	assert.Equal(t, videoItem, msg.ItemList[0].VideoItem)
}

func TestNewFileMessage(t *testing.T) {
	fileItem := &FileItem{
		FileName: "test.pdf",
		MD5:      "md5hash",
	}

	msg := NewFileMessage("user1", "token123", fileItem)

	assert.Equal(t, MessageItemTypeFile, msg.ItemList[0].Type)
	assert.Equal(t, fileItem, msg.ItemList[0].FileItem)
}

func TestAPIError(t *testing.T) {
	err := &APIError{Code: -14, Message: "session expired"}

	assert.Equal(t, "api error: code=-14, message=session expired", err.Error())
	assert.True(t, err.IsSessionExpired())

	err2 := &APIError{Code: 0, Message: "ok"}
	assert.False(t, err2.IsSessionExpired())
}
