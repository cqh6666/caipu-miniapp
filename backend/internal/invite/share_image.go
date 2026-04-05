package invite

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	shareImageWidth  = 1080
	shareImageHeight = 864
)

var (
	shareImageBackground     = rgbaHex(0xF7F1EA, 255)
	shareImageCardBackground = rgbaHex(0xFFFDF9, 255)
	shareImageCardBorder     = rgbaHex(0xE6DACD, 255)
	shareImageShadowColor    = rgbaHex(0x6F5645, 18)
	shareImageTextPrimary    = rgbaHex(0x2F2822, 255)
	shareImageTextMuted      = rgbaHex(0x8B7A6B, 255)
	shareImageChipWarm       = rgbaHex(0xF2E8DD, 255)
	shareImageChipWarmText   = rgbaHex(0x89624A, 255)
	shareImageChipGreen      = rgbaHex(0xE8F0E8, 255)
	shareImageChipGreenText  = rgbaHex(0x5E7A65, 255)
	shareImageFooterDark     = rgbaHex(0x4B392F, 255)
	shareImageFooterText     = rgbaHex(0xFFF9F3, 255)
	shareImageDangerBg       = rgbaHex(0xFAE8DF, 255)
	shareImageDangerText     = rgbaHex(0xA15F39, 255)
)

type ShareImageRenderer struct {
	regularFontPath string
	boldFontPath    string

	mu          sync.Mutex
	regularFont *sfnt.Font
	boldFont    *sfnt.Font
	faceCache   map[string]font.Face
}

type ShareImageData struct {
	KitchenName   string
	InviterName   string
	InviteCode    string
	Status        string
	MemberCount   int
	RemainingUses int
	ExpiresAt     string
}

func NewShareImageRenderer(regularFontPath, boldFontPath string) *ShareImageRenderer {
	return &ShareImageRenderer{
		regularFontPath: strings.TrimSpace(regularFontPath),
		boldFontPath:    strings.TrimSpace(boldFontPath),
		faceCache:       make(map[string]font.Face),
	}
}

