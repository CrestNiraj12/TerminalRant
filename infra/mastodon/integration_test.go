package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type staticToken string

func (s staticToken) AccessToken() (string, error) { return string(s), nil }

type handlerRoundTripper struct {
	h http.Handler
}

func (rt handlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := newResponseRecorder()
	rt.h.ServeHTTP(rec, req)
	return rec.response(req), nil
}

type responseRecorder struct {
	header http.Header
	body   strings.Builder
	code   int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header), code: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header         { return r.header }
func (r *responseRecorder) Write(p []byte) (int, error) { return r.body.Write(p) }
func (r *responseRecorder) WriteHeader(statusCode int)  { r.code = statusCode }

func (r *responseRecorder) response(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: r.code,
		Header:     r.header.Clone(),
		Body:       io.NopCloser(strings.NewReader(r.body.String())),
		Request:    req,
	}
}

func newTestClient(h http.Handler) *Client {
	return &Client{
		baseURL:       "http://example.test",
		tokenProvider: staticToken("tok"),
		http:          &http.Client{Transport: handlerRoundTripper{h: h}},
	}
}

func TestTimelineService_FetchByHashtagPage_RequestShapeAndMapping(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer tok" {
			t.Fatalf("missing auth header: %q", auth)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":                "10",
			"content":           "<p>hello &lt;x&gt;</p>",
			"created_at":        time.Now().UTC().Format(time.RFC3339),
			"url":               "https://x/10",
			"favourited":        false,
			"favourites_count":  1,
			"replies_count":     2,
			"in_reply_to_id":    nil,
			"media_attachments": []any{},
			"account": map[string]any{
				"id":           "acct-1",
				"display_name": "Name",
				"acct":         "user1",
			},
		}})
	})

	client := newTestClient(h)
	svc := NewTimelineService(client, "")

	rants, err := svc.FetchByHashtagPage(context.Background(), "terminalrant", 20, "123")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if gotPath != "/api/v1/timelines/tag/terminalrant" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("limit") != "20" || gotQuery.Get("max_id") != "123" {
		t.Fatalf("unexpected query: %v", gotQuery)
	}
	if len(rants) != 1 || !strings.Contains(rants[0].Content, "<x>") {
		t.Fatalf("unexpected mapped payload: %+v", rants)
	}
}

func TestTimelineService_FetchTrendingPage_FallbackToPublic(t *testing.T) {
	var hitTrends, hitPublic int
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trends/statuses":
			hitTrends++
			_ = json.NewEncoder(w).Encode([]any{})
		case "/api/v1/timelines/public":
			hitPublic++
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "1", "content": "ok", "created_at": time.Now().UTC().Format(time.RFC3339),
				"url": "", "favourited": false, "favourites_count": 0, "replies_count": 0,
				"in_reply_to_id": nil, "media_attachments": []any{},
				"account": map[string]any{"id": "a", "display_name": "", "acct": "u"},
			}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	client := newTestClient(h)
	svc := NewTimelineService(client, "")
	_, err := svc.FetchTrendingPage(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if hitTrends == 0 || hitPublic == 0 {
		t.Fatalf("expected trends then public fallback, got trends=%d public=%d", hitTrends, hitPublic)
	}
}

func TestPostService_Post_SendsForm(t *testing.T) {
	var body string
	var ctype string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/statuses" {
			t.Fatalf("unexpected req: %s %s", r.Method, r.URL.Path)
		}
		ctype = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "10", "content": "posted", "created_at": time.Now().UTC().Format(time.RFC3339),
			"url": "", "favourited": false, "favourites_count": 0, "replies_count": 0,
			"in_reply_to_id": nil, "media_attachments": []any{},
			"account": map[string]any{"id": "a", "display_name": "", "acct": "u"},
		})
	})

	client := newTestClient(h)
	svc := NewPostService(client)
	_, err := svc.Post(context.Background(), "hello", "terminalrant")
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if !strings.Contains(ctype, "application/x-www-form-urlencoded") {
		t.Fatalf("expected form content-type, got %q", ctype)
	}
	vals, err := url.ParseQuery(body)
	if err != nil {
		t.Fatalf("bad body: %v", err)
	}
	if vals.Get("visibility") != "public" {
		t.Fatalf("expected public visibility")
	}
	if !strings.Contains(strings.ToLower(vals.Get("status")), "#terminalrant") {
		t.Fatalf("expected required hashtag in status: %q", vals.Get("status"))
	}
}

