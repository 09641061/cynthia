package whatsapp

// Config holds startup-injected Meta WhatsApp Cloud API settings.
type Config struct {
	AccessToken     string
	PhoneNumberID   string
	GraphAPIVersion string
	VerifyToken     string
	AppSecret       string
}

type sendRequest struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             sendTextBody `json:"text"`
}

type sendTextBody struct {
	Body string `json:"body"`
}

type sendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}