func (r *ShareImageRenderer) Render(data ShareImageData) ([]byte, error) {
	if _, err := r.face(false, 24); err != nil {
		return nil, err
	}
	if _, err := r.face(true, 24); err != nil {
		return nil, err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, shareImageWidth, shareImageHeight))
	fillRect(canvas, canvas.Bounds(), shareImageBackground)

	drawCircle(canvas, 140, 110, 180, rgbaHex(0xF3D7BC, 118))
	drawCircle(canvas, shareImageWidth-84, 148, 220, rgbaHex(0xE8D9C8, 92))
	drawCircle(canvas, shareImageWidth-120, shareImageHeight-96, 180, rgbaHex(0xD9E7D8, 100))

	cardRect := image.Rect(74, 58, shareImageWidth-74, shareImageHeight-92)
	drawLayeredShadow(canvas, cardRect, 50)
	fillRoundedRect(canvas, cardRect, 48, shareImageCardBackground)
	strokeRoundedRect(canvas, cardRect, 48, 2, shareImageCardBorder, shareImageCardBackground)

	appChipRect := image.Rect(cardRect.Min.X+56, cardRect.Min.Y+54, cardRect.Min.X+336, cardRect.Min.Y+104)
	fillRoundedRect(canvas, appChipRect, 25, shareImageChipWarm)
	drawText(canvas, appChipRect.Min.X+28, appChipRect.Min.Y+35, "我们的数字厨房", mustFace(r.face(false, 24)), shareImageChipWarmText)

	statusLabel, statusBg, statusText := buildStatusVisual(data.Status)
	statusWidth := max(176, measureTextWidth(mustFace(r.face(true, 22)), statusLabel)+50)
	statusRect := image.Rect(cardRect.Max.X-statusWidth-56, cardRect.Min.Y+54, cardRect.Max.X-56, cardRect.Min.Y+104)
	fillRoundedRect(canvas, statusRect, 25, statusBg)
	drawCenteredText(canvas, statusRect, statusLabel, mustFace(r.face(true, 22)), statusText)

	avatarRect := image.Rect(cardRect.Min.X+56, cardRect.Min.Y+144, cardRect.Min.X+140, cardRect.Min.Y+228)
	fillRoundedRect(canvas, avatarRect, 28, shareImageChipWarm)
	drawCenteredText(canvas, avatarRect, inviterInitial(data.InviterName), mustFace(r.face(true, 34)), shareImageChipWarmText)

	inviterLine := fmt.Sprintf("%s 邀请你加入", safeFallback(data.InviterName, "厨房成员"))
	drawText(canvas, cardRect.Min.X+166, cardRect.Min.Y+188, inviterLine, mustFace(r.face(true, 34)), shareImageChipWarmText)

	titleLines := wrapText(mustFace(r.face(true, 66)), safeFallback(data.KitchenName, "这间共享厨房"), 2, 620)
	titleY := cardRect.Min.Y + 292
	for index, line := range titleLines {
		drawText(canvas, cardRect.Min.X+56, titleY+index*82, line, mustFace(r.face(true, 66)), shareImageTextPrimary)
	}

	drawText(
		canvas,
		cardRect.Min.X+56,
		cardRect.Min.Y+444,
		"一起维护菜单，同步想吃和吃过",
		mustFace(r.face(false, 30)),
		shareImageTextMuted,
	)
	drawText(
		canvas,
		cardRect.Min.X+56,
		cardRect.Min.Y+492,
		"加入后可查看同一份菜谱、菜单安排和厨房成员。",
		mustFace(r.face(false, 24)),
		shareImageTextMuted,
	)

	chipY := cardRect.Min.Y + 540
	drawMiniChip(canvas, image.Rect(cardRect.Min.X+56, chipY, cardRect.Min.X+196, chipY+48), "同步菜单", shareImageChipWarm, shareImageChipWarmText, mustFace(r.face(true, 22)))
	drawMiniChip(canvas, image.Rect(cardRect.Min.X+210, chipY, cardRect.Min.X+350, chipY+48), "共享菜谱", shareImageChipWarm, shareImageChipWarmText, mustFace(r.face(true, 22)))
	drawMiniChip(canvas, image.Rect(cardRect.Min.X+364, chipY, cardRect.Min.X+504, chipY+48), "自由切换", shareImageChipGreen, shareImageChipGreenText, mustFace(r.face(true, 22)))

	panelRect := image.Rect(cardRect.Min.X+42, cardRect.Min.Y+612, cardRect.Max.X-42, cardRect.Min.Y+760)
	fillRoundedRect(canvas, panelRect, 32, rgbaHex(0xF8F1E8, 255))
	strokeRoundedRect(canvas, panelRect, 32, 2, rgbaHex(0xE7DBCE, 255), rgbaHex(0xF8F1E8, 255))

	drawMetricCard(canvas, image.Rect(panelRect.Min.X+26, panelRect.Min.Y+22, panelRect.Min.X+214, panelRect.Max.Y-22), "当前成员", fmt.Sprintf("%d", max(data.MemberCount, 1)), "位成员", mustFace(r.face(true, 40)), mustFace(r.face(false, 22)))
	drawMetricCard(canvas, image.Rect(panelRect.Min.X+232, panelRect.Min.Y+22, panelRect.Min.X+420, panelRect.Max.Y-22), "剩余名额", fmt.Sprintf("%d", max(data.RemainingUses, 0)), "位可加入", mustFace(r.face(true, 40)), mustFace(r.face(false, 22)))
	drawMetricCard(canvas, image.Rect(panelRect.Min.X+438, panelRect.Min.Y+22, panelRect.Max.X-26, panelRect.Max.Y-22), "有效期", formatShareExpiryText(data.ExpiresAt, data.Status), buildFooterMeta(data.Status), mustFace(r.face(true, 34)), mustFace(r.face(false, 20)))

	footerRect := image.Rect(cardRect.Min.X+42, cardRect.Max.Y-74, cardRect.Max.X-42, cardRect.Max.Y-14)
	fillRoundedRect(canvas, footerRect, 30, shareImageFooterDark)
	footerText := fmt.Sprintf("邀请码 %s", formatInviteCode(safeFallback(data.InviteCode, "---- ----")))
	drawCenteredText(canvas, footerRect, footerText, mustFace(r.face(true, 30)), shareImageFooterText)

	buffer := bytes.NewBuffer(nil)
	if err := png.Encode(buffer, canvas); err != nil {
		return nil, fmt.Errorf("encode share image: %w", err)
	}

	return buffer.Bytes(), nil
}

func (r *ShareImageRenderer) face(bold bool, size float64) (font.Face, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fontKey := fmt.Sprintf("%t-%.2f", bold, size)
	if face, ok := r.faceCache[fontKey]; ok {
		return face, nil
	}

	targetFont, err := r.fontFile(bold)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(targetFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create share image font face: %w", err)
	}

	r.faceCache[fontKey] = face
	return face, nil
}

