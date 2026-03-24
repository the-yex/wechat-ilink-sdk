package types

// Message represents a unified WeChat message.
type Message struct {
	Seq          int64          `json:"seq,omitempty"`
	MessageID    int64          `json:"message_id,omitempty"`
	FromUserID   string         `json:"from_user_id,omitempty"`
	ToUserID     string         `json:"to_user_id,omitempty"`
	ClientID     string         `json:"client_id,omitempty"`
	CreateTimeMs int64          `json:"create_time_ms,omitempty"`
	UpdateTimeMs int64          `json:"update_time_ms,omitempty"`
	DeleteTimeMs int64          `json:"delete_time_ms,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	GroupID      string         `json:"group_id,omitempty"`
	MessageType  MessageType    `json:"message_type,omitempty"`
	MessageState MessageState   `json:"message_state,omitempty"`
	ItemList     []*MessageItem `json:"item_list,omitempty"`
	ContextToken string         `json:"context_token,omitempty"`
}

// GetText extracts text content from a message.
func (m *Message) GetText() string {
	for _, item := range m.ItemList {
		if item.Type == MessageItemTypeText && item.TextItem != nil {
			return item.TextItem.Text
		}
	}
	return ""
}

// GetFirstMediaItem returns the first media item (image > video > file > voice).
func (m *Message) GetFirstMediaItem() *MessageItem {
	for _, t := range []MessageItemType{
		MessageItemTypeImage,
		MessageItemTypeVideo,
		MessageItemTypeFile,
		MessageItemTypeVoice,
	} {
		for _, item := range m.ItemList {
			if item.Type == t {
				return item
			}
		}
	}
	return nil
}

// IsFromUser returns true if the message is from a user (not a bot).
func (m *Message) IsFromUser() bool {
	return m.MessageType == MessageTypeUser
}

// IsNew returns true if the message state is NEW.
func (m *Message) IsNew() bool {
	return m.MessageState == MessageStateNew
}

// NewTextMessage creates a new text message for sending.
func NewTextMessage(toUserID, text, contextToken string) *Message {
	return &Message{
		ToUserID:     toUserID,
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: contextToken,
		ItemList: []*MessageItem{
			{
				Type:     MessageItemTypeText,
				TextItem: &TextItem{Text: text},
			},
		},
	}
}

// NewImageMessage creates a new image message for sending.
func NewImageMessage(toUserID, contextToken string, imageItem *ImageItem) *Message {
	return &Message{
		ToUserID:     toUserID,
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: contextToken,
		ItemList: []*MessageItem{
			{
				Type:      MessageItemTypeImage,
				ImageItem: imageItem,
			},
		},
	}
}

// NewVideoMessage creates a new video message for sending.
func NewVideoMessage(toUserID, contextToken string, videoItem *VideoItem) *Message {
	return &Message{
		ToUserID:     toUserID,
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: contextToken,
		ItemList: []*MessageItem{
			{
				Type:      MessageItemTypeVideo,
				VideoItem: videoItem,
			},
		},
	}
}

// NewFileMessage creates a new file message for sending.
func NewFileMessage(toUserID, contextToken string, fileItem *FileItem) *Message {
	return &Message{
		ToUserID:     toUserID,
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: contextToken,
		ItemList: []*MessageItem{
			{
				Type:     MessageItemTypeFile,
				FileItem: fileItem,
			},
		},
	}
}