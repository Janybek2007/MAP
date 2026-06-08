package api

type tokenRequest struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type addChildCategoryRequest struct {
	Label string `json:"label"`
	Key   string `json:"key,omitempty"`
}

type updateChildCategoryRequest struct {
	Label string `json:"label"`
	Key   string `json:"key,omitempty"`
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