func (r *ShareImageRenderer) fontFile(bold bool) (*sfnt.Font, error) {
	if bold && r.boldFont != nil {
		return r.boldFont, nil
	}
	if !bold && r.regularFont != nil {
		return r.regularFont, nil
	}

	targetPath := r.regularFontPath
	fallbacks := shareImageRegularFontCandidates()
	if bold {
		targetPath = r.boldFontPath
		fallbacks = shareImageBoldFontCandidates()
	}

	fontFile, err := loadFontFromCandidates(targetPath, fallbacks)
	if err != nil {
		if bold && r.regularFont != nil {
			return r.regularFont, nil
		}
		return nil, err
	}

	if bold {
		r.boldFont = fontFile
		if r.regularFont == nil {
			r.regularFont = fontFile
		}
		return r.boldFont, nil
	}

	r.regularFont = fontFile
	return r.regularFont, nil
}

func loadFontFromCandidates(primary string, candidates []string) (*sfnt.Font, error) {
	paths := make([]string, 0, len(candidates)+1)
	if strings.TrimSpace(primary) != "" {
		paths = append(paths, primary)
	}
	paths = append(paths, candidates...)

	seen := make(map[string]struct{}, len(paths))
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		if collection, err := sfnt.ParseCollection(data); err == nil {
			for index := 0; index < collection.NumFonts(); index++ {
				fontFile, fontErr := collection.Font(index)
				if fontErr == nil {
					return fontFile, nil
				}
			}
		}

		fontFile, err := opentype.Parse(data)
		if err == nil {
			return fontFile, nil
		}
	}

	return nil, fmt.Errorf("no available font found for invite share image; set INVITE_SHARE_FONT_PATH / INVITE_SHARE_FONT_BOLD_PATH")
}

func shareImageRegularFontCandidates() []string {
	return []string{
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/Supplemental/PingFang.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansSC-Regular.otf",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	}
}

func shareImageBoldFontCandidates() []string {
	return []string{
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/Supplemental/PingFang.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansSC-Bold.otf",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	}
}

func buildStatusVisual(status string) (label string, background color.Color, foreground color.Color) {
	switch strings.TrimSpace(status) {
	case statusExpired:
		return "已过期", shareImageDangerBg, shareImageDangerText
	case statusUsedUp:
		return "名额已满", shareImageDangerBg, shareImageDangerText
	case statusRevoked:
		return "已失效", shareImageDangerBg, shareImageDangerText
	default:
		return "可立即加入", shareImageChipGreen, shareImageChipGreenText
	}
}

func buildFooterMeta(status string) string {
	switch strings.TrimSpace(status) {
	case statusExpired:
		return "邀请已过期"
	case statusUsedUp:
		return "当前名额已满"
	case statusRevoked:
		return "邀请已失效"
	default:
		return "打开后即可加入"
	}
}

func inviterInitial(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "厨"
	}

	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return "厨"
	}

	return strings.ToUpper(string(r))
}

func formatShareExpiryText(value, status string) string {
	if strings.TrimSpace(status) == statusExpired {
		return "当前不可加入"
	}

	if strings.TrimSpace(value) == "" {
		return "--"
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return strings.ReplaceAll(value, "T", " ")
	}

	return parsed.Format("01-02 15:04")
}

func formatInviteCode(value string) string {
	normalized := normalizeInviteCode(value)
	if normalized == "" {
		return "---- ----"
	}
	if len(normalized) <= 4 {
		return normalized
	}
	return normalized[:4] + " " + normalized[4:]
}

func safeFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func drawLayeredShadow(img *image.RGBA, rect image.Rectangle, radius int) {
	for _, layer := range []struct {
		offsetX int
		offsetY int
		expand  int
		alpha   uint8
	}{
		{offsetX: 0, offsetY: 12, expand: 0, alpha: 18},
		{offsetX: 0, offsetY: 18, expand: 6, alpha: 10},
		{offsetX: 0, offsetY: 24, expand: 12, alpha: 6},
	} {
		shadowRect := image.Rect(
			rect.Min.X-layer.expand+layer.offsetX,
			rect.Min.Y-layer.expand+layer.offsetY,
			rect.Max.X+layer.expand+layer.offsetX,
			rect.Max.Y+layer.expand+layer.offsetY,
		)
		fillRoundedRect(img, shadowRect, radius+layer.expand, color.RGBA{
			R: shareImageShadowColor.R,
			G: shareImageShadowColor.G,
			B: shareImageShadowColor.B,
			A: layer.alpha,
		})
	}
}

func drawMiniChip(img *image.RGBA, rect image.Rectangle, label string, background color.Color, foreground color.Color, face font.Face) {
	fillRoundedRect(img, rect, rect.Dy()/2, background)
	drawCenteredText(img, rect, label, face, foreground)
}

