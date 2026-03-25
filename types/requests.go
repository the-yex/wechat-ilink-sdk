package types

// GetUpdatesRequest represents a getUpdates API request.
type GetUpdatesRequest struct {
	// Deprecated: Use GetUpdatesBuf instead
	SyncBuf string `json:"sync_buf,omitempty"`
	// Full context buf cached locally; send "" when none (first request or after reset)
	GetUpdatesBuf string   `json:"get_updates_buf,omitempty"`
	BaseInfo      BaseInfo `json:"base_info"`
}

// GetUpdatesResponse represents a getUpdates API response.
type GetUpdatesResponse struct {
	Ret               int64      `json:"ret,omitempty"`
	ErrCode           int        `json:"errcode,omitempty"`
	ErrMsg            string     `json:"errmsg,omitempty"`
	Messages          []*Message `json:"msgs,omitempty"`
	SyncBuf           string     `json:"sync_buf,omitempty"` // Deprecated
	GetUpdatesBuf     string     `json:"get_updates_buf,omitempty"`
	LongPollTimeoutMs int        `json:"longpolling_timeout_ms,omitempty"`
}

// SendMessageRequest represents a sendMessage API request.
type SendMessageRequest struct {
	Message  *Message  `json:"msg,omitempty"`
	BaseInfo BaseInfo  `json:"base_info"`
}

// SendMessageResponse represents a sendMessage API response.
type SendMessageResponse struct {
	// Empty response on success
}

// GetUploadURLRequest represents a getUploadUrl API request.
type GetUploadURLRequest struct {
	FileKey         string          `json:"filekey,omitempty"`
	MediaType       UploadMediaType `json:"media_type,omitempty"`
	ToUserID        string          `json:"to_user_id,omitempty"`
	RawSize         int             `json:"rawsize,omitempty"`
	RawFileMD5      string          `json:"rawfilemd5,omitempty"`
	FileSize        int             `json:"filesize,omitempty"`
	ThumbRawSize    int             `json:"thumb_rawsize,omitempty"`
	ThumbRawFileMD5 string          `json:"thumb_rawfilemd5,omitempty"`
	ThumbFileSize   int             `json:"thumb_filesize,omitempty"`
	NoNeedThumb     bool            `json:"no_need_thumb,omitempty"`
	AESKey          string          `json:"aeskey,omitempty"`
	BaseInfo        BaseInfo        `json:"base_info"`
}

// GetUploadURLResponse represents a getUploadUrl API response.
type GetUploadURLResponse struct {
	UploadParam      string `json:"upload_param,omitempty"`
	ThumbUploadParam string `json:"thumb_upload_param,omitempty"`
}

// SendTypingRequest represents a sendTyping API request.
type SendTypingRequest struct {
	ILinkUserID  string   `json:"ilink_user_id,omitempty"`
	TypingTicket string   `json:"typing_ticket,omitempty"`
	Status       int      `json:"status,omitempty"` // 1=typing, 2=cancel
	BaseInfo     BaseInfo `json:"base_info"`
}

// SendTypingResponse represents a sendTyping API response.
type SendTypingResponse struct {
	Ret    int    `json:"ret,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
}

// GetConfigRequest represents a getConfig API request.
type GetConfigRequest struct {
	ILinkUserID  string   `json:"ilink_user_id,omitempty"`
	ContextToken string   `json:"context_token,omitempty"`
	BaseInfo     BaseInfo `json:"base_info"`
}

// GetConfigResponse represents a getConfig API response.
type GetConfigResponse struct {
	Ret          int    `json:"ret,omitempty"`
	ErrCode      int    `json:"errcode,omitempty"`
	ErrMsg       string `json:"errmsg,omitempty"`
	TypingTicket string `json:"typing_ticket,omitempty"`
}

// GetBotQRCodeRequest represents a get_bot_qrcode API request.
type GetBotQRCodeRequest struct {
	BotType  string   `json:"bot_type,omitempty"`
	BaseInfo BaseInfo `json:"base_info"`
}

// GetBotQRCodeResponse represents a get_bot_qrcode API response.
type GetBotQRCodeResponse struct {
	Ret      int    `json:"ret,omitempty"`
	QRCode   string `json:"qrcode,omitempty"`
	ImageURL string `json:"qrcode_img_content,omitempty"` // API returns qrcode_img_content
}

// GetQRCodeStatusRequest represents a get_qrcode_status API request.
type GetQRCodeStatusRequest struct {
	QRCode   string   `json:"qrcode,omitempty"`
	BaseInfo BaseInfo `json:"base_info"`
}

// GetQRCodeStatusResponse represents a get_qrcode_status API response.
type GetQRCodeStatusResponse struct {
	Ret         int         `json:"ret,omitempty"`
	ErrCode     int         `json:"errcode,omitempty"`
	ErrMsg      string      `json:"errmsg,omitempty"`
	Status      LoginStatus `json:"status,omitempty"`
	BotToken    string      `json:"bot_token,omitempty"`
	ILinkBotID  string      `json:"ilink_bot_id,omitempty"`
	ILinkUserID string      `json:"ilink_user_id,omitempty"`
	BaseURL     string      `json:"baseurl,omitempty"`
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	Token     string // Bot token
	AccountID string // Bot account ID (ilink_bot_id)
	UserID    string // User ID who scanned (ilink_user_id)
	BaseURL   string // API base URL (may differ per account)
}
