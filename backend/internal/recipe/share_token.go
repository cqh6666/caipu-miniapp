package recipe

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

// shareTokenLength 是 base64url 截断后的最终长度，约 132 位熵
const shareTokenLength = 22

// PublicRecipe 公开只读视图的菜谱主体（白名单 DTO）
// Open Q1 改进：明确白名单字段，避免后续给 Recipe 加字段时默认泄漏到公开接口
//
// 剔除原因：
//   - Note（个人备注）：偏私人内容，不应让接收者看到
//   - Link（原始来源链接）：可能是私域 / 内部链接，避免暴露
//   - KitchenID / CreatedBy / UpdatedBy：内部 ID，公开场景无意义
//   - CreatedAt / UpdatedAt / PinnedAt：时间戳与排序内部状态，公开场景无意义
//   - FlowchartProvider / FlowchartModel / FlowchartStatus 等流程图过程字段：仅当 FlowchartImageURL 非空时才展示流程图，过程信息无需暴露
//   - ParseStatus / ParseSource / ParseError 等解析过程字段：同上
//   - ShareToken：递归暴露 token 没有意义
type PublicRecipe struct {
	ID                  string        `json:"id"`
	Title               string        `json:"title"`
	Ingredient          string        `json:"ingredient"`
	Summary             string        `json:"summary"`
	ImageURL            string        `json:"imageUrl"`
	ImageURLs           []string      `json:"imageUrls"`
	FlowchartImageURL   string        `json:"flowchartImageUrl"`
	MealType            string        `json:"mealType"`
	Status              string        `json:"status"`
	ParsedContent       ParsedContent `json:"parsedContent"`
	ParsedContentEdited bool          `json:"parsedContentEdited"`
}

// toPublicRecipe 把内部 Recipe 投影成公开白名单 DTO
func toPublicRecipe(r Recipe) PublicRecipe {
	return PublicRecipe{
		ID:                  r.ID,
		Title:               r.Title,
		Ingredient:          r.Ingredient,
		Summary:             r.Summary,
		ImageURL:            r.ImageURL,
		ImageURLs:           r.ImageURLs,
		FlowchartImageURL:   r.FlowchartImageURL,
		MealType:            r.MealType,
		Status:              r.Status,
		ParsedContent:       r.ParsedContent,
		ParsedContentEdited: r.ParsedContentEdited,
	}
}

// PublicRecipeView 公开只读视图：菜谱主体 + 上下文（空间名 / 创建者昵称）
// 用于未登录或非空间成员通过 share_token 访问菜谱详情
type PublicRecipeView struct {
	Recipe      PublicRecipe `json:"recipe"`
	KitchenName string       `json:"kitchenName"`
	CreatorName string       `json:"creatorName"`
}

// EnsureShareToken 幂等地为菜谱生成 / 取回 share_token
// - 调用方需为空间成员（复用 GetByID 的鉴权链路）
// - 已存在 token：直接返回
// - 不存在：生成新 token 走「条件 UPDATE」原子写入；并发竞争失败时回查库里的现有 token
// P1 修复（并发原子化）：避免「先查空-生成-无条件写」导致并发下后写者覆盖先写者，
// 让先返回给前端的链接立刻失效的问题
func (s *Service) EnsureShareToken(ctx context.Context, userID int64, recipeID string) (string, error) {
	recipeID = strings.TrimSpace(recipeID)
	if recipeID == "" {
		return "", common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest)
	}

	// GetByID 内已做空间成员鉴权 + 软删过滤
	if _, err := s.GetByID(ctx, userID, recipeID); err != nil {
		return "", err
	}

	// 快路径：先读，已有则直接返回（绝大多数请求走这里）
	existing, err := s.repo.GetShareToken(ctx, recipeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", common.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}

	// 慢路径：生成新 token 走条件写
	token, err := generateShareToken()
	if err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	written, err := s.repo.SetShareToken(ctx, recipeID, token, now)
	if err != nil {
		return "", err
	}
	if written {
		// 本次成功抢到写入资格
		return token, nil
	}

	// 并发竞争失败：另一请求已先写入，回查 DB 里真正生效的 token
	winner, err := s.repo.GetShareToken(ctx, recipeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", common.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if winner == "" {
		// 极端罕见：刚被竞争对手写入又被异常清空，等同 NotFound 处理
		return "", common.ErrNotFound
	}
	return winner, nil
}

// GetByShareToken 公开只读访问：通过 share_token 查菜谱 + 空间名 + 创建者昵称
// 不做空间成员鉴权，token 失效或菜谱已删返回 ErrNotFound
func (s *Service) GetByShareToken(ctx context.Context, token string) (PublicRecipeView, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return PublicRecipeView{}, common.NewAppError(common.CodeBadRequest, "shareToken is required", http.StatusBadRequest)
	}

	item, err := s.repo.FindByShareToken(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicRecipeView{}, common.ErrNotFound
	}
	if err != nil {
		return PublicRecipeView{}, err
	}

	kitchenName, creatorName, err := s.repo.FindKitchenAndCreatorMeta(ctx, item.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicRecipeView{}, common.ErrNotFound
	}
	if err != nil {
		// 元数据查询失败不致命，返回菜谱主体即可保证只读体验可用
		kitchenName = ""
		creatorName = ""
	}

	return PublicRecipeView{
		Recipe:      toPublicRecipe(s.decorateRecipeRuntimeState(ctx, item)),
		KitchenName: kitchenName,
		CreatorName: creatorName,
	}, nil
}

// generateShareToken 生成 22 字符 base64url 的强随机 token
func generateShareToken() (string, error) {
	const rawBytes = 18 // 18 字节 → base64 24 字符 → 截断 22
	buf := make([]byte, rawBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	if len(encoded) > shareTokenLength {
		encoded = encoded[:shareTokenLength]
	}
	return encoded, nil
}
