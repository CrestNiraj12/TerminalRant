package feed

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestRenderSelectedMediaPreviewPanel_GridModes(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media: []domain.MediaAttachment{
				{Type: "image", PreviewURL: "u1"},
				{Type: "image", PreviewURL: "u2"},
				{Type: "image", PreviewURL: "u3"},
				{Type: "image", PreviewURL: "u4"},
				{Type: "image", PreviewURL: "u5"},
			},
		},
	}}
	m.cursor = 0
	m.mediaPreview[mediaPreviewBaseKey("u1")] = "p1"
	m.mediaPreview[mediaPreviewBaseKey("u2")] = "p2"
	m.mediaPreview[mediaPreviewBaseKey("u3")] = "p3"
	m.mediaPreview[mediaPreviewBaseKey("u4")] = "p4"
	m.mediaPreview[mediaPreviewBaseKey("u5")] = "p5"

	panel := m.renderSelectedMediaPreviewPanel()
	if !strings.Contains(panel, "+1 more") {
		t.Fatalf("expected overflow indicator in panel: %q", panel)
	}
}

func TestRenderSelectedMediaPreviewPanel_SingleUsesHighResolutionPreview(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
		},
	}}
	m.cursor = 0
	m.mediaPreview[mediaPreviewBaseKey("u1")] = "LOW_PREVIEW"
	m.mediaPreview[mediaPreviewSingleKey("u1")] = "HIGH_PREVIEW"

	panel := m.renderSelectedMediaPreviewPanel()
	if !strings.Contains(panel, "HIGH_PREVIEW") {
		t.Fatalf("expected high-resolution preview in single-image panel")
	}
	if strings.Contains(panel, "LOW_PREVIEW") {
		t.Fatalf("expected base preview to be replaced when high-resolution is ready")
	}
}

func TestEnsureMediaPreviewCmd_SingleImageQueuesHighAndBase(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
		},
	}}
	m.cursor = 0

	cmd := m.ensureMediaPreviewCmd()
	if cmd == nil {
		t.Fatalf("expected preview fetch command")
	}
	if !m.mediaLoading[mediaPreviewBaseKey("u1")] {
		t.Fatalf("expected base preview request queued")
	}
	if !m.mediaLoading[mediaPreviewSingleKey("u1")] {
		t.Fatalf("expected single-image high-resolution request queued")
	}
}

func TestRenderMediaDetail_RendersItems(t *testing.T) {
	out := renderMediaDetail([]domain.MediaAttachment{
		{Type: "image", Width: 10, Height: 20, Description: "desc", URL: "https://x"},
		{Type: "video", Width: 30, Height: 40, URL: "https://y"},
	})
	if !strings.Contains(out, "Media (2)") || !strings.Contains(out, "image 10x20") || !strings.Contains(out, "video 30x40") {
		t.Fatalf("unexpected media detail rendering: %q", out)
	}
}

func TestRenderANSIThumbnail_HighResolutionSamplesMorePixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 8), B: 100, A: 255})
		}
	}

	base := renderANSIThumbnail(img, 12, 6)
	high := renderANSIThumbnail(img, 24, 12)
	basePixels := strings.Count(base, "\x1b[48;2;")
	highPixels := strings.Count(high, "\x1b[48;2;")

	if basePixels != 72 {
		t.Fatalf("expected 72 sampled pixels for base preview, got %d", basePixels)
	}
	if highPixels != 288 {
		t.Fatalf("expected 288 sampled pixels for high preview, got %d", highPixels)
	}
	if highPixels <= basePixels {
		t.Fatalf("expected high-resolution preview to sample more pixels (%d <= %d)", highPixels, basePixels)
	}
}

func TestMediaPreviewTargets_VideoUsesSourceWithFallback(t *testing.T) {
	targets := mediaPreviewTargets([]domain.MediaAttachment{
		{Type: "video", URL: "https://cdn/video.mp4", PreviewURL: "https://cdn/preview.jpg"},
		{Type: "image", URL: "https://cdn/image.jpg", PreviewURL: "https://cdn/image-preview.jpg"},
		{Type: "gifv", URL: "https://cdn/anim.gif", PreviewURL: "https://cdn/anim-preview.jpg"},
	})
	if len(targets) != 3 {
		t.Fatalf("unexpected targets count: %d", len(targets))
	}
	if targets[0].URL != "https://cdn/video.mp4" || targets[0].FallbackURL != "https://cdn/preview.jpg" || !targets[0].Animated {
		t.Fatalf("unexpected video target: %#v", targets[0])
	}
	if targets[1].URL != "https://cdn/image-preview.jpg" || targets[1].Animated {
		t.Fatalf("unexpected image target: %#v", targets[1])
	}
	if targets[2].URL != "https://cdn/anim.gif" || !targets[2].Animated {
		t.Fatalf("unexpected gifv target: %#v", targets[2])
	}
}

func TestAdvanceMediaFrames_CyclesPreview(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	key := mediaPreviewBaseKey("https://cdn/video.mp4")
	m.mediaFrames[key] = []string{"f1", "f2", "f3"}
	m.mediaFrameIndex[key] = 0
	m.mediaPreview[key] = "f1"

	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f2" {
		t.Fatalf("expected frame 2 after first tick, got %q", m.mediaPreview[key])
	}
	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f3" {
		t.Fatalf("expected frame 3 after second tick, got %q", m.mediaPreview[key])
	}
	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f1" {
		t.Fatalf("expected wrap to frame 1, got %q", m.mediaPreview[key])
	}
}
