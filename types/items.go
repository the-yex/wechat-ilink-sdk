package types

// TextItem represents text content in a message.
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// CDNMedia represents a CDN media reference with AES key.
type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	EncryptType       int    `json:"encrypt_type,omitempty"`
}

// ImageItem represents an image in a message.
type ImageItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	AESKey      string    `json:"aeskey,omitempty"`
	URL         string    `json:"url,omitempty"`
	MidSize     int       `json:"mid_size,omitempty"`
	ThumbSize   int       `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	HDSize      int       `json:"hd_size,omitempty"`
}

// VoiceItem represents a voice message.
type VoiceItem struct {
	Media         *CDNMedia `json:"media,omitempty"`
	EncodeType    int       `json:"encode_type,omitempty"`    // 1=pcm 2=adpcm 3=feature 4=speex 5=amr 6=silk 7=mp3 8=ogg-speex
	BitsPerSample int       `json:"bits_per_sample,omitempty"`
	SampleRate    int       `json:"sample_rate,omitempty"`
	Playtime      int       `json:"playtime,omitempty"` // Duration in milliseconds
	Text          string    `json:"text,omitempty"`     // Voice-to-text content
}

// FileItem represents a file attachment.
type FileItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	MD5      string    `json:"md5,omitempty"`
	Len      string    `json:"len,omitempty"`
}

// VideoItem represents a video message.
type VideoItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	VideoSize   int       `json:"video_size,omitempty"`
	PlayLength  int       `json:"play_length,omitempty"`
	VideoMD5    string    `json:"video_md5,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	ThumbSize   int       `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
}

// RefMessage represents a referenced (quoted) message.
type RefMessage struct {
	MessageItem *MessageItem `json:"message_item,omitempty"`
	Title       string       `json:"title,omitempty"`
}

// MessageItem represents a single item in a message.
type MessageItem struct {
	Type         MessageItemType `json:"type,omitempty"`
	CreateTimeMs int64           `json:"create_time_ms,omitempty"`
	UpdateTimeMs int64           `json:"update_time_ms,omitempty"`
	IsCompleted  bool            `json:"is_completed,omitempty"`
	MsgID        string          `json:"msg_id,omitempty"`
	TextItem     *TextItem       `json:"text_item,omitempty"`
	ImageItem    *ImageItem      `json:"image_item,omitempty"`
	VoiceItem    *VoiceItem      `json:"voice_item,omitempty"`
	FileItem     *FileItem       `json:"file_item,omitempty"`
	VideoItem    *VideoItem      `json:"video_item,omitempty"`
	RefMsg       *RefMessage     `json:"ref_msg,omitempty"`
}