func drawMetricCard(img *image.RGBA, rect image.Rectangle, label string, value string, meta string, valueFace font.Face, metaFace font.Face) {
	fillRoundedRect(img, rect, 24, shareImageCardBackground)
	strokeRoundedRect(img, rect, 24, 2, shareImageCardBorder, shareImageCardBackground)

	drawText(img, rect.Min.X+20, rect.Min.Y+34, label, metaFace, shareImageTextMuted)
	drawText(img, rect.Min.X+20, rect.Min.Y+84, value, valueFace, shareImageTextPrimary)
	drawText(img, rect.Min.X+20, rect.Min.Y+120, meta, metaFace, shareImageTextMuted)
}

func mustFace(face font.Face, err error) font.Face {
	if err != nil {
		panic(err)
	}
	return face
}

func drawText(img *image.RGBA, x, y int, text string, face font.Face, clr color.Color) {
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(text)
}

func drawCenteredText(img *image.RGBA, rect image.Rectangle, text string, face font.Face, clr color.Color) {
	width := measureTextWidth(face, text)
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()
	x := rect.Min.X + max((rect.Dx()-width)/2, 0)
	y := rect.Min.Y + max((rect.Dy()-height)/2, 0) + metrics.Ascent.Ceil()
	drawText(img, x, y, text, face, clr)
}

func measureTextWidth(face font.Face, text string) int {
	drawer := &font.Drawer{Face: face}
	return drawer.MeasureString(text).Ceil()
}

func wrapText(face font.Face, text string, maxLines int, maxWidth int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	runes := []rune(text)
	lines := make([]string, 0, maxLines)
	current := make([]rune, 0, len(runes))

	appendLine := func(force bool) {
		if len(current) == 0 {
			return
		}
		lines = append(lines, string(current))
		current = current[:0]
		if force && len(lines) >= maxLines {
			lines[maxLines-1] = ellipsizeText(face, lines[maxLines-1], maxWidth)
		}
	}

	for _, runeValue := range runes {
		next := append(append([]rune(nil), current...), runeValue)
		if measureTextWidth(face, string(next)) <= maxWidth || len(current) == 0 {
			current = next
			continue
		}

		appendLine(false)
		if len(lines) >= maxLines {
			lines[maxLines-1] = ellipsizeText(face, lines[maxLines-1]+string(runeValue), maxWidth)
			return lines[:maxLines]
		}
		current = []rune{runeValue}
	}

	appendLine(false)
	if len(lines) == 0 {
		return []string{text}
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = ellipsizeText(face, lines[maxLines-1], maxWidth)
	}
	return lines
}

func ellipsizeText(face font.Face, text string, maxWidth int) string {
	text = strings.TrimSpace(text)
	if measureTextWidth(face, text) <= maxWidth {
		return text
	}

	runes := []rune(text)
	for len(runes) > 0 {
		candidate := strings.TrimSpace(string(runes)) + "…"
		if measureTextWidth(face, candidate) <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}

	return "…"
}

func fillRect(img *image.RGBA, rect image.Rectangle, clr color.Color) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, clr)
		}
	}
}

func drawCircle(img *image.RGBA, centerX, centerY, radius int, clr color.Color) {
	radiusSquared := radius * radius
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= radiusSquared {
				if image.Pt(x, y).In(img.Bounds()) {
					img.Set(x, y, clr)
				}
			}
		}
	}
}

func fillRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, clr color.Color) {
	if rect.Empty() {
		return
	}
	radius = min(radius, min(rect.Dx()/2, rect.Dy()/2))
	if radius <= 0 {
		fillRect(img, rect, clr)
		return
	}

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		left := rect.Min.X
		right := rect.Max.X

		if y < rect.Min.Y+radius {
			dy := float64(rect.Min.Y+radius-y) - 0.5
			offset := radius - int(math.Sqrt(math.Max(float64(radius*radius)-dy*dy, 0)))
			left += offset
			right -= offset
		} else if y >= rect.Max.Y-radius {
			dy := float64(y-(rect.Max.Y-radius)) + 0.5
			offset := radius - int(math.Sqrt(math.Max(float64(radius*radius)-dy*dy, 0)))
			left += offset
			right -= offset
		}

		for x := left; x < right; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, clr)
			}
		}
	}
}

func strokeRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, width int, clr color.Color, innerFill color.Color) {
	if width <= 0 {
		return
	}
	fillRoundedRect(img, rect, radius, clr)
	inner := image.Rect(rect.Min.X+width, rect.Min.Y+width, rect.Max.X-width, rect.Max.Y-width)
	if inner.Empty() {
		return
	}
	fillRoundedRect(img, inner, max(radius-width, 0), innerFill)
}

func rgbaHex(rgb uint32, alpha uint8) color.RGBA {
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8((rgb >> 8) & 0xFF),
		B: uint8(rgb & 0xFF),
		A: alpha,
	}
}
