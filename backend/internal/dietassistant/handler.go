package dietassistant

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) StreamChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req ChatStreamRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	messages, err := normalizeRequestMessages(req.Messages)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteError(w, common.NewAppError(common.CodeInternalServer, "streaming is not supported", http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	emit := func(event StreamEvent) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte("data: ")); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := h.service.StreamChat(r.Context(), ChatContext{
		UserID:    userID,
		KitchenID: req.KitchenID,
	}, messages, emit); err != nil {
		_ = emit(StreamEvent{
			Type:    "error",
			Message: streamErrorMessage(err),
		})
	}
}

func normalizeRequestMessages(messages []ChatMessage) ([]ChatMessage, error) {
	if len(messages) == 0 {
		return nil, common.NewAppError(common.CodeBadRequest, "messages are required", http.StatusBadRequest)
	}
	if len(messages) > 24 {
		messages = messages[len(messages)-24:]
	}

	result := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(strings.ToLower(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			return nil, common.NewAppError(common.CodeBadRequest, "message role must be user or assistant", http.StatusBadRequest)
		}
		if len([]rune(content)) > 2000 {
			return nil, common.NewAppError(common.CodeBadRequest, "message content is too long", http.StatusBadRequest)
		}
		result = append(result, ChatMessage{
			Role:    role,
			Content: content,
		})
	}
	if len(result) == 0 {
		return nil, common.NewAppError(common.CodeBadRequest, "messages are required", http.StatusBadRequest)
	}
	if result[len(result)-1].Role != "user" {
		return nil, common.NewAppError(common.CodeBadRequest, "last message must be from user", http.StatusBadRequest)
	}
	return result, nil
}

func streamErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "饮食管家暂时不可用"
	}
	return message
}