func TestAccountService_LookupFollowing_EncodesIDs(t *testing.T) {
	var gotQuery url.Values
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "a", "following": true}, {"id": "b", "following": false}})
	})

	client := newTestClient(h)
	svc := NewAccountService(client)
	res, err := svc.LookupFollowing(context.Background(), []string{"a", "b", "a"})
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if len(gotQuery["id[]"]) != 2 {
		t.Fatalf("expected unique id[] entries, got: %v", gotQuery["id[]"])
	}
	if !res["a"] || res["b"] {
		t.Fatalf("unexpected follow map: %#v", res)
	}
}

func TestClient_MethodWrappers_UseExpectedHTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{
			name: "put",
			call: func(c *Client) error {
				_, err := c.Put("/x/put", strings.NewReader("a=b"))
				return err
			},
			method: http.MethodPut,
			path:   "/x/put",
		},
		{
			name: "patch",
			call: func(c *Client) error {
				_, err := c.Patch("/x/patch", strings.NewReader("a=b"))
				return err
			},
			method: http.MethodPatch,
			path:   "/x/patch",
		},
		{
			name: "delete",
			call: func(c *Client) error {
				_, err := c.Delete("/x/delete")
				return err
			},
			method: http.MethodDelete,
			path:   "/x/delete",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			})
			client := newTestClient(h)
			if err := tc.call(client); err != nil {
				t.Fatalf("call failed: %v", err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Fatalf("unexpected req got %s %s want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
		})
	}
}

func TestTimelineService_FetchHomePage_AndFetchThread(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/timelines/home":
			if r.URL.Query().Get("max_id") != "88" {
				t.Fatalf("expected max_id in home query")
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{statusJSON("10", "acct-1", "name", "user1", "home")})
		case "/api/v1/statuses/10/context":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ancestors":   []map[string]any{statusJSON("9", "acct-2", "name2", "u2", "ancestor")},
				"descendants": []map[string]any{statusJSON("11", "acct-3", "name3", "u3", "desc")},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	client := newTestClient(h)
	svc := NewTimelineService(client, "acct-1")

	home, err := svc.FetchHomePage(context.Background(), 20, "88")
	if err != nil {
		t.Fatalf("home failed: %v", err)
	}
	if len(home) != 1 || !home[0].IsOwn {
		t.Fatalf("unexpected home mapping: %#v", home)
	}

	anc, desc, err := svc.FetchThread(context.Background(), "10")
	if err != nil {
		t.Fatalf("thread failed: %v", err)
	}
	if len(anc) != 1 || len(desc) != 1 {
		t.Fatalf("unexpected thread payload: anc=%d desc=%d", len(anc), len(desc))
	}
}

func TestPostService_EditDeleteLikeUnlikeReply_RequestShape(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/statuses/12":
			body, _ := io.ReadAll(r.Body)
			vals, _ := url.ParseQuery(string(body))
			if vals.Get("status") == "" || !strings.Contains(strings.ToLower(vals.Get("status")), "#terminalrant") {
				t.Fatalf("edit must include status with required hashtag, got %q", vals.Get("status"))
			}
			_ = json.NewEncoder(w).Encode(statusJSON("12", "acct-1", "n", "u", "edited"))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/statuses/12":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/statuses/12/favourite":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/statuses/12/unfavourite":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/statuses":
			body, _ := io.ReadAll(r.Body)
			vals, _ := url.ParseQuery(string(body))
			if vals.Get("in_reply_to_id") != "12" {
				t.Fatalf("expected in_reply_to_id for reply, got %q", vals.Get("in_reply_to_id"))
			}
			if vals.Get("visibility") != "public" {
				t.Fatalf("expected public reply visibility")
			}
			_ = json.NewEncoder(w).Encode(statusJSON("13", "acct-1", "n", "u", "reply"))
		default:
			t.Fatalf("unexpected req: %s %s", r.Method, r.URL.Path)
		}
	})
	client := newTestClient(h)
	svc := NewPostService(client)

	if _, err := svc.Edit(context.Background(), "12", "edited", "terminalrant"); err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if err := svc.Delete(context.Background(), "12"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := svc.Like(context.Background(), "12"); err != nil {
		t.Fatalf("like failed: %v", err)
	}
	if err := svc.Unlike(context.Background(), "12"); err != nil {
		t.Fatalf("unlike failed: %v", err)
	}
	if _, err := svc.Reply(context.Background(), "12", "hello", "terminalrant"); err != nil {
		t.Fatalf("reply failed: %v", err)
	}
}

