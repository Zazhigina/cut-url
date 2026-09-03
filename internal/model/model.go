package model

type CreateURLRequest struct {
	URL string `json:"url"`
}

type GetURLResponse struct {
	URL string `json:"url"`
}

type CreateURLResponse struct {
	ShortURL string `json:"cut_url"`
	Created  bool   `json:"created"`
	Message  string `json:"message,omitempty"`
}
