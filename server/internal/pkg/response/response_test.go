package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
)

func setup() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	return c, w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
}

func TestOK(t *testing.T) {
	c, w := setup()
	response.OK(c, gin.H{"foo": "bar"})
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := decode(t, w)
	if body["code"].(float64) != 0 || body["msg"] != "成功" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestOKPage(t *testing.T) {
	c, w := setup()
	response.OKPage(c, []string{"a"}, 1)
	body := decode(t, w)
	data := body["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("unexpected data: %v", data)
	}
}

func TestFail(t *testing.T) {
	c, w := setup()
	response.Fail(c, errcode.NotFound, "")
	if w.Code != 200 {
		t.Fatalf("business failure should keep 200, got %d", w.Code)
	}
	body := decode(t, w)
	if int(body["code"].(float64)) != int(errcode.NotFound) {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestAbort(t *testing.T) {
	c, w := setup()
	response.Abort(c, 401, errcode.Unauthorized, "no token")
	if w.Code != 401 {
		t.Fatalf("status %d", w.Code)
	}
}