func TestAccountService_Endpoints_RequestShapeAndMapping(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/verify_credentials":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "acct-me",
				"acct":            "me",
				"display_name":    "Me",
				"note":            "<p>bio &lt;safe&gt;</p>",
				"statuses_count":  3,
				"followers_count": 4,
				"following_count": 5,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/accounts/update_credentials":
			body, _ := io.ReadAll(r.Body)
			vals, _ := url.ParseQuery(string(body))
			if vals.Get("display_name") != "New Name" || vals.Get("note") != "New Bio" {
				t.Fatalf("unexpected profile update form: %q", body)
			}
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/42/follow":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/42/unfollow":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/42":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "42",
				"acct":            "u42",
				"display_name":    "User 42",
				"note":            "<p>about</p>",
				"statuses_count":  12,
				"followers_count": 20,
				"following_count": 30,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/42/statuses":
			_ = json.NewEncoder(w).Encode([]map[string]any{statusJSON("201", "42", "User 42", "u42", "post one")})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/42/block":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/blocks":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "42", "acct": "u42", "display_name": "User 42"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/42/unblock":
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected req: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
	})
	client := newTestClient(h)
	svc := NewAccountService(client)

	p, err := svc.CurrentProfile(context.Background())
	if err != nil {
		t.Fatalf("current profile failed: %v", err)
	}
	if p.ID != "acct-me" || !strings.Contains(p.Bio, "<safe>") {
		t.Fatalf("unexpected current profile: %#v", p)
	}
	id, err := svc.CurrentAccountID(context.Background())
	if err != nil || id != "acct-me" {
		t.Fatalf("current account id failed: id=%q err=%v", id, err)
	}
	if err := svc.UpdateProfile(context.Background(), "New Name", "New Bio"); err != nil {
		t.Fatalf("update profile failed: %v", err)
	}
	if err := svc.FollowUser(context.Background(), "42"); err != nil {
		t.Fatalf("follow failed: %v", err)
	}
	if err := svc.UnfollowUser(context.Background(), "42"); err != nil {
		t.Fatalf("unfollow failed: %v", err)
	}
	profile, err := svc.ProfileByID(context.Background(), "42")
	if err != nil {
		t.Fatalf("profile by id failed: %v", err)
	}
	if profile.Username != "u42" || profile.PostsCount != 12 {
		t.Fatalf("unexpected profile mapping: %#v", profile)
	}
	posts, err := svc.PostsByAccount(context.Background(), "42", 20, "199")
	if err != nil {
		t.Fatalf("posts by account failed: %v", err)
	}
	if len(posts) != 1 || posts[0].AccountID != "42" {
		t.Fatalf("unexpected posts mapping: %#v", posts)
	}
	if err := svc.BlockUser(context.Background(), "42"); err != nil {
		t.Fatalf("block failed: %v", err)
	}
	blocked, err := svc.ListBlockedUsers(context.Background(), 20)
	if err != nil {
		t.Fatalf("list blocked failed: %v", err)
	}
	if len(blocked) != 1 || blocked[0].AccountID != "42" {
		t.Fatalf("unexpected blocked mapping: %#v", blocked)
	}
	if err := svc.UnblockUser(context.Background(), "42"); err != nil {
		t.Fatalf("unblock failed: %v", err)
	}
}

func TestAPIErrorPropagation_ContainsPathAndStatus(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	})
	client := newTestClient(h)
	postSvc := NewPostService(client)
	_, err := postSvc.Edit(context.Background(), "12", "x", "terminalrant")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "/api/v1/statuses/12") || !strings.Contains(err.Error(), "422") {
		t.Fatalf("expected path and status in wrapped error, got %v", err)
	}
}

func statusJSON(id, accountID, display, acct, content string) map[string]any {
	return map[string]any{
		"id":                id,
		"content":           fmt.Sprintf("<p>%s</p>", content),
		"created_at":        time.Now().UTC().Format(time.RFC3339),
		"url":               "https://example/" + id,
		"favourited":        false,
		"favourites_count":  1,
		"replies_count":     2,
		"in_reply_to_id":    nil,
		"media_attachments": []any{},
		"account": map[string]any{
			"id":           accountID,
			"display_name": display,
			"acct":         acct,
		},
	}
}
